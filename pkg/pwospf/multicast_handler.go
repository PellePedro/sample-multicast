package pwospf

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/vishvananda/netlink"
	"golang.org/x/net/ipv4"
)

type State int

const (
	UP State = iota
)

type Port struct {
	State       State
	Index       int
	IfName      string
	Ip          net.IP
	Cidr        string
	NetworkCidr string
	Mac         net.HardwareAddr
}

type ConnInfo struct {
	IfName    string
	Interface *net.Interface
	RawConn   *ipv4.RawConn
}

type MulticastConnection struct {
	closeFunc func()
	r         *ipv4.RawConn
	// Packet Connections keyed on interfaces
	conInfo    map[string]ConnInfo
	packetConn map[string]*ipv4.PacketConn

	pwospfInCh  chan interface{}
	pwospfOutCh chan interface{}
	fanoutCh    map[string]chan interface{}
}

const (
	ospfPort = 89
)

var (
	AllSPFRouters = net.IPv4(224, 0, 0, 5)
	bindAddress   = net.IPv4(0, 0, 0, 0)
)

func NewMulticastConnection(inboundCh, outboundCh chan interface{}) *MulticastConnection {
	mc := &MulticastConnection{
		conInfo:     make(map[string]ConnInfo),
		packetConn:  make(map[string]*ipv4.PacketConn),
		pwospfInCh:  inboundCh,
		pwospfOutCh: outboundCh,
		fanoutCh:    make(map[string]chan interface{}),
	}

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		select {
		case <-ticker.C:
			go mc.SendOnMultiChannel()
		}
	}()
	return mc
}

func (mc MulticastConnection) SendOnMultiChannel() {
	fmt.Println("Started Multichannel Fan Out")
	for {
		select {
		case msg := <- mc.pwospfOutCh:
			for key, ch := range mc.fanoutCh {
				if ch != nil {
					fmt.Printf("Sending output on channel %s\n", key)
					data := msg
					ch <- data
				}
			}
		}
	}
}

func createPort(state State, index int, ifName string, ip net.IP, ipCIDR string, networkCIDR string, mac net.HardwareAddr) Port {
	port := Port{
		State:       state,
		Index:       index,
		IfName:      ifName,
		Ip:          ip,
		Cidr:        ipCIDR,
		NetworkCidr: networkCIDR,
		Mac:         mac,
	}
	fmt.Printf("Detected a new interface and port [%#v]\n", port)
	return port
}

func (mc *MulticastConnection) StartMulticastOnStaticInterfaces() {

	interfaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, f := range interfaces {
		if f.Flags&net.FlagBroadcast != net.FlagBroadcast {
			continue
		}
		if !strings.HasPrefix(f.Name, "halo") {
			fmt.Printf("Interface [%s] does not match prefix halo ... skipping", f.Name)
			continue
		}
		addr, _ := f.Addrs()
		switch a := addr[0].(type) {
		case *net.IPNet:
			cidr := a.String() // e.g 192.168.1.110/24
			ip, ipnet, _ := net.ParseCIDR(cidr)
			index := f.Index
			// cidr := a.String() // e.g. "192.168.1.100/24"
			// ipStr := ip.String()   // e.g. 192.168.1.100
			port := createPort(UP, index, f.Name, ip, cidr, ipnet.String(), f.HardwareAddr)
			mc.OpenMulticastConnection(port.IfName)
			mc.Start(port)
		default:
			fmt.Printf("========> ProcessStaticInterfaces() with Unexpected Addrs Type (%#t)\n", a)
		}
	}
}

func (mc *MulticastConnection) ListenForDynamicallyAttachedInterfaces() {
	fmt.Println("-----> ListenForDynamicallyAttachedInterfaces() -->")
	updateCh := make(chan netlink.AddrUpdate)
	doneCh := make(chan struct{})
	err := netlink.AddrSubscribe(updateCh, doneCh)
	if err != nil {
		panic("Failed to subscribe to netlink Addr Updates")
	}
	for {
		select {
		case update := <-updateCh:
			fmt.Printf("========> Detected Link updates [%#v] -->\n", update)
			var updatedLink netlink.Link
			if update.NewAddr {
				updatedLink, err = netlink.LinkByIndex(update.LinkIndex)
				if err == nil {
					name := updatedLink.Attrs().Name
					if !strings.HasPrefix(name, "halo") {
						fmt.Printf("Interface [%s] does not match prefix halo ... skipping", name)
						continue
					}
					cidr := update.LinkAddress.String()
					ip, ipnet, _ := net.ParseCIDR(cidr)
					index := update.LinkIndex
					port := createPort(UP, index, name, ip, cidr, ipnet.String(), updatedLink.Attrs().HardwareAddr)
					mc.OpenMulticastConnection(name)
					mc.Start(port)
				}
			}
		}
	}
}

