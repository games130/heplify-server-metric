package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"sync"

	_ "net/http/pprof"

	"github.com/koding/multiconfig"
	"github.com/games130/logp"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/games130/heplify-server-metric/config"
	input "github.com/games130/heplify-server-metric/server"
)

type server interface {
	Run()
	End()
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
	var servers []server
	var sigCh = make(chan os.Signal, 1)
	var wg sync.WaitGroup
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	startServer := func() {
		hep := input.NewHEPInput()
		servers = []server{hep}
		for _, srv := range servers {
			wg.Add(1)
			go func(s server) {
				defer wg.Done()
				s.Run()
			}(srv)
		}
	}
	endServer := func() {
		for _, srv := range servers {
			wg.Add(1)
			go func(s server) {
				defer wg.Done()
				s.End()
			}(srv)
		}
		wg.Wait()
	}
	
	if promAddr := config.Setting.PromAddr; len(promAddr) > 2 {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			err := http.ListenAndServe(promAddr, nil)
			if err != nil {
				logp.Err("%v", err)
			}
		}()
	}
	
	startServer()
	fmt.Println("server started")
	<-sigCh
	fmt.Println("server closed") 
	endServer()
}
