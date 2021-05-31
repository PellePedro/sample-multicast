package halo

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	hal "github.com/drivenets/vmw_tsf/tsf-hal"
	"github.com/pellepedro/sample-multicast/pkg/pwospf"
)

const (
	// https://tools.ietf.org/html/rfc2328#section-9.5.1
	DEFAULT_HELLO_INTERAL = 20
)

var (
	helloInterval = time.Duration(3)
	lsuInterval   = time.Duration(2)
)

func init() {
	if helloIntervalStr, ok := os.LookupEnv("OSPF_HELLO_INTERVALL"); ok {
		if timeSeconds, err := strconv.Atoi(helloIntervalStr); err == nil {
			_ = timeSeconds
			//helloInterval = time.Duration(timeSeconds)
		}
	}
}

type FlowKey struct {
	Protocol  uint8
	SrcAddr   string
	DstAddr   string
	SrcPort   uint16
	DstPort   uint16
	SteerLink string
}

type HalClient struct {
	counter    int
	h          hal.DnHal
	flowKeys   map[FlowKey]hal.FlowTelemetry
	interfaces map[string]hal.InterfaceTelemetry
}

type PwosfpHandler struct {
	haloRouter *HaloRouter
	myIP       net.IP
	inCh       chan interface{}
	ospfOutCh  chan interface{}
	grpcOutCh  chan interface{}
	h          hal.DnHal
	flowKeys   map[FlowKey]*hal.FlowTelemetry
	interfaces map[string]*hal.InterfaceTelemetry
}

func NewPwospfHandler(ospfInboundCh, outboundCh chan interface{}) *PwosfpHandler {
	return &PwosfpHandler{
		haloRouter: NewHaloRouter(),
		inCh:       ospfInboundCh,
		ospfOutCh:  outboundCh,
	}
}

var mCastNetworkActive = false
var ports = make(map[string]pwospf.Port)

func (handler *PwosfpHandler) processInbounds() {
	go func() {
		for {
			select {
			case data := <-handler.inCh:
				switch message := data.(type) {
				case pwospf.Port:
					if mCastNetworkActive == false {
						// pick RouterId from first interface
						fmt.Printf("Setting own IP to [%s]", message.Ip)
						handler.myIP = message.Ip
					}
					mCastNetworkActive = true
					fmt.Printf("Received Network Active [%#v]\n", message)
					ports[message.IfName] = message
					//handler.GetHalMetrics()
				case pwospf.PWOSPF:
					senderIP := make(net.IP, 4)
					binary.BigEndian.PutUint32(senderIP, message.RouterID)
					if senderIP.String() == handler.myIP.String() {
						fmt.Println("Dropping Multicast from myself")
					} else {
						switch message.Content.(type) {
						case pwospf.LSUpdate:
							handler.handleLinkStateUpdate(message)
						case pwospf.HelloPkgV2:
							handler.handleHello(message)
						}
					}
				}
			}
		}
	}()
}

func (handler *PwosfpHandler) Start() {
	handler.processInbounds()
	go func() {
		helloTick := time.NewTicker(helloInterval * time.Second)
		lsuTick := time.NewTicker(lsuInterval * time.Second)
		for {
			select {
			case <-helloTick.C:
				if mCastNetworkActive == false {
					fmt.Println("Time to Send Hello but network not ready")
					break
				}
				handler.sendHallo()
			case <-lsuTick.C:
				if mCastNetworkActive == false {
					fmt.Println("Time to Send LSU but network not ready")
					break
				}
				// handler.GetHalMetrics()
				// handler.sendLinkStateUpdate()
			}
		}
	}()
}

func (handler *PwosfpHandler) sendHallo() {

	builder := pwospf.NewHello()
	builder.SetRouterID(IPFromNetIPToUint32(handler.myIP))

	neighbors := handler.haloRouter.GetNeighbors()
	for _, nbr := range neighbors {
		builder.AddNeighBor(nbr.rid)
	}

	ospf := builder.BuildRequest()

	fmt.Printf("=> Sending Helllo with neighbors %#v\n", neighbors)
	handler.ospfOutCh <- ospf
	_ = ospf
}

