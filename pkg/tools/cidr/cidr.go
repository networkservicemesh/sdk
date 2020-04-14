// Package cidr provides common functions useful when working with Classless Inter-Domain Routing (CIDR)
package cidr

import (
	"encoding/binary"
	"net"
)

// NetworkAddress teturns the first IP address of an IP network
func NetworkAddress(ipNet *net.IPNet) net.IP {
	prefixNetwork := ipNet.IP.Mask(ipNet.Mask)
	return prefixNetwork
}

// BroadcastAddress returns the last IP address of an IP network
func BroadcastAddress(ipNet *net.IPNet) net.IP {
	first := NetworkAddress(ipNet)
	ones, _ := ipNet.Mask.Size()
	var shift uint32 = 1
	shift <<= ones
	intip := binary.BigEndian.Uint32(first.To4())
	intip = intip + shift - 1
	last := make(net.IP, 4)
	binary.BigEndian.PutUint32(last, intip)
	return last
}
