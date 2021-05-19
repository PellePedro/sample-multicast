package pwospf

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// OSPFType denotes what kind of OSPF type it is
type OSPFType uint8

// Potential values for OSPF.Type.
const (
	OSPFHello                   OSPFType = 1
	OSPFLinkStateUpdate         OSPFType = 4
	OSPFLinkStateAcknowledgment OSPFType = 5
)

// LSA Function Codes for LSAheader.LSType
const (
	RouterLSAtypeV2         = 0x1
	NetworkLSAtypeV2        = 0x2
	SummaryLSANetworktypeV2 = 0x3
	SummaryLSAASBRtypeV2    = 0x4
	ASExternalLSAtypeV2     = 0x5
	NSSALSAtypeV2           = 0x7
)

var LayerTypeOSPF = gopacket.RegisterLayerType(1201, gopacket.LayerTypeMetadata{Name: "PWOSPF", Decoder: gopacket.DecodeFunc(decodeOSPF)})

func init() {
	layers.IPProtocolMetadata[layers.IPProtocolOSPF] = layers.EnumMetadata{DecodeWith: gopacket.DecodeFunc(decodeOSPF), Name: "PWOSPF", LayerType: LayerTypeOSPF}
}

// String conversions for OSPFType
func (i OSPFType) String() string {
	switch i {
	case OSPFHello:
		return "Hello"
	case OSPFLinkStateUpdate:
		return "Link State Update"
	case OSPFLinkStateAcknowledgment:
		return "Link State Acknowledgment"
	default:
		return ""
	}
}

// Prefix extends IntraAreaPrefixLSA
type Prefix struct {
	PrefixLength  uint8
	PrefixOptions uint8
	Metric        uint16
	AddressPrefix []byte
}

// IntraAreaPrefixLSA is the struct from RFC 5340  A.4.10.
type IntraAreaPrefixLSA struct {
	NumOfPrefixes  uint16
	RefLSType      uint16
	RefLinkStateID uint32
	RefAdvRouter   uint32
	Prefixes       []Prefix
}

// LinkLSA is the struct from RFC 5340  A.4.9.
type LinkLSA struct {
	RtrPriority      uint8
	Options          uint32
	LinkLocalAddress []byte
	NumOfPrefixes    uint32
	Prefixes         []Prefix
}

// ASExternalLSAV2 is the struct from RFC 2328  A.4.5.
type ASExternalLSAV2 struct {
	NetworkMask       uint32
	ExternalBit       uint8
	Metric            uint32
	ForwardingAddress uint32
	ExternalRouteTag  uint32
}

// ASExternalLSA is the struct from RFC 5340  A.4.7.
type ASExternalLSA struct {
	Flags             uint8
	Metric            uint32
	PrefixLength      uint8
	PrefixOptions     uint8
	RefLSType         uint16
	AddressPrefix     []byte
	ForwardingAddress []byte
	ExternalRouteTag  uint32
	RefLinkStateID    uint32
}

// InterAreaRouterLSA is the struct from RFC 5340  A.4.6.
type InterAreaRouterLSA struct {
	Options             uint32
	Metric              uint32
	DestinationRouterID uint32
}

// InterAreaPrefixLSA is the struct from RFC 5340  A.4.5.
type InterAreaPrefixLSA struct {
	Metric        uint32
	PrefixLength  uint8
	PrefixOptions uint8
	AddressPrefix []byte
}

// NetworkLSA is the struct from RFC 5340  A.4.4.
type NetworkLSA struct {
	Options        uint32
	AttachedRouter []uint32
}

// NetworkLSAV2 is the struct from RFC 2328  A.4.3.
type NetworkLSAV2 struct {
	NetworkMask    uint32
	AttachedRouter []uint32
}

// RouterV2 extends RouterLSAV2
type RouterV2 struct {
	Type     uint8
	LinkID   uint32
	LinkData uint32
	Metric   uint16
}

// RouterLSAV2 is the struct from RFC 2328  A.4.2.
type RouterLSAV2 struct {
	Flags   uint8
	Links   uint16
	Routers []RouterV2
}

// Router extends RouterLSA
type Router struct {
	Type                uint8
	Metric              uint16
	InterfaceID         uint32
	NeighborInterfaceID uint32
	NeighborRouterID    uint32
}

// RouterLSA is the struct from RFC 5340  A.4.3.
type RouterLSA struct {
	Flags   uint8
	Options uint32
	Routers []Router
}

// LSAheader is the struct from RFC 5340  A.4.2 and RFC 2328 A.4.1.
type LSAheader struct {
	LSAge       uint16
	LSType      uint16
	LinkStateID uint32
	AdvRouter   uint32
	LSSeqNumber uint32
	LSChecksum  uint16
	Length      uint16
	LSOptions   uint8
}