func (mc *MulticastConnection) OpenMulticastConnection(interfaceName string) error {
	c, err := net.ListenPacket("ip4:89", "0.0.0.0") // OSPF for IPv4
	if err != nil {
		return err
	}

	r, err := ipv4.NewRawConn(c)
	if err != nil {
		return err
	}
	mc.r = r

	fmt.Printf("MulticastConnection() ... Finding interface [%s]\n", interfaceName)
	ifName, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return err
	}

	allSPFRouters := net.IPAddr{IP: AllSPFRouters}
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
	mc.conInfo[interfaceName] = ConnInfo{IfName: interfaceName, Interface: ifName, RawConn: r}

	return nil
}

func (pw *MulticastConnection) CloseConnection() {
	pw.closeFunc()
}

func (pw *MulticastConnection) mcGroupConnectionReader(port Port) {

	interfaceName := port.IfName
	con, found := pw.conInfo[interfaceName]
	if !found {
		fmt.Printf("ConnectionReader not found for interface [%s]... Dropping start request\n", interfaceName)
		return
	}

	b := make([]byte, 1500)
	for {
		iph, payload, cm, err := con.RawConn.ReadFrom(b)
		_ = cm
		if err != nil {
			fmt.Println("Error Reading from multicast socket connection")
			return
		}
		if iph.Version != ipv4.Version {
			continue
		}
		if iph.Dst.IsMulticast() {
			if !iph.Dst.Equal(AllSPFRouters) {
				continue
			}
		}

		var pwospf PWOSPF
		parser := gopacket.NewDecodingLayerParser(LayerTypeOSPF, &pwospf)
		layrs := make([]gopacket.LayerType, 0, 1)

		if err = parser.DecodeLayers(payload, &layrs); err != nil {
			fmt.Printf("Failed to Parse Package [%s] \n", err.Error())
			continue
		}
		pwospf.Port = port
		pw.pwospfInCh <- pwospf
	}
}

func (pw *MulticastConnection) mcGroupConnectionWriter(interfaceName string, outCh chan interface{}) {
	const DiffServCS6 = 0xc0
	con, found := pw.conInfo[interfaceName]
	if !found {
		fmt.Printf("ConnectionReader not found for interface [%s]... Dropping start request\n", interfaceName)
		return
	}
	for {
		fmt.Printf("About to Write [%s]\n", interfaceName)
		select {
		case data := <-outCh:
			pwospf, ok := data.(PWOSPF)
			if !ok {
				fmt.Println("Failed to Type assert PWOSP")
			}
			buf := gopacket.NewSerializeBuffer()
			opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
			err := gopacket.SerializeLayers(buf, opts, &pwospf)
			if err != nil {
				fmt.Println("Failed to serialize pwospf")
			}
			pwospfByteArray := buf.Bytes()
			iph := &ipv4.Header{}
			iph.Version = ipv4.Version
			iph.Len = ipv4.HeaderLen
			iph.TOS = DiffServCS6
			iph.TotalLen = ipv4.HeaderLen + len(pwospfByteArray)
			iph.TTL = 1
			iph.Protocol = 89
			iph.Dst = AllSPFRouters
			//fmt.Printf("Sending message on interface[%#v]\n", con.Interface)

			if err = con.RawConn.WriteTo(iph, pwospfByteArray, nil); err != nil {
				fmt.Printf("Failed to write to multicast group [%s]", err.Error())
			}
		}
	}
}

func (pw *MulticastConnection) Start(port Port) {
	pw.pwospfInCh <- port
	go pw.mcGroupConnectionReader(port)

	outCh := make(chan interface{})
	pw.fanoutCh[port.IfName] = outCh

	go pw.mcGroupConnectionWriter(port.IfName, outCh)
}
