package input

import (
	"runtime"
	"sync"
	"context"
	"fmt"

	"github.com/games130/logp"
	"github.com/games130/heplify-server-metric/config"
	"github.com/games130/heplify-server-metric/decoder"
	"github.com/games130/heplify-server-metric/metric"
	proto "github.com/games130/heplify-server-metric/proto"

	"github.com/micro/go-micro"
	"github.com/micro/go-micro/broker"
	"github.com/micro/go-micro/server"
	"github.com/micro/go-plugins/broker/nats"
)

type HEPInput struct {
	inCh	  chan *proto.Event
	pmCh      chan *decoder.HEP
	wg        *sync.WaitGroup
	quit      chan bool
}

func (h *HEPInput) subEv(ctx context.Context, event *proto.Event) error {
	//log.Logf("[pubsub.2] Received event %+v with metadata %+v\n", event, md)
	fmt.Println("received %s and %s", event.GetCID(), event.GetFirstMethod())
	
	// do something with event
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

	return h
}

func (h *HEPInput) Run() {
	for n := 0; n < runtime.NumCPU()*4; n++ {
		h.wg.Add(1)
		go h.hepWorker()
	}
	
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
	
	fmt.Println("micro server before start")
	go func (){
		if err := service.Run(); err != nil {
			logp.Err("%v", err)
		}
	}()	

	fmt.Println("metric server before start")
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
			fmt.Println("want to start decoding %s and %s", msg.GetCID(), msg.GetFirstMethod())
			hepPkt, _ := decoder.DecodeHEP(msg)
			h.pmCh <- hepPkt
		}
	}
}


