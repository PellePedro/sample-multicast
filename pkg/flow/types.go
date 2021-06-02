package flow

import (
	"net"

	hal "github.com/drivenets/vmw_tsf/tsf-hal"
)

type FlowProto uint8

const (
	TCP = FlowProto(0x06)
	UDP = FlowProto(0x11)
)

type FlowKey struct {
	Protocol FlowProto
	SrcAddr  net.IP
	DstAddr  net.IP
	NextHop1 net.IP
	SrcPort  uint16
	DstPort  uint16
}

// Note interface naming needs a translation layer between NCP ports
// used by the SI and interfaces presented to the HALO container
//type string string

// Currently we don't split delay and jitter depending on traffic
// direction: egress or ingress. Instead we're using aggregated
// round-trip values depending on measurement method
type LinkTelemetry struct {
	// assumption: focused on the TX
	Delay  float64
	Jitter float64
}

type InterfaceTelemetry struct {
	IfName  string
	Speed   uint64
	RxBytes uint64
	RxBps   uint64
	TxBytes uint64
	TxBps   uint64
	Link    LinkTelemetry
}

// Note: Tx counters are currently not supported because of J2 limitations
type FlowTelemetry struct {
	Protocol hal.FlowProto
	SrcAddr  net.IP
	DstAddr  net.IP
	NextHop1 net.IP
	SrcPort  uint16
	DstPort  uint16
	//FlowKey
	// Rate
	RxRatePps uint64
	TxRatePps uint64
	RxRateBps uint64
	TxRateBps uint64

	// Total counters
	RxTotalPkts  uint64
	TxTotalPkts  uint64
	RxTotalBytes uint64
	TxTotalBytes uint64

	// Interfaces
	IngressIf string
	EgressIf  string
}

type InterfaceVisitor func(string, *InterfaceTelemetry) error
type FlowVisitor func(*FlowKey, *FlowTelemetry) error
