package cidr

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNetworkAddress(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.3.4/16")
	assert.Equal(t, "192.168.0.0", NetworkAddress(ipnet).String())
}

func TestBroadcastAddress(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.3.4/16")
	assert.Equal(t, "192.168.255.255", BroadcastAddress(ipnet).String())
}

func Test24(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.0/24")
	assert.Equal(t, "192.168.1.0", NetworkAddress(ipnet).String())
	assert.Equal(t, "192.168.1.255", BroadcastAddress(ipnet).String())
}

func Test32(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.1/32")
	assert.Equal(t, "192.168.1.1", NetworkAddress(ipnet).String())
	assert.Equal(t, "192.168.1.1", BroadcastAddress(ipnet).String())
}
