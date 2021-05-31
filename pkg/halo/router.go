package halo

import (
	"fmt"

	"github.com/pellepedro/sample-multicast/pkg/pwospf"
)

type Topology struct {
}

func (t Topology) AddNode(id uint32)             {}
func (t Topology) AddLink(s, d uint32, cost int) {}

type Neighbor struct {
	rid uint32
	inf string
}

type LSA struct {
	rid     uint32
	subnet  uint32
	netmask uint32
	txRate  uint16
}

// Represent a Router with Neighbors
type LSDBEntry struct {
	aid        uint32
	rid        uint32
	seq        uint32
	csum       uint16
	lastUpdate uint16
	lsa        map[uint32]LSA // key : subnet cidr "10.10.1.1/24"
}

func NewLSDBEntry(ospf pwospf.PWOSPF) *LSDBEntry {
	//rid ospf.RouterID

	switch ospf.Type {
	case pwospf.OSPFLinkStateUpdate:
		lsUpdate := ospf.Content.(pwospf.LSUpdate)

		for i, lsa := range lsUpdate.LSAs {
			switch lsaType := lsa.Content.(type) {
			case pwospf.RouterLSAV2:
				_ = lsaType
				_ = i

				entry := LSDBEntry{
					rid:        ospf.RouterID,
					aid:        ospf.AreaID,
					seq:        lsa.LSSeqNumber,
					csum:       lsa.LSChecksum,
					lastUpdate: lsa.LSAge,
					lsa:        make(map[uint32]LSA),
				}

				for _, r := range lsaType.Routers {
					l := LSA{
						rid:    r.LinkID,
						subnet: r.LinkData,
						txRate: r.Metric,
					}
					entry.lsa[r.LinkData] = l
				}
				return &entry
			}
		}
	}
	return nil

}
func (entry LSDBEntry) populateLSA(n int, lsa map[string]string) {

	/*
		for i, val := range lsa {

		}
	*/
}

// ----------------------------------------------------
type LSDB struct {
	entries map[string]LSDBEntry
}

func NewLSDB() *LSDB {
	return &LSDB{
		entries: make(map[string]LSDBEntry),
	}
}

func (db LSDB) UpdateTopology(router HaloRouter) Topology {
	topology := Topology{}
	topology.AddNode(router.id)
	for _, nbr := range router.neigbor {
		topology.AddLink(router.router_id, nbr.rid, 1)
	}

	return topology
}

// ----------------------------------------------------

type RoutingTable struct {
}

type HaloRouter struct {
	router_id uint32
	id        uint32
	topology  Topology
	neigbor   map[uint32]Neighbor
	rt        RoutingTable
}

func NewHaloRouter() *HaloRouter {
	return &HaloRouter{
		neigbor: make(map[uint32]Neighbor),
	}
}

func (r *HaloRouter) GetNeighborByIP(ip uint32) (Neighbor, bool) {
	nbr, found := r.neigbor[ip]
	return nbr, found
}

func (r *HaloRouter) GetNeighbors() []Neighbor {
	result := make([]Neighbor, 0)
	for _, nbr := range r.neigbor {
		result = append(result, nbr)
	}
	return result
}

func (r *HaloRouter) AddNeighborByIP(ip uint32, inf string) {

	fmt.Printf("Adding Neighbour with ip [%s] and interface [%s]", IPFromUint32toString(ip), inf)
	nbr := Neighbor{rid: ip, inf: inf}
	r.neigbor[ip] = nbr
}

func (r *HaloRouter) RecomputeRoute() {
	fmt.Printf("Recompute Route")
}

func (r *HaloRouter) SetLSDB() bool {
	return true
}
