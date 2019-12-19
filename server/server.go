package input

import (
	"runtime"
	"sync"
	"sync/atomic"
	"context"
	"time"

	"github.com/games130/logp"
	"github.com/games130/heplify-server-metric/config"
	"github.com/games130/heplify-server-metric/decoder"
	"github.com/games130/heplify-server-metric/metric"
	//proto "github.com/games130/heplify-server-metric/proto"
	proto "github.com/games130/microProtocSIP"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-plugins/broker/nats"
)

type HEPInput struct {
	inCh	      chan *proto.Event
	pmCh          chan *decoder.HEP
	wg            *sync.WaitGroup
	quit          chan bool
	perMSGDebug   bool
	stats         HEPStats
}

type HEPStats struct {
	HEPCount 		uint64
	INVITECount		uint64
	REGISTERCount		uint64
	BYECount		uint64
	PRACKCount		uint64
	R180Count		uint64
	R183Count 		uint64
	R200Count 		uint64
	R400Count 		uint64
	R404Count 		uint64
	R406Count 		uint64
	R408Count 		uint64
	R416Count 		uint64
	R420Count 		uint64
	R422Count 		uint64
	R480Count 		uint64
	R481Count 		uint64
	R484Count 		uint64
	R485Count 		uint64
	R488Count 		uint64
	R500Count 		uint64
	R502Count 		uint64
	R503Count 		uint64
	R504Count 		uint64
	R603Count 		uint64
	R604Count 		uint64
	OtherCount 		uint64
}

func (h *HEPInput) subEv(ctx context.Context, event *proto.Event) error {
	//log.Logf("[pubsub.2] Received event %+v with metadata %+v\n", event, md)
	//fmt.Println("received %s and %s", event.GetCID(), event.GetFirstMethod())
	
	// do something with event
	atomic.AddUint64(&h.stats.HEPCount, 1)
	h.inCh <- event
	
	return nil
}

func NewHEPInput() *HEPInput {
	h := &HEPInput{
		inCh:      make(chan *proto.Event, 40000),
		pmCh:	   make(chan *decoder.HEP, 40000),
		wg:        &sync.WaitGroup{},
		quit:      make(chan bool),
	}
	
	h.perMSGDebug = config.Setting.PerMSGDebug

	return h
}

func (h *HEPInput) Run() {
	logp.Info("creating hepWorker totaling: %s", runtime.NumCPU()*4)
	for n := 0; n < runtime.NumCPU()*4; n++ {
		h.wg.Add(1)
		go h.hepWorker()
	}
	
	go h.logStats()
	
	b := nats.NewBroker(
		broker.Addrs(config.Setting.BrokerAddr),
	)
	
	// create a service
	service := micro.NewService(
		micro.Name("go.micro.srv.metric"),
		micro.Broker(b),
	)
	// parse command line
	service.Init()
	
	// register subscriber
	micro.RegisterSubscriber(config.Setting.BrokerTopic, service.Server(), h.subEv, server.SubscriberQueue(config.Setting.BrokerQueue))

	m := metric.New("prometheus")
	m.Chan = h.pmCh
	
	//fmt.Println("micro server before start")
	go func (){
		if err := service.Run(); err != nil {
			logp.Err("%v", err)
		}
	}()	

	//fmt.Println("metric server before start")
	if err := m.Run(); err != nil {
		logp.Err("%v", err)
	}
	defer m.End()
	h.wg.Wait()
}

func (h *HEPInput) End() {
	logp.Info("stopping heplify-server...")

	h.quit <- true
	<-h.quit

	logp.Info("heplify-server has been stopped")
}

func (h *HEPInput) hepWorker() {
	for {
		select {
		case <-h.quit:
			h.quit <- true
			h.wg.Done()
			return
		case msg := <-h.inCh:
			//fmt.Println("want to start decoding %s and %s", msg.GetCID(), msg.GetFirstMethod())
			hepPkt, _ := decoder.DecodeHEP(msg)
			
			if h.perMSGDebug {
				logp.Info("perMSGDebug: ,HEPCount,%s, SrcIP,%s, DstIP,%s, CID,%s, FirstMethod,%s, FromUser,%s, ToUser,%s", h.stats.HEPCount, hepPkt.SrcIP, hepPkt.DstIP, hepPkt.CallID, hepPkt.FirstMethod, hepPkt.FromUser, hepPkt.ToUser)
			}
			
			h.statsCount(hepPkt.FirstMethod)
			h.pmCh <- hepPkt
		}
	}
}



