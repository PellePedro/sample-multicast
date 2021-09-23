package main

import (
	"pellep.io/pkg/app"
	"pellep.io/pkg/pwospf"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	Build           = "undefined"
	Version         = "undefined"
	capturedSignals = []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT}
)

func registerSignalHandlers() <-chan struct{} {
	notifyCh := make(chan os.Signal, 2)
	stopCh := make(chan struct{})

	go func() {
		<-notifyCh
		close(stopCh)
		<-notifyCh
		os.Exit(1)
	}()

	signal.Notify(notifyCh, capturedSignals...)

	return stopCh
}

func main() {

	log.Infof("Started Build[%s] Version[%s]", Build, Version)
	var wg sync.WaitGroup
	wg.Add(1)

	inCh := make(chan interface{}, 1000)
	outCh := make(chan interface{}, 1000)
	stopCh := registerSignalHandlers()

	mc := pwospf.NewMulticastConnection(inCh, outCh, stopCh)
	go mc.DiscoverStaticInterfaces()
	go mc.DiscoverDynamicallyAttachedInterfaces()

	handler := app.NewRouter(inCh, outCh)
	handler.Start()

	wg.Wait()
}
