package input

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/games130/logp"
	"github.com/games130/heplify-server-metric/config"
	"github.com/games130/heplify-server-metric/metric"
	
	proto "github.com/games130/protoMetric"
	"github.com/micro/go-log"
	"github.com/micro/go-micro"
	_ "github.com/micro/go-plugins/broker/nats"
	"github.com/micro/go-micro/server"
)

type HEPInput struct {
	inputCh   chan []byte
	promCh    chan *decoder.HEP
	wg        *sync.WaitGroup
	buffer    *sync.Pool
	quitUDP   chan bool
	quit      chan bool
	usePM     bool
}

const maxPktLen = 8192

func NewHEPInput() *HEPInput {
	h := &HEPInput{
		inputCh:   make(chan []byte, 40000),
		buffer:    &sync.Pool{New: func() interface{} { return make([]byte, maxPktLen) }},
		wg:        &sync.WaitGroup{},
		quit:      make(chan bool),
		quitUDP:   make(chan bool),
	}
	if len(config.Setting.PromAddr) > 2 {
		h.usePM = true
		h.promCh = make(chan *decoder.HEP, 40000)
	}
	
	// create a service
	service := micro.NewService(
		micro.Name("go.micro.srv.metric"),
	)
	// parse command line
	service.Init()
	
	// register subscriber
	micro.RegisterSubscriber("heplify.server.metric.1", service.Server(), new(Sub), server.SubscriberQueue("hep.metric.queue.1"))


	if err := service.Run(); err != nil {
		log.Fatal(err)
	}

	return h
}

func (h *HEPInput) Run() {
	for n := 0; n < runtime.NumCPU(); n++ {
		h.wg.Add(1)
		go h.hepWorker()
	}

	logp.Info("start %s with %#v\n", config.Version, config.Setting)

	if config.Setting.HEPAddr != "" {
		go h.serveUDP(config.Setting.HEPAddr)
	}

	if h.usePM {
		m := metric.New("prometheus")
		m.Chan = h.promCh

		if err := m.Run(); err != nil {
			logp.Err("%v", err)
		}
		defer m.End()
	}
	h.wg.Wait()
}

func (h *HEPInput) End() {
	logp.Info("stopping heplify-server...")

	if config.Setting.HEPAddr != "" {
		h.quitUDP <- true
		<-h.quitUDP
	}

	h.quit <- true
	<-h.quit

	logp.Info("heplify-server has been stopped")
}

func (h *HEPInput) hepWorker() {
	lastWarn := time.Now()
	msg := h.buffer.Get().([]byte)

	for {
		h.buffer.Put(msg[:maxPktLen])
		select {
		case <-h.quit:
			h.quit <- true
			h.wg.Done()
			return
		case msg = <-h.inputCh:
			hepPkt, err := decoder.DecodeHEP(msg)
			if err != nil {
				atomic.AddUint64(&h.stats.ErrCount, 1)
				continue
			} else if hepPkt.ProtoType == 0 {
				atomic.AddUint64(&h.stats.DupCount, 1)
				continue
			}
			atomic.AddUint64(&h.stats.HEPCount, 1)

			if h.usePM {
				select {
				case h.promCh <- hepPkt:
				default:
					if time.Since(lastWarn) > 5e8 {
						logp.Warn("overflowing metric channel")
					}
					lastWarn = time.Now()
				}
			}
		}
	}
}

