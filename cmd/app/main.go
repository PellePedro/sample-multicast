package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/sirupsen/logrus"
	_ "vmware.com/tec/halo/pkg/api"
	"vmware.com/tec/halo/pkg/network"
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

func findLocalIP() (net.IP, error) {
	fp, err := os.Open("/etc/hosts")
	if err != nil {
		fmt.Printf("Could not open /etc/hosts: %s\n", err.Error())
		return nil, nil
	}

	defer fp.Close()

	rd := bufio.NewReader(fp)

	var localIP net.IP
	for {
		ln, _, err := rd.ReadLine()
		if err != nil {
			return localIP, nil
		}

		if len(ln) <= 1 || ln[0] == '#' {
			continue
		}
		fields := bytes.Fields(ln)

		// ensure that it is IPv4 address
		ip := net.ParseIP(string(fields[0]))
		if ip == nil || ip.To4() == nil {
			continue
		}
		localIP = ip
	}
}

func main() {

	fmt.Printf("Halo Controller: Version(%s) Build(%s)\n", Version, Build)

	ticker := time.NewTicker(2000 * time.Millisecond)
	termination := time.NewTimer(20 * time.Minute)

	localIP, err := findLocalIP()
	_, _ = err, localIP

	fmt.Printf("Found Local IP[%s]\n", localIP.String())

	con := network.NewPWConnection(localIP)
	if err := con.OpenBroadcastConnection(); err != nil {
		panic("Error Open Broadcast Connection")
	}

	go con.ReadConnection()

	// Gracefully Handle External Termination
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, signals...)

Loop:
	for {
		select {
		case t := <-ticker.C:
			con.WriteConnection(nil)
			fmt.Println("Sending Hello at ", t)
		case <-termination.C:
			ticker.Stop()
			break Loop
		case s := <-sigChan:
			_ = s
			ticker.Stop()
			break Loop
		}
	}
	con.CloseConnection()
	fmt.Printf("Controller Terminated")
}
