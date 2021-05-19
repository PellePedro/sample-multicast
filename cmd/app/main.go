package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	_ "time"

	"github.com/pellepedro/sample-multicast/pkg/grpc"
	"github.com/pellepedro/sample-multicast/pkg/pwospf"
	"github.com/vishvananda/netlink"
)

const (
	grpcServerEnv  = "GRPC_SERVER"
	grpcServerPort = 50051
)

var (
	Build   = "undefined"
	Version = "undefined"
)

func generateClientId() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, nil
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

func listInterfaces() {
	l, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, f := range l {
		fmt.Println(f.Name)
	}
}

func main3() {
	updateCh := make(chan netlink.AddrUpdate)
	doneCh := make(chan struct{})
	err := netlink.AddrSubscribe(updateCh, doneCh)
	if err != nil {
	}
	for {
		select {
		case update := <-updateCh:
			// listInterfaces()
			var updatedLink netlink.Link
			if update.NewAddr {
				fmt.Printf("Detected new Link with Address [%s]]\n", update.LinkAddress.String())
				updatedLink, err = netlink.LinkByIndex(update.LinkIndex)
				if err == nil {
					name := updatedLink.Attrs().Name
					mac := updatedLink.Attrs().HardwareAddr.String()
					address := update.LinkAddress.String()
					fmt.Printf("Detected a new interface with Name [%s] MAC[%s] IP[%s]", name, mac, address)
				}
			}
		}
	}
}

func main() {
	var myIP string
	flag.StringVar(&myIP, "ip", "192.168.1.1", "Local IP")
	flag.Parse()


	fmt.Printf("My IP is [%s]\n", myIP)

	var wg sync.WaitGroup
	wg.Add(1)

	// Get grpc Server Endpoint
	grpcServerEndpoint, found := os.LookupEnv(grpcServerEnv)
	if !found {
		errorText := fmt.Sprintf("The environment Variable [ %s ] is not defined", grpcServerEnv)
		panic(errorText)
	}
	_ = grpcServerEndpoint

	clientId, err := generateClientId()
	if err != nil {
		panic("Failed to generate Client ID")
	}
	fmt.Printf("\n---> Starting Halo, Client[%s] Build[%s] Version[%s]\n", clientId, Build, Version)

	localIP := net.ParseIP(myIP)
	fmt.Printf("My IP is [%s]\n", localIP.String())

	inCh := make(chan interface{}, 1000)
	grpcOutCh := make(chan interface{}, 1000)
	pwospfOutCh := make(chan interface{}, 1000)

	// ------------------------------------------------------------
	// Start OSPF Multicast Connection
	mc := pwospf.NewMulticastConnection(localIP, inCh, pwospfOutCh)

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		if i.Name == "lo" || ! strings.HasPrefix(i.Name, "halo")   {
			continue
		}
		err = mc.OpenMulticastConnection(i.Name)
		if err != nil {
			fmt.Println(err.Error())
			panic("Failed to open OSPF multicast connection")
		}
		// ------------------------------------------------------------
		mc.Start(i.Name)

    }
	// ------------------------------------------------------------
	// Create & Start OSPF Handler
	ospfh := pwospf.NewPwospfHandler(localIP, inCh, pwospfOutCh, grpcOutCh)
	ospfh.Start()

	// ------------------------------------------------------------
	// Setup GRPC Client

	client, errorCh := grpc.NewGrpcClient(inCh, grpcOutCh)

	for client.Connect(grpcServerEndpoint, 50051) != nil {
		fmt.Println("Failed to connect to grpc server ... retrying")
		time.Sleep(10 * time.Second)
	}

	err = client.SubscribeToLinkState(clientId)
	if err != nil {
		panic(fmt.Sprintf("Failed to SubscribeToLinkState [%s]", err.Error()))
	}
	 _, _ = client, errorCh
	mc.ListenForDynamicallyAttachedInterfaces()
	wg.Wait()
	fmt.Println("<--- Stopping Halo Container")
}
