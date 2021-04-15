package network

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"

	"golang.org/x/net/ipv4"
)

type OSPFHeader struct {
	Version  byte
	Type     byte
	Len      uint16
	RouterID uint32
	AreaID   uint32
	Checksum uint16
}

const (
	OSPFHeaderLen      = 14
	OSPFHelloHeaderLen = 20
	OSPF_VERSION       = 2
	OSPF_TYPE_HELLO    = iota + 1
	OSPF_TYPE_DB_DESCRIPTION
	OSPF_TYPE_LS_REQUEST
	OSPF_TYPE_LS_UPDATE
	OSPF_TYPE_LS_ACK
)

var (
	AllSPFRouters = net.IPv4(224, 0, 0, 5)
	AllDRouters   = net.IPv4(224, 0, 0, 6)
)

type IPv4Flag uint8

const (
	IPv4EvilBit       IPv4Flag = 1 << 2 // http://tools.ietf.org/html/rfc3514 ;)
	IPv4DontFragment  IPv4Flag = 1 << 1
	IPv4MoreFragments IPv4Flag = 1 << 0
)

const (
	Version   = 4  // protocol version
	HeaderLen = 20 // header length without extension headers
)

type PWConnection struct {
	myLocalIP net.IP
	closeFunc func()
	r         *ipv4.RawConn
}

func NewPWConnection(localip net.IP) *PWConnection {
	fmt.Printf("NewPWConnection with ip %s\n",localip.String())
	return &PWConnection{
		myLocalIP: localip,
	}
}

func (pw *PWConnection) OpenBroadcastConnection() error {

	var interfaceName string
	var found bool
	if interfaceName, found = os.LookupEnv("CONTAINER_INTERFACE"); !found {
		interfaceName = "eth0"
	}

	c, err := net.ListenPacket("ip4:89", "0.0.0.0") // OSPF for IPv4
	if err != nil {
		return err
	}

	r, err := ipv4.NewRawConn(c)
	if err != nil {
		return err
	}
	pw.r = r

	ifName, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return err
	}

	// RFC 2328
	allSPFRouters := net.IPAddr{IP: net.IPv4(224, 0, 0, 5)}
	if err := r.JoinGroup(ifName, &allSPFRouters); err != nil {
		return err
	}

	err = r.SetControlMessage(ipv4.FlagDst|ipv4.FlagInterface, true)
	if err != nil {
		return err
	}
	err = r.SetMulticastInterface(ifName)
	if err != nil {
		return err
	}
	pw.closeFunc = func() {
		fmt.Println("Executing PWConnection Closure Func")
		r.LeaveGroup(ifName, &allSPFRouters)
		c.Close()
	}
	return nil
}

func (pw *PWConnection) CloseConnection() {
	pw.closeFunc()
}

