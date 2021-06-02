package halo

import (
	"encoding/binary"
	"fmt"
	"net"
)

func IPFromUint32toString(ipUint32 uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ipUint32>>24), byte(ipUint32>>16), byte(ipUint32>>8), byte(ipUint32))
}

func IPFromUint32toNetIP(ipUint32 uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipUint32)
	return ip
}

func IPFromNetIPToUint32(ip net.IP) uint32 {
	return uint32(ip[12])<<24 | uint32(ip[13])<<16 | uint32(ip[14])<<8 | uint32(ip[15])
}

/*

// String => net.IP
ip := net.ParseIP("192.168.1.100")

// net.IP => uint32
ipU := binary.BigEndian.Uint32(ip[12:16])

// uint32 => uint32
ipstr := fmt.Sprintf("%d.%d.%d.%d", byte(ipU>>24) , byte(ipU>>16), byte(ipU>>8), byte(ipU) )

*/
