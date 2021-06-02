module github.com/pellepedro/sample-multicast

go 1.16

require (
	github.com/drivenets/vmw_tsf/tsf-hal v0.0.0-20210526133636-9122735a64f8
	github.com/google/gopacket v1.1.19
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5
)

replace github.com/drivenets/vmw_tsf/tsf-hal => ../vmw_tsf/tsf-hal

replace github.com/drivenets/vmw_tsf/tsf-twamp => ../vmw_tsf/tsf-twamp
