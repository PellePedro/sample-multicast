package pwospf

type HelloBuilder struct {
	RouterId      uint32
	AreaId        uint32
	InterfaceID   uint32
	NetworkMask   uint32
	HelloInterval uint16
	Padding       uint16
}

func NewHello() *HelloBuilder {
	return &HelloBuilder{}
}
func (r *HelloBuilder) SetRouterID(id uint32) {
	r.RouterId = id
}

func (r *HelloBuilder) SetAreaID(id uint32) {
	r.AreaId = id
}

func (r *HelloBuilder) SetNetworkMask(mask uint32) {
	r.NetworkMask = mask
}

func (r *HelloBuilder) SetHelloInterval(interval uint16) {
	r.HelloInterval = interval
}

func (r *HelloBuilder) SetPadding(padding uint16) {
	r.Padding = padding
}

func (r *HelloBuilder) BuildRequest() PWOSPF {

	var length uint16 = 34
	hello := PWOspfHello{}
	hello.NetworkMask = r.NetworkMask
	hello.HelloInt = r.HelloInterval
	hello.Padding = r.Padding

	ospf := PWOSPF{Type: Hello, PacketLength: length, Content: hello}
	ospf.RouterID = r.RouterId
	ospf.AreaID = r.AreaId
	ospf.PacketLength = length
	return ospf
}

type LSABuilder struct {
	RemoteRouterId uint32
	RouterId       uint32
	AreaId         uint32
	Length         uint32
	lsu            PWOspfLsu
}

func NewLsaBuilder() *LSABuilder {

	lsu := PWOspfLsu{
		LSAS: make([]PWOspfLsa, 0),
	}

	return &LSABuilder{
		lsu: lsu,
	}
}

func (r *LSABuilder) AddLsa(subnet, mask, remoteRouter, txRate uint32) {
	lsa := PWOspfLsa{
		Subnet:   subnet,
		Mask:     mask,
		RouterID: remoteRouter,
		TxRate:   txRate,
	}
	r.lsu.LSAS = append(r.lsu.LSAS, lsa)
}

func (r *LSABuilder) SetRouterID(id uint32) {
	r.RouterId = id
}

func (r *LSABuilder) SetAreaID(id uint32) {
	r.AreaId = id
}

func (r *LSABuilder) SetSeq(seq uint16) {
	r.lsu.Seq = seq
}

func (r *LSABuilder) BuildRequest() PWOSPF {

	r.lsu.NoLSA = uint32(len(r.lsu.LSAS))
	r.lsu.TTL = 1
	r.lsu.Seq = 1
	ospf := PWOSPF{Type: LSA, Content: r.lsu}
	ospf.RouterID = r.RouterId
	ospf.AreaID = r.AreaId
	ospf.Content = r.lsu
	ospf.PacketLength = uint16(24 + 8 + 16*len(r.lsu.LSAS))

	return ospf
}
