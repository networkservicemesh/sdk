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