// LSA links LSAheader with the structs from RFC 5340  A.4.
type LSA struct {
	LSAheader
	Content interface{}
}

// LSUpdate is the struct from RFC 5340  A.3.5.
type LSUpdate struct {
	NumOfLSAs uint32
	LSAs      []LSA
}

// LSReq is the struct from RFC 5340  A.3.4.
type LSReq struct {
	LSType    uint16
	LSID      uint32
	AdvRouter uint32
}

// DbDescPkg is the struct from RFC 5340  A.3.3.
type DbDescPkg struct {
	Options      uint32
	InterfaceMTU uint16
	Flags        uint16
	DDSeqNumber  uint32
	LSAinfo      []LSAheader
}

// HelloPkg  is the struct from RFC 5340  A.3.2.
type HelloPkg struct {
	InterfaceID              uint32
	RtrPriority              uint8
	Options                  uint32
	HelloInterval            uint16
	RouterDeadInterval       uint32
	DesignatedRouterID       uint32
	BackupDesignatedRouterID uint32
	NeighborID               []uint32
}

// HelloPkgV2 extends the HelloPkg struct with OSPFv2 information
type HelloPkgV2 struct {
	HelloPkg
	NetworkMask uint32
}

//PWOSPF extend the OSPF head with version 2 specific fields
type PWOSPF struct {
	layers.BaseLayer
	Version        uint8
	Type           OSPFType
	PacketLength   uint16
	RouterID       uint32
	AreaID         uint32
	Checksum       uint16
	AuType         uint16
	Authentication uint64
	Content        interface{}
}

// getLSAsv2 parses the LSA information from the packet for OSPFv2
func getLSAsv2(num uint32, data []byte) ([]LSA, error) {
	var lsas []LSA
	var i uint32 = 0
	var offset uint32 = 0
	for ; i < num; i++ {
		lstype := uint16(data[offset+3])
		lsalength := binary.BigEndian.Uint16(data[offset+18 : offset+20])
		content, err := extractLSAInformation(lstype, lsalength, data[offset:])
		if err != nil {
			return nil, fmt.Errorf("Could not extract Link State type.")
		}
		lsa := LSA{
			LSAheader: LSAheader{
				LSAge:       binary.BigEndian.Uint16(data[offset : offset+2]),
				LSOptions:   data[offset+2],
				LSType:      lstype,
				LinkStateID: binary.BigEndian.Uint32(data[offset+4 : offset+8]),
				AdvRouter:   binary.BigEndian.Uint32(data[offset+8 : offset+12]),
				LSSeqNumber: binary.BigEndian.Uint32(data[offset+12 : offset+16]),
				LSChecksum:  binary.BigEndian.Uint16(data[offset+16 : offset+18]),
				Length:      lsalength,
			},
			Content: content,
		}
		lsas = append(lsas, lsa)
		offset += uint32(lsalength)
	}
	return lsas, nil
}

// extractLSAInformation extracts all the LSA information
func extractLSAInformation(lstype, lsalength uint16, data []byte) (interface{}, error) {
	if lsalength < 20 {
		return nil, fmt.Errorf("Link State header length %v too short, %v required", lsalength, 20)
	}
	if len(data) < int(lsalength) {
		return nil, fmt.Errorf("Link State header length %v too short, %v required", len(data), lsalength)
	}
	var content interface{}
	switch lstype {
	case RouterLSAtypeV2:
		var routers []RouterV2
		var j uint32
		for j = 24; j < uint32(lsalength); j += 12 {
			if len(data) < int(j+12) {
				return nil, errors.New("LSAtypeV2 too small")
			}
			router := RouterV2{
				LinkID:   binary.BigEndian.Uint32(data[j : j+4]),
				LinkData: binary.BigEndian.Uint32(data[j+4 : j+8]),
				Type:     uint8(data[j+8]),
				Metric:   binary.BigEndian.Uint16(data[j+10 : j+12]),
			}
			routers = append(routers, router)
		}
		if len(data) < 24 {
			return nil, errors.New("LSAtypeV2 too small")
		}
		links := binary.BigEndian.Uint16(data[22:24])
		content = RouterLSAV2{
			Flags:   data[20],
			Links:   links,
			Routers: routers,
		}
	case NSSALSAtypeV2:
		fallthrough
	case ASExternalLSAtypeV2:
		content = ASExternalLSAV2{
			NetworkMask:       binary.BigEndian.Uint32(data[20:24]),
			ExternalBit:       data[24] & 0x80,
			Metric:            binary.BigEndian.Uint32(data[24:28]) & 0x00FFFFFF,
			ForwardingAddress: binary.BigEndian.Uint32(data[28:32]),
			ExternalRouteTag:  binary.BigEndian.Uint32(data[32:36]),
		}
	case NetworkLSAtypeV2:
		var routers []uint32
		var j uint32
		for j = 24; j < uint32(lsalength); j += 4 {
			routers = append(routers, binary.BigEndian.Uint32(data[j:j+4]))
		}
		content = NetworkLSAV2{
			NetworkMask:    binary.BigEndian.Uint32(data[20:24]),
			AttachedRouter: routers,
		}
	default:
		return nil, fmt.Errorf("Unknown Link State type.")
	}
	return content, nil
}

