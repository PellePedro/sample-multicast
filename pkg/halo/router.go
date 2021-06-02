package halo

import (
	"fmt"
)

type Router struct {
	RouterId uint32
	Id       uint32
	Inf      string
	Neigbors map[uint32]Router
}

func NewRouter() *Router {
	return &Router{
		Neigbors: make(map[uint32]Router),
	}
}

func (r *Router) GetNeighborByIfName(ifName string) (Router, bool) {
	for _, nbr := range r.Neigbors {
		if nbr.Inf == ifName {
			return nbr, true
		}
	}
	return Router{}, false
}

func (r *Router) GetNeighborByIP(ip uint32) (Router, bool) {
	nbr, found := r.Neigbors[ip]
	return nbr, found
}

func (r *Router) GetNeighbors() []Router {
	result := make([]Router, 0)
	for _, nbr := range r.Neigbors {
		result = append(result, nbr)
	}
	return result
}

func (r *Router) AddNeighborByIP(ip uint32, inf string) {

	fmt.Printf("Adding Neighbour with ip [%s] and interface [%s]", IPFromUint32toString(ip), inf)
	nbr := Router{RouterId: ip, Inf: inf}
	r.Neigbors[ip] = nbr
}

func (r *Router) RecomputeRoute() {
	fmt.Printf("Recompute Route")
}

