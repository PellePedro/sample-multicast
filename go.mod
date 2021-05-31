module github.com/pellepedro/sample-multicast

go 1.16

require (
	github.com/drivenets/vmw_tsf/tsf-hal v0.0.0-20210526133636-9122735a64f8
	github.com/google/gopacket v1.1.19
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/net v0.0.0-20210330075724-22f4162a9025
	google.golang.org/grpc v1.38.0
	google.golang.org/protobuf v1.25.0
)

//replace github.com/drivenets/vmw_tsf/tsf-hal => github.com/drivenets/vmw_tsf/tsf-hal v0.0.0-20210526133636-9122735a64f8
