package pwospf

import "net"

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

func (r *LinkStateBuilder) setRouterID(id net.IP) {
	r.routerId = id
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
