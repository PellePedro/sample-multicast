package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	_ "github.com/sirupsen/logrus"
	_ "vmware.com/tec/halo/pkg/api"
	_ "vmware.com/tec/halo/pkg/network"
)

const (
	app              = "halo"
	GRPC_SERVER_IP   = "GRPC_SERVER_IP"
	GRPC_SERVER_PORT = 50051
)

var (
	Version = "unknown"
	Build   = "unknown"
)

var signals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGQUIT,
	syscall.SIGHUP,
	syscall.SIGSTOP,
	syscall.SIGCONT,
}

func StartToplologyBroadcast() error {
	fmt.Println("Started Network Boadcasting")
	return nil
}

func main() {

	fmt.Printf("Halo Controller: Version(%s) Build(%s)\n", Version, Build)

	var wg sync.WaitGroup
	wg.Add(1)

	//	grpcServerIP, found := os.LookupEnv(GRPC_SERVER_IP)
	//	if found {
	//		panic(fmt.Sprintf("Environment Variable %s is not defined\n", GRPC_SERVER_IP))
	//	}

	err := StartToplologyBroadcast()
	_ = err

	// Gracefully Handle External Termination
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, signals...)

	for s := range sigChan {
		switch s {
		case os.Interrupt:
			fallthrough
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGKILL:
			wg.Done()
		}
		fmt.Printf("Controller Terminated")
		break
	}

	// Done Channel
	// stopCh := make(chan bool)
	wg.Wait()

}