func (h *HEPInput) statsCount(FirstMethod string) {
	switch FirstMethod {
		case "INVITE": atomic.AddUint64(&h.stats.INVITECount, 1)
		case "REGISTER": atomic.AddUint64(&h.stats.REGISTERCount, 1)
		case "BYE": atomic.AddUint64(&h.stats.BYECount, 1)
		case "PRACK": atomic.AddUint64(&h.stats.PRACKCount, 1)
		case "180": atomic.AddUint64(&h.stats.R180Count, 1)
		case "183": atomic.AddUint64(&h.stats.R183Count, 1)
		case "200": atomic.AddUint64(&h.stats.R200Count, 1)
		case "400": atomic.AddUint64(&h.stats.R400Count, 1)
		case "404": atomic.AddUint64(&h.stats.R404Count, 1)
		case "406": atomic.AddUint64(&h.stats.R406Count, 1)
		case "408": atomic.AddUint64(&h.stats.R408Count, 1)
		case "416": atomic.AddUint64(&h.stats.R416Count, 1)
		case "420": atomic.AddUint64(&h.stats.R420Count, 1)
		case "422": atomic.AddUint64(&h.stats.R422Count, 1)
		case "480": atomic.AddUint64(&h.stats.R480Count, 1)
		case "481": atomic.AddUint64(&h.stats.R481Count, 1)
		case "484": atomic.AddUint64(&h.stats.R484Count, 1)
		case "485": atomic.AddUint64(&h.stats.R485Count, 1)
		case "488": atomic.AddUint64(&h.stats.R488Count, 1)
		case "500": atomic.AddUint64(&h.stats.R500Count, 1)
		case "502": atomic.AddUint64(&h.stats.R502Count, 1)
		case "503": atomic.AddUint64(&h.stats.R503Count, 1)
		case "504": atomic.AddUint64(&h.stats.R504Count, 1)
		case "603": atomic.AddUint64(&h.stats.R603Count, 1)
		case "604": atomic.AddUint64(&h.stats.R604Count, 1)
		default: atomic.AddUint64(&h.stats.OtherCount, 1)
	}
}



func (h *HEPInput) logStats() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			logp.Info("stats since last 5 minutes. HEP: %d, INVITECount: %d, REGISTERCount: %d, BYECount: %d, PRACKCount: %d, 180Count: %d, 183Count: %d, 200Count: %d, 400Count: %d, 404Count: %d, 406Count: %d, 408Count: %d, 416Count: %d, 420Count: %d, 422Count: %d, 480Count: %d, 481Count: %d, 484Count: %d, 485Count: %d, 488Count: %d, 500Count: %d, 502Count: %d, 503Count: %d, 504Count: %d, 603Count: %d, 604Count: %d, OtherCount: %d",
				atomic.LoadUint64(&h.stats.HEPCount),
				atomic.LoadUint64(&h.stats.INVITECount),
				atomic.LoadUint64(&h.stats.REGISTERCount),
				atomic.LoadUint64(&h.stats.BYECount),
				atomic.LoadUint64(&h.stats.PRACKCount),
				atomic.LoadUint64(&h.stats.R180Count),
				atomic.LoadUint64(&h.stats.R183Count),
				atomic.LoadUint64(&h.stats.R200Count),
				atomic.LoadUint64(&h.stats.R400Count),
				atomic.LoadUint64(&h.stats.R404Count),
				atomic.LoadUint64(&h.stats.R406Count),
				atomic.LoadUint64(&h.stats.R408Count),
				atomic.LoadUint64(&h.stats.R416Count),
				atomic.LoadUint64(&h.stats.R420Count),
				atomic.LoadUint64(&h.stats.R422Count),
				atomic.LoadUint64(&h.stats.R480Count),
				atomic.LoadUint64(&h.stats.R481Count),
				atomic.LoadUint64(&h.stats.R484Count),
				atomic.LoadUint64(&h.stats.R485Count),
				atomic.LoadUint64(&h.stats.R488Count),
				atomic.LoadUint64(&h.stats.R500Count),
				atomic.LoadUint64(&h.stats.R502Count),
				atomic.LoadUint64(&h.stats.R503Count),
				atomic.LoadUint64(&h.stats.R504Count),
				atomic.LoadUint64(&h.stats.R603Count),
				atomic.LoadUint64(&h.stats.R604Count),
				atomic.LoadUint64(&h.stats.OtherCount),
			)
			atomic.StoreUint64(&h.stats.HEPCount, 0)
			atomic.StoreUint64(&h.stats.INVITECount, 0)
			atomic.StoreUint64(&h.stats.REGISTERCount, 0)
			atomic.StoreUint64(&h.stats.BYECount, 0)
			atomic.StoreUint64(&h.stats.PRACKCount, 0)
			atomic.StoreUint64(&h.stats.R180Count, 0)
			atomic.StoreUint64(&h.stats.R183Count, 0)
			atomic.StoreUint64(&h.stats.R200Count, 0)
			atomic.StoreUint64(&h.stats.R400Count, 0)
			atomic.StoreUint64(&h.stats.R404Count, 0)
			atomic.StoreUint64(&h.stats.R406Count, 0)
			atomic.StoreUint64(&h.stats.R408Count, 0)
			atomic.StoreUint64(&h.stats.R416Count, 0)
			atomic.StoreUint64(&h.stats.R420Count, 0)
			atomic.StoreUint64(&h.stats.R422Count, 0)
			atomic.StoreUint64(&h.stats.R480Count, 0)
			atomic.StoreUint64(&h.stats.R481Count, 0)
			atomic.StoreUint64(&h.stats.R484Count, 0)
			atomic.StoreUint64(&h.stats.R485Count, 0)
			atomic.StoreUint64(&h.stats.R488Count, 0)
			atomic.StoreUint64(&h.stats.R500Count, 0)
			atomic.StoreUint64(&h.stats.R502Count, 0)
			atomic.StoreUint64(&h.stats.R503Count, 0)
			atomic.StoreUint64(&h.stats.R504Count, 0)
			atomic.StoreUint64(&h.stats.R603Count, 0)
			atomic.StoreUint64(&h.stats.R604Count, 0)
			atomic.StoreUint64(&h.stats.OtherCount, 0)

		case <-h.quit:
			h.quit <- true
			return
		}
	}
}


