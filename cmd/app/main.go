package main

import (
	"fmt"
	"sync"
	_ "time"

	"github.com/pellepedro/sample-multicast/pkg/halo"
	"github.com/pellepedro/sample-multicast/pkg/pwospf"
)

const (
	grpcServerEnv  = "GRPC_SERVER"
	grpcServerPort = 50051
)

var (
	Build   = "undefined"
	Version = "undefined"
)

func main() {

	var wg sync.WaitGroup
	wg.Add(1)

	inCh := make(chan interface{}, 1000)
	outCh := make(chan interface{}, 1000)

	// ------------------------------------------------------------
	// Start OSPF Multicast Connection
	mc := pwospf.NewMulticastConnection(inCh, outCh)
	mc.StartMulticastOnStaticInterfaces()
	// ------------------------------------------------------------
	// Create & Start OSPF Handler
	ospfh := halo.NewPwospfHandler(inCh, outCh)
	ospfh.Start()

	mc.ListenForDynamicallyAttachedInterfaces()
	wg.Wait()
	fmt.Println("<--- Stopping Halo Container")
}