func (handler *PwosfpHandler) handleHello(req pwospf.PWOSPF) {
	senderIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(senderIP, req.RouterID)
	fmt.Printf("... => I'm Router [%s]: Received OSPF HELLO Message from Router ID [%s]\n", handler.myIP.String(), senderIP.String())

	router := handler.haloRouter

	nbr, found := router.GetNeighborByIP(req.RouterID)
	if !found {
		router.AddNeighborByIP(req.RouterID, req.Port.IfName)
		router.RecomputeRoute()
	}

	fmt.Println("=========== Topology After receiving Hello ============")
	fmt.Printf("I'm Roter %s\n", handler.myIP.String())
	nbrs := router.GetNeighbors()
	for _, nbr = range nbrs {
		fmt.Printf("Neighboring Router %s\n", IPFromUint32toString(nbr.rid))
	}
	fmt.Println("=======================================================")
	handler.sendLinkStateUpdate()
}

func (handler *PwosfpHandler) sendLinkStateUpdate() {
	builder := pwospf.LinkStateBuilder{}
	nbrs := handler.haloRouter.GetNeighbors()
	for _, nbr := range nbrs {
		fmt.Printf("LSA Neighboring Router %s\n", IPFromUint32toString(nbr.rid))
		builder.AddRouterLSA(nbr.rid, 2, 10)
	}
	_ = nbrs

	builder.SetRouterID(IPFromNetIPToUint32(handler.myIP))
	ospf := builder.BuildRequest()
	fmt.Println("=> Sending LinkStateUpdate")
	handler.ospfOutCh <- ospf
}

func (handler *PwosfpHandler) handleLinkStateUpdate(req pwospf.PWOSPF) {
	router := handler.haloRouter
	senderIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(senderIP, req.RouterID)
	fmt.Printf("... => I'm Router [%s]: Received OSPF Link State Update Message from Router [%s]\n", handler.myIP.String(), senderIP.String())

	// get LSDB entry for sender
	// if lsa seq is valid
	// forwardLSUpdate

	if lsu, ok := req.Content.(pwospf.LSUpdate); ok {
		n := int(lsu.NumOfLSAs)
		for i := 0; i < n; i++ {
			lsa := lsu.LSAs[i]
			if rlsas, rok := lsa.Content.(pwospf.RouterLSAV2); rok {
				for i, val := range rlsas.Routers {
					fmt.Printf("========> Received Rouer lsa from Router [%s] index [%d] [%#v] Metric[%d] \n", senderIP.String(), i, IPFromUint32toString(val.LinkID), val.Metric)
				}
			}
		}
	}

	modified := router.SetLSDB()
	if modified {
		router.RecomputeRoute()
	}
}

/*
func (handler *PwosfpHandler) GetHalMetrics() {

	handler.h.GetFlows(
		func(fk *hal.FlowKey, tm *hal.FlowTelemetry) error {
			key := FlowKey{
				Protocol: uint8(fk.Protocol),
				SrcAddr:  fk.SrcAddr.String(),
				DstAddr:  fk.DstAddr.String(),
				SrcPort:  fk.SrcPort,
				DstPort:  fk.DstPort,
			}
			_ = key
			handler.flowKeys[key] = tm
			return nil
		},
	)
	handler.h.GetInterfaces(
		func(ifc string, tm *hal.InterfaceTelemetry) error {
			handler.interfaces[ifc] = tm
			return nil
		},
	)
	// Send LSU

	builder := pwospf.LinkStateBuilder{}
	builder.AddRouterLSA(1, 2, 10)
	builder.AddRouterLSA(1, 4, 20)
	builder.SetRouterID(handler.myIP)
	ospf := builder.BuildRequest()
	fmt.Println("=> Sending LinkStateUpdate")
	handler.ospfOutCh <- ospf
}
*/
