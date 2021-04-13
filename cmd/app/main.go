package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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


	ticker := time.NewTicker(2000 * time.Millisecond)
	termination := time.NewTimer(2 * time.Minute)

	//	grpcServerIP, found := os.LookupEnv(GRPC_SERVER_IP)
	//	if found {
	//		panic(fmt.Sprintf("Environment Variable %s is not defined\n", GRPC_SERVER_IP))
	//	}

	err := StartToplologyBroadcast()
	_ = err

	// Gracefully Handle External Termination
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, signals...)

	Loop:
	for {
		select {
		case t := <-ticker.C:
			fmt.Println("Tick at ", t)
		case <-termination.C:
			ticker.Stop()
			break Loop
		case s := <-sigChan:
			_ = s
			ticker.Stop()
			break Loop
		}
	}
	// Done Channel
	// stopCh := make(chan bool)
	fmt.Printf("Controller Terminated")
}