// DecodeFromBytes decodes the given bytes into the OSPF layer.
func (ospf *PWOSPF) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	if len(data) < 24 {
		return fmt.Errorf("Packet too smal for OSPF Version 2")
	}

	ospf.Version = uint8(data[0])
	ospf.Type = OSPFType(data[1])
	ospf.PacketLength = binary.BigEndian.Uint16(data[2:4])
	ospf.RouterID = binary.BigEndian.Uint32(data[4:8])
	ospf.AreaID = binary.BigEndian.Uint32(data[8:12])
	ospf.Checksum = binary.BigEndian.Uint16(data[12:14])
	ospf.AuType = binary.BigEndian.Uint16(data[14:16])
	ospf.Authentication = binary.BigEndian.Uint64(data[16:24])

	switch ospf.Type {
	case OSPFHello:
		var neighbors []uint32
		for i := 44; uint16(i+4) <= ospf.PacketLength; i += 4 {
			neighbors = append(neighbors, binary.BigEndian.Uint32(data[i:i+4]))
		}
		ospf.Content = HelloPkgV2{
			NetworkMask: binary.BigEndian.Uint32(data[24:28]),
			HelloPkg: HelloPkg{
				HelloInterval:            binary.BigEndian.Uint16(data[28:30]),
				Options:                  uint32(data[30]),
				RtrPriority:              uint8(data[31]),
				RouterDeadInterval:       binary.BigEndian.Uint32(data[32:36]),
				DesignatedRouterID:       binary.BigEndian.Uint32(data[36:40]),
				BackupDesignatedRouterID: binary.BigEndian.Uint32(data[40:44]),
				NeighborID:               neighbors,
			},
		}
	case OSPFLinkStateUpdate:
		num := binary.BigEndian.Uint32(data[24:28])

		lsas, err := getLSAsv2(num, data[28:])
		if err != nil {
			return fmt.Errorf("Cannot parse Link State Update packet: %v", err)
		}
		ospf.Content = LSUpdate{
			NumOfLSAs: num,
			LSAs:      lsas,
		}
	case OSPFLinkStateAcknowledgment:
		var lsas []LSAheader
		for i := 24; uint16(i+20) <= ospf.PacketLength; i += 20 {
			lsa := LSAheader{
				LSAge:       binary.BigEndian.Uint16(data[i : i+2]),
				LSOptions:   data[i+2],
				LSType:      uint16(data[i+3]),
				LinkStateID: binary.BigEndian.Uint32(data[i+4 : i+8]),
				AdvRouter:   binary.BigEndian.Uint32(data[i+8 : i+12]),
				LSSeqNumber: binary.BigEndian.Uint32(data[i+12 : i+16]),
				LSChecksum:  binary.BigEndian.Uint16(data[i+16 : i+18]),
				Length:      binary.BigEndian.Uint16(data[i+18 : i+20]),
			}
			lsas = append(lsas, lsa)
		}
		ospf.Content = lsas
	}
	return nil
}

// LayerType returns LayerTypeOSPF
func (ospf *PWOSPF) LayerType() gopacket.LayerType {
	return LayerTypeOSPF
}

// NextLayerType returns the layer type contained by this DecodingLayer.
func (ospf *PWOSPF) NextLayerType() gopacket.LayerType {
	return gopacket.LayerTypeZero
}

// CanDecode returns the set of layer types that this DecodingLayer can decode.
func (ospf *PWOSPF) CanDecode() gopacket.LayerClass {
	return LayerTypeOSPF
}

func decodeOSPF(data []byte, p gopacket.PacketBuilder) error {
	if len(data) < 14 {
		return fmt.Errorf("Packet too smal for OSPF")
	}

	switch uint8(data[0]) {
	case 2:
		ospf := &PWOSPF{}

		err := ospf.DecodeFromBytes(data, p)
		if err != nil {
			return err
		}
		p.AddLayer(ospf)
		next := ospf.NextLayerType()
		if next == gopacket.LayerTypeZero {
			return nil
		}
		return p.NextDecoder(next)
	default:
	}

	return fmt.Errorf("Unable to determine OSPF type.")
}