func (pw *PWConnection) WriteConnection(b []byte) error {
	hello := make([]byte, OSPFHelloHeaderLen)
	ospf := make([]byte, OSPFHeaderLen)
	ospf[0] = OSPF_VERSION
	ospf[1] = OSPF_TYPE_HELLO
	myip := ip2int(pw.myLocalIP)
	binary.BigEndian.PutUint32(ospf[4:8], myip)
	ospf = append(ospf, hello...)
	iph := &ipv4.Header{}
	iph.Version = ipv4.Version
	iph.Len = ipv4.HeaderLen
	iph.TOS = DiffServCS6
	iph.TotalLen = ipv4.HeaderLen + len(ospf)
	iph.TTL = 1
	iph.Protocol = 89
	iph.Dst = AllSPFRouters

	err := pw.r.WriteTo(iph, ospf, nil)
	if err != nil {
		fmt.Printf("Failed to write Hello, %s\n", err)
	}

	return nil
}
func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func int2ip(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

func (pw *PWConnection) ReadConnection() error {

	parseOSPFHeader := func(b []byte) *OSPFHeader {
		if len(b) < OSPFHeaderLen {
			return nil
		}
		return &OSPFHeader{
			Version:  b[0],
			Type:     b[1],
			Len:      uint16(b[2])<<8 | uint16(b[3]),
			RouterID: uint32(b[4])<<24 | uint32(b[5])<<16 | uint32(b[6])<<8 | uint32(b[7]),
			AreaID:   uint32(b[8])<<24 | uint32(b[9])<<16 | uint32(b[10])<<8 | uint32(b[11]),
			Checksum: uint16(b[12])<<8 | uint16(b[13]),
		}
	}

	b := make([]byte, 1500)
	for {
		iph, p, _, err := pw.r.ReadFrom(b)
		if err != nil {
			fmt.Println("Error Reading from connection")
			return err
		}
		if iph.Version != ipv4.Version {
			continue
		}
		if iph.Dst.IsMulticast() {
			if !iph.Dst.Equal(AllSPFRouters) && !iph.Dst.Equal(AllDRouters) {
				continue
			}
		}
		ospfh := parseOSPFHeader(p)
		if ospfh == nil {
			continue
		}
		if ospfh.Version != OSPF_VERSION {
			continue
		}

		if ospfh.RouterID == ip2int(pw.myLocalIP) {
			// Drop messages from ourself
			continue
		}

		switch ospfh.Type {
		case OSPF_TYPE_HELLO:
			remoteIP := int2ip(ospfh.RouterID)
			fmt.Printf("Received OSPF Hello from remote Router[%s], My Local IP is [%s]", remoteIP.String(), pw.myLocalIP.String())
		case OSPF_TYPE_DB_DESCRIPTION:
		case OSPF_TYPE_LS_REQUEST:
		case OSPF_TYPE_LS_UPDATE:
		case OSPF_TYPE_LS_ACK:
		}
	}

}

// https://github.com/mindscratch/gobrew_fw/blob/789e0974992b2a746af8984df16beb2c5690948b/src/code.google.com/p/go.net/ipv4/example_test.go#L153
func ExampleIPOSPFListener() {
	var ifs []*net.Interface
	en0, err := net.InterfaceByName("en0")
	if err != nil {
		log.Fatal(err)
	}
	ifs = append(ifs, en0)
	en1, err := net.InterfaceByIndex(911)
	if err != nil {
		log.Fatal(err)
	}
	ifs = append(ifs, en1)

	c, err := net.ListenPacket("ip4:89", "0.0.0.0") // OSFP for IPv4
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	r, err := ipv4.NewRawConn(c)
	if err != nil {
		log.Fatal(err)
	}
	for _, ifi := range ifs {
		err := r.JoinGroup(ifi, &net.IPAddr{IP: AllSPFRouters})
		if err != nil {
			log.Fatal(err)
		}
		err = r.JoinGroup(ifi, &net.IPAddr{IP: AllDRouters})
		if err != nil {
			log.Fatal(err)
		}
	}

	err = r.SetControlMessage(ipv4.FlagDst|ipv4.FlagInterface, true)
	if err != nil {
		log.Fatal(err)
	}
	//r.SetTOS(DiffServCS6)

	parseOSPFHeader := func(b []byte) *OSPFHeader {
		if len(b) < OSPFHeaderLen {
			return nil
		}
		return &OSPFHeader{
			Version:  b[0],
			Type:     b[1],
			Len:      uint16(b[2])<<8 | uint16(b[3]),
			RouterID: uint32(b[4])<<24 | uint32(b[5])<<16 | uint32(b[6])<<8 | uint32(b[7]),
			AreaID:   uint32(b[8])<<24 | uint32(b[9])<<16 | uint32(b[10])<<8 | uint32(b[11]),
			Checksum: uint16(b[12])<<8 | uint16(b[13]),
		}
	}

	b := make([]byte, 1500)
	for {
		iph, p, _, err := r.ReadFrom(b)
		if err != nil {
			log.Fatal(err)
		}
		if iph.Version != ipv4.Version {
			continue
		}
		if iph.Dst.IsMulticast() {
			if !iph.Dst.Equal(AllSPFRouters) && !iph.Dst.Equal(AllDRouters) {
				continue
			}
		}
		ospfh := parseOSPFHeader(p)
		if ospfh == nil {
			continue
		}
		if ospfh.Version != OSPF_VERSION {
			continue
		}
		switch ospfh.Type {
		case OSPF_TYPE_HELLO:
		case OSPF_TYPE_DB_DESCRIPTION:
		case OSPF_TYPE_LS_REQUEST:
		case OSPF_TYPE_LS_UPDATE:
		case OSPF_TYPE_LS_ACK:
		}
	}
}

func ExampleWriteIPOSPFHello() {
	var ifs []*net.Interface
	en0, err := net.InterfaceByName("en0")
	if err != nil {
		log.Fatal(err)
	}
	ifs = append(ifs, en0)
	en1, err := net.InterfaceByIndex(911)
	if err != nil {
		log.Fatal(err)
	}
	ifs = append(ifs, en1)

	c, err := net.ListenPacket("ip4:89", "0.0.0.0") // OSPF for IPv4
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	r, err := ipv4.NewRawConn(c)
	if err != nil {
		log.Fatal(err)
	}
	for _, ifi := range ifs {
		err := r.JoinGroup(ifi, &net.IPAddr{IP: AllSPFRouters})
		if err != nil {
			log.Fatal(err)
		}
		err = r.JoinGroup(ifi, &net.IPAddr{IP: AllDRouters})
		if err != nil {
			log.Fatal(err)
		}
	}

	hello := make([]byte, OSPFHelloHeaderLen)
	ospf := make([]byte, OSPFHeaderLen)
	ospf[0] = OSPF_VERSION
	ospf[1] = OSPF_TYPE_HELLO
	ospf = append(ospf, hello...)
	iph := &ipv4.Header{}
	iph.Version = ipv4.Version
	iph.Len = ipv4.HeaderLen
	//iph.TOS = DiffServCS6
	iph.TotalLen = ipv4.HeaderLen + len(ospf)
	iph.TTL = 1
	iph.Protocol = 89
	iph.Dst = AllSPFRouters

	for _, ifi := range ifs {
		err := r.SetMulticastInterface(ifi)
		if err != nil {
			return
		}
		err = r.WriteTo(iph, ospf, nil)
		if err != nil {
			return
		}
	}
}
