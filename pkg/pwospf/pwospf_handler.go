package pwospf

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/pellepedro/sample-multicast/protos"
)

const (
	// https://tools.ietf.org/html/rfc2328#section-9.5.1
	DEFAULT_HELLO_INTERAL = 20
)

var (
	helloInterval = time.Duration(5)
)

func init() {
	if helloIntervalStr, ok := os.LookupEnv("OSPF_HELLO_INTERVALL"); ok {
		if timeSeconds, err := strconv.Atoi(helloIntervalStr); err == nil {
			helloInterval = time.Duration(timeSeconds)
		}
	}
}

type PwosfpHandler struct {
	myLocalIP net.IP
	inCh      chan interface{}
	ospfOutCh chan interface{}
	grpcOutCh chan interface{}
}

func NewPwospfHandler(localip net.IP, ospfInboundCh, outboundCh, grpcOutboundCh chan interface{}) *PwosfpHandler {
	return &PwosfpHandler{
		myLocalIP: localip,
		inCh:      ospfInboundCh,
		ospfOutCh: outboundCh,
		grpcOutCh: grpcOutboundCh,
	}
}

func (pwh *PwosfpHandler) processInbounds() {
	go func() {
		for {
			select {
			case data := <-pwh.inCh:
				switch message := data.(type) {
				case PWOSPF:
					pwh.handlePwOspf(message)
				case *protos.LinkMetricsStream:
					fmt.Printf("Received GRPC Link status src[%s] dst[%s] jitter[%d] latency[%d]\n",
						message.GetSrc(), message.GetDst(), int(message.GetJitter()), int(message.GetLatency()))
				}
			}
		}
	}()
}

func (pwh *PwosfpHandler) Start() {
	pwh.processInbounds()
	go func() {
		for {
			tick := time.NewTicker(helloInterval * time.Second)
			select {
			case <-tick.C:
				pwh.sendHallo()
			}
		}
	}()
}

func (pwh *PwosfpHandler) sendHallo() {
	fmt.Printf("<----- Sending PWOSPF Hello, my ip is [%s]\n", pwh.myLocalIP.String())
	ospf := PWOSPF{
		Type:         OSPFHello,
		RouterID:     uint32(pwh.myLocalIP[12])<<24 | uint32(pwh.myLocalIP[13])<<16 | uint32(pwh.myLocalIP[14])<<8 | uint32(pwh.myLocalIP[15]),
		PacketLength: 44,
		Content:      HelloPkgV2{},
	}

	pwh.ospfOutCh <- ospf
	_ = ospf
}

func (pwh *PwosfpHandler) handlePwOspf(req PWOSPF) {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, req.RouterID)
	fmt.Printf("------> I'm Router [%s]: Received PWOSPF Message from Router [%s]\n", pwh.myLocalIP.String(), ip.String())
}
