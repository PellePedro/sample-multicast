package pwospf

import "net"

type HelloBuilder struct {
	RouterId    uint32
	InterfaceID uint32
	Neighbors   []uint32
}

func NewHello() *HelloBuilder {
	return &HelloBuilder{
		Neighbors: make([]uint32,0),
	}
}
func (r *HelloBuilder) SetRouterID(id uint32) {
	r.RouterId = id
}

func (r *HelloBuilder) AddNeighBor(nbr uint32) {
	r.Neighbors = append(r.Neighbors, nbr)
}

func (r *HelloBuilder) BuildRequest() PWOSPF {

	var length uint16
	length = 44
	hello := HelloPkgV2{}

	for _, nbr := range r.Neighbors {
		hello.NeighborID = append(hello.NeighborID, nbr)
		length += 4
	}
	ospf := PWOSPF{Type: OSPFHello, PacketLength: 44, Content: hello}
	ospf.RouterID = r.RouterId
	ospf.PacketLength = length
	return ospf
}

type LinkStateBuilder struct {
	routerId net.IP
	length   int
	header   LSAheader

	links int
	flags int
	nbr   []RouterV2
}

func NewLinkstateUpdate(h LSAheader) *LinkStateBuilder {
	return &LinkStateBuilder{
		length: 20,
		header: h,
		nbr:    make([]RouterV2, 2),
	}
}

func (r *LinkStateBuilder) AddRouterLSA(linkId int, data int, metric int) {
	r.length = r.length + 12
	lsa := RouterV2{
		LinkID:   uint32(linkId),
		LinkData: uint32(data),
		Metric:   uint16(metric),
	}
	r.nbr = append(r.nbr, lsa)
}

func (r *LinkStateBuilder) SetRouterID(id net.IP) {
	r.routerId = id
}

func (r *LinkStateBuilder) SetSeq(seq uint32) {
	r.header.LSSeqNumber = seq
}

func (r *LinkStateBuilder) BuildRequest() PWOSPF {
	ospf := PWOSPF{Type: OSPFLinkStateUpdate, Content: LSUpdate{}}
	ospf.RouterID = uint32(r.routerId[12])<<24 | uint32(r.routerId[13])<<16 | uint32(r.routerId[14])<<8 | uint32(r.routerId[15])

	noRouterLSAs := len(r.nbr)
	if noRouterLSAs > 0 {
		lsupdate := LSUpdate{NumOfLSAs: 1}
		anlsa := LSA{
			LSAheader: LSAheader{
				LSType: 0x1,
			},
		}
		routerLsas := RouterLSAV2{Links: uint16(noRouterLSAs)}

		for _, routerLsa := range r.nbr {
			routerLsas.Routers = append(routerLsas.Routers, routerLsa)
		}
		anlsa.LSAheader.Length = uint16(20 + noRouterLSAs*12 + 4)
		anlsa.Content = routerLsas
		lsupdate.LSAs = append(lsupdate.LSAs, anlsa)
		ospf.Content = lsupdate
	}

	lsaLength := 20 + noRouterLSAs*12 + 4
	ospf.PacketLength = uint16(24 + 4 + lsaLength)

	return ospf
}