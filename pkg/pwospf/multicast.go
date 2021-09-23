package pwospf

import (
	"fmt"
	"net"
	"strings"

	"github.com/google/gopacket"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/net/ipv4"
)

const (
	ospfPort        = 89
	trafficIfPrefix = "eth"
	localIfPrefix   = "halo_local"
)

var (
	AllOSPFRouters = net.IPv4(224, 0, 0, 5)
	pwospfGroup    = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 5), Port: ospfPort}
)

type NicInfo struct {
	IfName      string
	CIDR        string
	IsLocalLan  bool
	IP          uint32
	SubNet      uint32
	NetMask     uint32
	NeighbourId uint32
	TXRate      uint32
}

type PacketConnectionInfo struct {
	NicInfo
	con *ipv4.RawConn
}

type MulticastConnection struct {
	inCh         chan interface{}
	outCh        chan interface{}
	linkNotifyCh chan *PacketConnectionInfo
	doneCh       <-chan struct{}
}

func NewMulticastConnection(inboundCh, outboundCh chan interface{}, doneCh <-chan struct{}) *MulticastConnection {
	mc := &MulticastConnection{
		inCh:         inboundCh,
		outCh:        outboundCh,
		doneCh:       doneCh,
		linkNotifyCh: make(chan *PacketConnectionInfo, 10),
	}
	go mcConnectionWriter(mc.outCh, mc.linkNotifyCh)
	return mc
}

func CreateNicInfo(name, cidr string, isLocalLan bool) NicInfo {
	log.WithFields(log.Fields{"name": name, "cidr": cidr}).Info("Create Nic Info")
	ip, ipnet, _ := net.ParseCIDR(cidr)
	log.WithFields(log.Fields{"name": name, "cidr": cidr, "ip": ip.String()}).Info("Create Nic Info")
	ip = ip.To4()
	if ip == nil {
		panic("panic in CreateNicInfo() as IP is not of type ipv4")
	}

	return NicInfo{
		IfName:     name,
		CIDR:       cidr,
		IsLocalLan: isLocalLan,
		IP:         uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3]),
		SubNet:     uint32(ipnet.IP[0])<<24 | uint32(ipnet.IP[1])<<16 | uint32(ipnet.IP[2])<<8 | uint32(ipnet.IP[3]),
		NetMask:    uint32(ipnet.Mask[0])<<24 | uint32(ipnet.Mask[1])<<16 | uint32(ipnet.Mask[2])<<8 | uint32(ipnet.Mask[3]),
	}
}

func (mc *MulticastConnection) RegisterLinkForMulticast(name, cidr string, isLocalLan bool) error {
	nicInfo := CreateNicInfo(name, cidr, isLocalLan)

	// A localLan represent an endpoint (stub network) and don't handle multicast
	// The subnet of the link is used as destination for traffic flow.

	if !nicInfo.IsLocalLan {
		con, err := Listen(name)
		if err != nil {
			log.WithFields(log.Fields{
				"error":          err.Error(),
				"interface Name": nicInfo.IfName,
			}).Error("Failed to listen on multicast socket")
			return err
		}

		ifcInfo := &PacketConnectionInfo{
			NicInfo: nicInfo,
			con:     con,
		}

		mc.ReadConnection(nicInfo.IfName, ifcInfo)
		mc.linkNotifyCh <- ifcInfo
	}
	mc.inCh <- nicInfo
	return nil
}

func (mc *MulticastConnection) DiscoverStaticInterfaces() {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Error("No network Interfaces found at startup")
		return
	}

	for _, f := range interfaces {
		if f.Name == "lo" || !strings.HasPrefix(f.Name, trafficIfPrefix) {
			continue
		}

		var isLocalLan bool
		if strings.HasPrefix(f.Name, localIfPrefix) {
			isLocalLan = true
		}

		addr, err := f.Addrs()
		if err != nil || addr == nil {
			log.Errorf("Failed to get Network Address for name [%s]", f.Name)
			continue
		}
		switch v := addr[0].(type) {
		case *net.IPNet:
			mc.RegisterLinkForMulticast(f.Name, v.String(), isLocalLan)
		}
	}
}

