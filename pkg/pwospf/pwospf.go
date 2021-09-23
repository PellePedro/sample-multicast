package pwospf

import (
	"encoding/binary"
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// OSPFType denotes what kind of OSPF type it is
type OSPFType uint8

// Potential values for OSPF.Type.
const (
	Hello OSPFType = 1
	LSA   OSPFType = 4
)

type State uint8

const (
	UP State = iota
	DOWN
	ALL string = "ALL"
)

var LayerTypePwospf = gopacket.RegisterLayerType(1201, gopacket.LayerTypeMetadata{Name: "PWOSPF", Decoder: gopacket.DecodeFunc(decodeOSPF)})

func init() {
	layers.IPProtocolMetadata[layers.IPProtocolOSPF] = layers.EnumMetadata{DecodeWith: gopacket.DecodeFunc(decodeOSPF), Name: "PWOSPF", LayerType: LayerTypePwospf}
}

// String conversions for OSPFType
func (i OSPFType) String() string {
	switch i {
	case Hello:
		return "Hello"
	case LSA:
		return "LSA"
	default:
		return ""
	}
}

type PWOspfHello struct {
	NetworkMask uint32
	HelloInt    uint16
	Padding     uint16
}

type PWOspfLsa struct {
	Subnet   uint32
	Mask     uint32
	RouterID uint32
	TxRate   uint32
}

type PWOspfLsu struct {
	Seq   uint16
	TTL   uint16
	NoLSA uint32
	LSAS  []PWOspfLsa
}

type PWOSPF struct {
	layers.BaseLayer
	LinkName       string
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
	case Hello:
		ospf.Content = PWOspfHello{
			NetworkMask: binary.BigEndian.Uint32(data[24:28]),
			HelloInt:    binary.BigEndian.Uint16(data[28:30]),
			Padding:     binary.BigEndian.Uint16(data[30:32]),
		}
	case LSA:

		numLsa := binary.BigEndian.Uint32(data[28:32])

		lsas := make([]PWOspfLsa, 0)
		for i := 32; uint16(i+16) <= ospf.PacketLength; i += 16 {
			lsa := PWOspfLsa{
				Subnet:   binary.BigEndian.Uint32(data[i : i+4]),
				Mask:     binary.BigEndian.Uint32(data[i+4 : i+8]),
				RouterID: binary.BigEndian.Uint32(data[i+8 : i+12]),
				TxRate:   binary.BigEndian.Uint32(data[i+12 : i+16]),
			}
			lsas = append(lsas, lsa)
		}

		ospf.Content = PWOspfLsu{
			Seq:   binary.BigEndian.Uint16(data[24:26]),
			TTL:   binary.BigEndian.Uint16(data[26:28]),
			NoLSA: numLsa,
			LSAS:  lsas,
		}
	}
	return nil
}

// LayerType returns LayerTypeOSPF
func (ospf *PWOSPF) LayerType() gopacket.LayerType {
	return LayerTypePwospf
}

// NextLayerType returns the layer type contained by this DecodingLayer.
func (ospf *PWOSPF) NextLayerType() gopacket.LayerType {
	return gopacket.LayerTypeZero
}

// CanDecode returns the set of layer types that this DecodingLayer can decode.
func (ospf *PWOSPF) CanDecode() gopacket.LayerClass {
	return LayerTypePwospf
}

func decodeOSPF(data []byte, p gopacket.PacketBuilder) error {
	if len(data) < 14 {
		return fmt.Errorf("Packet too small for OSPF")
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
	case Hello:
		hello := ospf.Content.(PWOspfHello)
		binary.BigEndian.PutUint32(bytes[24:28], hello.NetworkMask)
		binary.BigEndian.PutUint16(bytes[28:32], hello.HelloInt)
		binary.BigEndian.PutUint16(bytes[32:34], hello.Padding)
	case LSA:
		lsu := ospf.Content.(PWOspfLsu)
		binary.BigEndian.PutUint16(bytes[24:26], lsu.Seq)
		binary.BigEndian.PutUint16(bytes[26:28], lsu.TTL)
		binary.BigEndian.PutUint32(bytes[28:32], lsu.NoLSA)

		entry := 0
		for _, ls := range lsu.LSAS {
			binary.BigEndian.PutUint32(bytes[entry+32:entry+36], ls.Subnet)
			binary.BigEndian.PutUint32(bytes[entry+36:entry+40], ls.Mask)
			binary.BigEndian.PutUint32(bytes[entry+40:entry+44], ls.RouterID)
			binary.BigEndian.PutUint32(bytes[entry+44:entry+48], ls.TxRate)
			entry += 16
		}
	}
	return nil
}
