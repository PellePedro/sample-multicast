package utils

import (
	"encoding/binary"
	"fmt"
	"net"
)

// Converts an ipv4 address from uint32 to string representation
func IPUint32toStr(ipUint32 uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ipUint32>>24), byte(ipUint32>>16), byte(ipUint32>>8), byte(ipUint32))
}

// Converts an ipv4 address from uint32 to net.IP (byte array) representation
func IPUint32toNetIP(ipUint32 uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipUint32)
	return ip
}

// Converts an ipv4 address from net.IP to uint32 representation
func IPNetIPToUint32(ip net.IP) uint32 {
	switch len(ip) {
	case 4:
		return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
	case 16:
		return uint32(ip[12])<<24 | uint32(ip[13])<<16 | uint32(ip[14])<<8 | uint32(ip[15])
	}
	return uint32(0)
}

func IPNetMASKToUint32(mask net.IPMask) uint32 {
	return uint32(mask[0])<<24 | uint32(mask[1])<<16 | uint32(mask[2])<<8 | uint32(mask[3])
}

func IPStrToUint32(ip string) uint32 {
	netip := net.ParseIP(ip)
	return IPNetIPToUint32(netip)
}

func GetNetworkCidr(subnet, netmask uint32) string {
	nw := IPUint32toNetIP(subnet)
	mask := net.IPv4Mask(byte(netmask>>24), byte(netmask>>16), byte(netmask>>8), byte(netmask))
	length, _ := mask.Size()
	networkCidr := fmt.Sprintf("%s/%d", nw.String(), length)
	return networkCidr
}

func CanRouteIP(ip uint32, cidr string) bool {
	_, iputil, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return iputil.Contains(IPUint32toNetIP(ip))
}
