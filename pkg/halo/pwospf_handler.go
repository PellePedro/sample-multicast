package halo

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	hal "github.com/drivenets/vmw_tsf/tsf-hal"
	"github.com/pellepedro/sample-multicast/pkg/flow"
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

type LinkState struct {
	SourceId uint32
	RemoteId uint32
	IfName   string
	TxGain   uint16
}
type HalClient struct {
	counter    int
	h          hal.DnHal
	flowKeys   map[FlowKey]hal.FlowTelemetry
	interfaces map[string]hal.InterfaceTelemetry
}

type PwosfpHandler struct {
	router     *Router
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
		router:    NewRouter(),
		inCh:      ospfInboundCh,
		ospfOutCh: outboundCh,
	}
}

var mCastNetworkActive = false
var ports = make(map[string]pwospf.Port)
var linkstate = make(map[string]*LinkState)

func (handler *PwosfpHandler) processInbounds() {
	go func() {
		for {
			select {
			case data := <-handler.inCh:
				switch message := data.(type) {
				case pwospf.Port:
					if mCastNetworkActive == false {
						// pick RouterId from first interface
						fmt.Printf("Setting own IP to [%s]\n", message.Ip)
						handler.myIP = message.Ip
					}
					mCastNetworkActive = true
					fmt.Printf("Received Network Active [%#v]\n", message)
					ports[message.IfName] = message
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
				handler.sendHello()
			case <-lsuTick.C:
				if mCastNetworkActive == false {
					fmt.Println("Time to Send LSU but network not ready")
					break
				}
				fmt.Printf("=======> Link State DB\n")
				for _, v := range linkstate {
					fmt.Printf("%#v\n", v)
				}
				fmt.Printf("=======> Link State DB\n")
				handler.UpdateMetrics()
			}
		}
	}()
}

func (handler *PwosfpHandler) sendHello() {

	builder := pwospf.NewHello()
	builder.SetRouterID(IPFromNetIPToUint32(handler.myIP))

	neighbors := handler.router.GetNeighbors()
	for _, nbr := range neighbors {
		builder.AddNeighBor(nbr.RouterId)
	}

	ospf := builder.BuildRequest()

	fmt.Printf("=> Sending Helllo to neighbors %#v\n", neighbors)
	handler.ospfOutCh <- ospf
	_ = ospf
}

func (h *PwosfpHandler) handleHello(req pwospf.PWOSPF) {
	senderIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(senderIP, req.RouterID)
	fmt.Printf("... => I'm Router [%s]: Received OSPF HELLO Message from Router ID [%s]\n", h.myIP.String(), senderIP.String())

	router := h.router

	nbr, found := router.GetNeighborByIP(req.RouterID)
	if !found {
		router.AddNeighborByIP(req.RouterID, req.Port.IfName)
	}

	fmt.Println("=========== Topology After receiving Hello ============")
	fmt.Printf("I'm Roter %s\n", h.myIP.String())
	nbrs := router.GetNeighbors()
	for _, nbr = range nbrs {
		fmt.Printf("Neighboring Router %s\n", IPFromUint32toString(nbr.RouterId))
	}
	fmt.Println("=======================================================")

	key := h.makeLsaKey(req)
	fmt.Printf("Adding lsa mapping with key %s\n", key)
	newLsa := &LinkState{
		SourceId: IPFromNetIPToUint32(h.myIP),
		RemoteId: req.RouterID,
		IfName:   req.Port.IfName,
	}
	linkstate[key] = newLsa
}

func (h *PwosfpHandler) makeLsaKey(req pwospf.PWOSPF) string {
	return fmt.Sprintf("%s:%s", h.myIP.String(), IPFromUint32toString(req.RouterID))
}

func (h *PwosfpHandler) sendLinkStateUpdate() {
	for _, lsa := range linkstate {
		if lsa.TxGain > 0 && lsa.SourceId != IPFromNetIPToUint32(h.myIP) {
			builder := pwospf.LinkStateBuilder{}
			builder.AddRouterLSA(lsa.RemoteId, lsa.SourceId, lsa.TxGain)
			builder.SetRouterID(IPFromNetIPToUint32(h.myIP))
			ospf := builder.BuildRequest()
			fmt.Printf("=> Sending LinkStateUpdate to Router [%#v]\n", IPFromUint32toString(lsa.RemoteId))
			h.ospfOutCh <- ospf
		}
	}
}

func (h *PwosfpHandler) handleLinkStateUpdate(req pwospf.PWOSPF) {

	fmt.Printf("... => I'm Router [%s]: Received OSPF Link State Update Message from Router [%s]\n",
		h.myIP.String(), IPFromUint32toString(req.RouterID))

	if lsu, ok := req.Content.(pwospf.LSUpdate); ok {
		n := int(lsu.NumOfLSAs)
		for i := 0; i < n; i++ {
			lsa := lsu.LSAs[i]
			if rlsas, rok := lsa.Content.(pwospf.RouterLSAV2); rok {
				for _, val := range rlsas.Routers {
					// Check if LSA exists
					key := h.makeLsaKey(req)
					fmt.Printf("Adding lsa mapping with key %s", key)
					fmt.Printf("Values are %#v", val)
					lsaentry, found := linkstate[key]
					if !found {
						newLsa := &LinkState{
							SourceId: val.LinkID,
							RemoteId: val.LinkData,
							IfName:   req.Port.IfName,
							TxGain:   val.Metric,
						}
						linkstate[key] = newLsa
						fmt.Printf("Receiving Linkstate Update, and adding entry %#v\n", linkstate)
					}
					_ = lsaentry
				}
			}
		}
	}
}

// Simulate Metrics
func (h *PwosfpHandler) UpdateMetrics() {
	fh := flow.NewFlowHandler()
	fmt.Println("======= FEtching Data from Hal API")

	ifCh := make(chan interface{}, 100)
	doneCh := make(chan bool, 1)

	go handleTelemetry(doneCh, ifCh)
	fh.CreateHalClient()

	// Blocking Untill all data is received
	fh.GetInterfaces(doneCh, ifCh)

	testFlowKey := fh.GetFlows(doneCh, ifCh)
	if testFlowKey != nil {
		fh.Steer(testFlowKey, "halo2")
	}

}

func handleTelemetry(doneCh chan bool, dataCh chan interface{}) {
	for {
		select {
		case data := <-dataCh:
			switch message := data.(type) {
			case flow.InterfaceTelemetry:
				fmt.Printf("Receiving Interface Telemetry for interface %s\n", message.IfName)
			case flow.FlowTelemetry:
				fmt.Printf("Receiving Flow Telemetry %#v", message)
			}
		case <-doneCh:
			fmt.Printf("Receiving End of Telemetry")
			return
		}
	}
}
