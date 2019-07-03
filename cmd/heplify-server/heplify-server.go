package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"context"

	_ "net/http/pprof"

	"github.com/koding/multiconfig"
	"github.com/games130/logp"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/games130/heplify-server-metric/config"
	"github.com/games130/heplify-server-metric/decoder"
	
	proto "github.com/games130/protoMetric"
	"github.com/micro/go-log"
	"github.com/micro/go-micro"
	_ "github.com/micro/go-plugins/broker/nats"
	"github.com/micro/go-micro/server"
)

// All methods of Sub will be executed when
// a message is received
type Sub struct{
	Chan chan *decoder.HEP
}

// Method can be of any name
func (s *Sub) Process(ctx context.Context, event *proto.Event) error {
	//log.Logf("[pubsub.1] Received event %+v with metadata %+v\n", event.GetCID(), md)
	//log.Logf("[pubsub.1] Received event %+v", event.GetFirstMethod())
	// do something with event
	hepPkt, err := decoder.DecodeHEP(event)
	s.Chan <- hepPkt
	
	return nil
}

func init() {
	var err error
	var logging logp.Logging

	c := multiconfig.New()
	cfg := new(config.HeplifyServer)
	c.MustLoad(cfg)
	config.Setting = *cfg

	if tomlExists(config.Setting.Config) {
		cf := multiconfig.NewWithPath(config.Setting.Config)
		err := cf.Load(cfg)
		if err == nil {
			config.Setting = *cfg
		} else {
			fmt.Println("Syntax error in toml config file, use flag defaults.", err)
		}
	} else {
		fmt.Println("Could not find toml config file, use flag defaults.", err)
	}

	logp.DebugSelectorsStr = &config.Setting.LogDbg
	logp.ToStderr = &config.Setting.LogStd
	logging.Level = config.Setting.LogLvl
	if config.Setting.LogSys {
		logging.ToSyslog = &config.Setting.LogSys
	} else {
		var fileRotator logp.FileRotator
		fileRotator.Path = "./"
		fileRotator.Name = "heplify-server.log"
		logging.Files = &fileRotator
	}

	err = logp.Init("heplify-server-metric", &logging)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func tomlExists(f string) bool {
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		return false
	} else if !strings.Contains(f, ".toml") {
		return false
	}
	return err == nil
}

func main() {
	if promAddr := config.Setting.PromAddr; len(promAddr) > 2 {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			err := http.ListenAndServe(promAddr, nil)
			if err != nil {
				logp.Err("%v", err)
			}
		}()
	}
	
	// create a service
	service := micro.NewService(
		micro.Name("go.micro.srv.metric"),
	)
	// parse command line
	service.Init()
	
	h:=new(Sub)
	// register subscriber
	micro.RegisterSubscriber("heplify.server.metric.1", service.Server(), h, server.SubscriberQueue("hep.metric.queue.1"))


	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
	
	m := metric.New("prometheus")
	m.Chan = h.Chan

	if err := m.Run(); err != nil {
		logp.Err("%v", err)
	}
	defer m.End()
}