/*
  ---------------------- Header ----------------------------------

+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 |   Version #   |     Type      |         Packet length         |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 |                          Router ID                            |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 |                           Area ID                             |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 |           Checksum            |             AuType            |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 |                       Authentication                          |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
 |                       Authentication                          |
 +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

func (ospf *PWOSPF) SerializeTo(b gopacket.SerializeBuffer, opts gopacket.SerializeOptions) error {

	bytes, err := b.PrependBytes(int(ospf.PacketLength))
	if err != nil {
		return err
	}

	const version = 2 // IPv4
	bytes[0] = version
	bytes[1] = byte(ospf.Type)
	binary.BigEndian.PutUint16(bytes[2:4], ospf.PacketLength)
	binary.BigEndian.PutUint32(bytes[4:8], ospf.RouterID)
	binary.BigEndian.PutUint32(bytes[8:12], ospf.AreaID)
	binary.BigEndian.PutUint16(bytes[12:14], ospf.Checksum)
	binary.BigEndian.PutUint16(bytes[14:16], ospf.AuType)
	binary.BigEndian.PutUint64(bytes[16:24], ospf.Authentication)

	switch ospf.Type {
	case OSPFHello:
		/*
				Hello Packet
			   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                            Header (len 24 bytes               |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                        Network Mask                           |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|         HelloInterval         |    Options    |    Rtr Pri    |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                     RouterDeadInterval                        |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                      Designated Router                        |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                   Backup Designated Router                    |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                   Neighbour ID                                |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                           ..                                  |
		*/
		ospfHello := ospf.Content.(HelloPkgV2)
		binary.BigEndian.PutUint32(bytes[24:28], ospfHello.NetworkMask)
		binary.BigEndian.PutUint16(bytes[28:30], ospfHello.HelloInterval)
		bytes[30] = byte(ospfHello.Options)
		bytes[31] = byte(ospfHello.RtrPriority)
		binary.BigEndian.PutUint32(bytes[32:36], ospfHello.RouterDeadInterval)
		binary.BigEndian.PutUint32(bytes[36:40], ospfHello.DesignatedRouterID)
		binary.BigEndian.PutUint32(bytes[40:44], ospfHello.BackupDesignatedRouterID)
		nn := 44
		for i := range ospfHello.NeighborID {
			binary.BigEndian.PutUint32(bytes[nn:nn+4], ospfHello.NeighborID[i])
			nn += 4
		}
	case OSPFLinkStateUpdate:
		/*
			   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                  Header (len 24 bytes                        |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                        LSA's                                 |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
				|                        ....                                 |
				+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		*/

		/*                          Router-LSAs

		 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                  Header (len 24 bytes                         |   byte[0:23]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|            LS age             |     Options   |       1       |   byte[24:27]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                        Link State ID                          |   byte[28:31]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                     Advertising Router                        |   byte[32:35]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                     LS sequence number                        |   byte[36:39]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|         LS checksum           |             length            |   byte[40:43]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|    0    |V|E|B|        0      |            # links            |   byte[46:49]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                          Link ID                              |   byte[50:53]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                         Link Data                             |   byte[54:57]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|     Type      |     # TOS     |            metric             |   byte[58:53]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                              ...                              |   byte[50:53]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|      TOS      |        0      |          TOS  metric          |   byte[50:53]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                          Link ID                              |   byte[50:53]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                         Link Data                             |   byte[50:53]
		+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		|                              ...                              |
		*/

		lsUpdate := ospf.Content.(LSUpdate)

		for i, lsa := range lsUpdate.LSAs {
			switch lsaType := lsa.Content.(type) {
			case RouterLSAV2:

				_ = i
				_ = lsaType

				binary.BigEndian.PutUint16(bytes[24:26], lsa.LSAge)
				bytes[26] = lsa.LSOptions
				bytes[27] = byte(lsa.LSType)
				binary.BigEndian.PutUint32(bytes[28:32], lsa.LinkStateID)
				binary.BigEndian.PutUint32(bytes[32:36], lsa.AdvRouter)
				binary.BigEndian.PutUint32(bytes[36:40], lsa.LSSeqNumber)
				binary.BigEndian.PutUint16(bytes[40:42], lsa.LSChecksum)
				binary.BigEndian.PutUint16(bytes[42:44], lsa.Length)
			}
		}

	case OSPFLinkStateAcknowledgment:
	}
	return nil
}
