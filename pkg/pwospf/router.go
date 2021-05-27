package pwospf

type Neighbor struct {
	inf string
}

type LSDB struct {
}

type RoutingTable struct {
}

type HaloRouter struct {
	neigbor map[string]*Neighbor
	rt      RoutingTable
}

func NewHaloRouter() *HaloRouter {
	return &HaloRouter{
		neigbor: make(map[string]*Neighbor),
	}
}

func (r *HaloRouter) GetNeighborByIP(ip string) (*Neighbor, bool) {
	nbr, found := r.neigbor[ip]
	return nbr, found
}

func (r *HaloRouter) AddNeighborByIP(inf, ip string) {
	nbr := &Neighbor{inf: inf}
	r.neigbor[ip] = nbr
}