func (mc *MulticastConnection) DiscoverDynamicallyAttachedInterfaces() {
	// Channels for netlink
	updateCh := make(chan netlink.AddrUpdate)
	doneCh := make(chan struct{})

	err := netlink.AddrSubscribe(updateCh, doneCh)
	if err != nil {
		panic("Failed to subscribe to netlink Addr Updates")
	}
	for {
		select {
		case update := <-updateCh:
			var updatedLink netlink.Link

			updatedLink, err = netlink.LinkByIndex(update.LinkIndex)
			if err != nil {
				continue
			}
			name := updatedLink.Attrs().Name
			cidr := update.LinkAddress.String()
			ip, _, _ := net.ParseCIDR(cidr)

			if update.NewAddr {
				// Link added
				if ip.To4() != nil {
					mc.RegisterLinkForMulticast(name, cidr, false)
				} else {
					// TOTO: Implement Link Removed
				}
			}
		}
	}
}

func (mc *MulticastConnection) Close() {
}

func Listen(ifcName string) (*ipv4.RawConn, error) {
	ifi, err := net.InterfaceByName(ifcName)
	if err != nil { // get interface
		return nil, err
	}

	conn, err := net.ListenPacket("ip4:89", "0.0.0.0")
	if err != nil {
		return nil, err
	}
	r, err := ipv4.NewRawConn(conn)
	if err != nil {
		return nil, err
	}

	if err := r.SetControlMessage(ipv4.FlagDst|ipv4.FlagInterface, true); err != nil {
		return nil, err
	}

	allOSPFRouters := net.IPAddr{IP: AllOSPFRouters}
	if err := r.JoinGroup(ifi, &allOSPFRouters); err != nil {
		return nil, err
	}

	if err := r.SetMulticastLoopback(false); err != nil {
		return nil, err
	}

	if err := r.SetMulticastInterface(ifi); err != nil {
		return nil, err
	}
	return r, nil
}

func (mc *MulticastConnection) ReadConnection(ifName string, conInfo *PacketConnectionInfo) {
	go func(ifcName string, conInfo *PacketConnectionInfo) {
		b := make([]byte, 1500)
		for {
			_, payload, cm, err := conInfo.con.ReadFrom(b)

			if err != nil {
				log.WithFields(log.Fields{
					"error": err},
				).Error("Failed reading from Connection")
				return
			}

			var ospf PWOSPF
			parser := gopacket.NewDecodingLayerParser(LayerTypePwospf, &ospf)
			layrs := make([]gopacket.LayerType, 0, 1)

			log.WithFields(log.Fields{
				"LinkName":  ifName,
				"LinkIndex": cm.IfIndex,
			}).Info("Reading from Connection")

			if err = parser.DecodeLayers(payload, &layrs); err != nil {
				fmt.Printf("Failed to Parse Package [%s] \n", err.Error())
				continue
			}
			ospf.LinkName = ifcName
			mc.inCh <- ospf
		}
	}(ifName, conInfo)
}

var mcConnectionWriter = func(outCh chan interface{}, notifCh chan *PacketConnectionInfo) {
	const DiffServCS6 = 0xc0
	interfaceInfo := make(map[string]*PacketConnectionInfo)
	for {
		select {
		case nicInfo := <-notifCh:
			log.WithFields(log.Fields{"LinkName": nicInfo.IfName}).Info("Registering Link to Writer")
			interfaceInfo[nicInfo.IfName] = nicInfo
		case data := <-outCh:
			ospf, ok := data.(PWOSPF)
			if !ok {
				log.Error("Failed to Type assert PWOSP")
				continue
			}
			log.WithFields(log.Fields{"message": ospf.Type.String()}).Debug("Prepare to Write Multicast")
			for _, nicInfo := range interfaceInfo {

				if nicInfo.IsLocalLan {
					continue
				}

				ospf.AreaID = nicInfo.SubNet

				buf := gopacket.NewSerializeBuffer()
				opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
				err := gopacket.SerializeLayers(buf, opts, &ospf)
				if err != nil {
					log.Error("Failed to serialize pwospf")
					continue
				}

				pwospfByteArray := buf.Bytes()
				iph := &ipv4.Header{}
				iph.Version = ipv4.Version
				iph.Len = ipv4.HeaderLen
				iph.TOS = DiffServCS6
				iph.TotalLen = ipv4.HeaderLen + len(pwospfByteArray)
				iph.TTL = 1
				iph.Protocol = 89
				iph.Dst = AllOSPFRouters

				if err = nicInfo.con.WriteTo(iph, pwospfByteArray, nil); err != nil {
					log.Errorf("Failed to write to multicast group [%s]", err.Error())
				}
			}
		}
	}
}
