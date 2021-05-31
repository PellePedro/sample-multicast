package halo

import (
	"fmt"
	"net"
	"testing"
)

func Test1(t *testing.T) {
	ip1 := IPfromNetIP(net.ParseIP("192.168.1.100"))
	ip1uint32 := ip1.uint32()
	ip2 := IPfromUint32(ip1uint32)
	if ip1.ip != ip2.ip {
		t.Fail()
	}
	fmt.Println(ip1.ip)
	fmt.Println(ip1.ip)
}
