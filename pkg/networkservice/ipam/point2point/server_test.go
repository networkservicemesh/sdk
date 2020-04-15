package point2point

import (
	"context"
	"net"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
)

func newRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "id",
			NetworkService: "ns",
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{},
			},
		},
		MechanismPreferences: nil,
	}
}

func TestServer(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.3.4/16")
	srv, err := NewServer([]*net.IPNet{ipnet})
	assert.NoError(t, err)

	conn1, err := srv.Request(context.Background(), newRequest())
	assert.NoError(t, err)

	assert.Equal(t, "192.168.0.0/32", conn1.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.0.1/32", conn1.Context.IpContext.SrcIpAddr)

	conn2, err := srv.Request(context.Background(), newRequest())
	assert.NoError(t, err)

	assert.Equal(t, "192.168.0.2/32", conn2.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.0.3/32", conn2.Context.IpContext.SrcIpAddr)

	_, err = srv.Close(context.Background(), conn1)
	assert.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	assert.NoError(t, err)

	assert.Equal(t, "192.168.0.0/32", conn3.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.0.1/32", conn3.Context.IpContext.SrcIpAddr)

	conn4, err := srv.Request(context.Background(), newRequest())
	assert.NoError(t, err)

	assert.Equal(t, "192.168.0.4/32", conn4.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.0.5/32", conn4.Context.IpContext.SrcIpAddr)
}

func TestNilPrefixes(t *testing.T) {
	srv, err := NewServer(nil)
	assert.Nil(t, srv)
	assert.Error(t, err)

	_, cidr1, _ := net.ParseCIDR("192.168.0.1/32")

	srv, err = NewServer([]*net.IPNet{
		cidr1,
		nil,
	})
	assert.Nil(t, srv)
	assert.Error(t, err)
}

func TestExclude32Prefix(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.0/24")
	srv, err := NewServer([]*net.IPNet{ipnet})
	assert.NotNil(t, srv)
	assert.NoError(t, err)

	// Test center of assigned
	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn1, err := srv.Request(context.Background(), req1)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.0/32", conn1.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.1.2/32", conn1.Context.IpContext.SrcIpAddr)

	// Test exclude before assigned
	req2 := newRequest()
	req2.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn2, err := srv.Request(context.Background(), req2)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.4/32", conn2.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.1.5/32", conn2.Context.IpContext.SrcIpAddr)

	// Test after assigned
	req3 := newRequest()
	req3.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn3, err := srv.Request(context.Background(), req3)
	assert.NoError(t, err)
	assert.Equal(t, "192.168.1.6/32", conn3.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.1.7/32", conn3.Context.IpContext.SrcIpAddr)
}

func TestOutOfIPs(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.2/31")
	srv, err := NewServer([]*net.IPNet{ipnet})
	assert.NotNil(t, srv)
	assert.NoError(t, err)

	req1 := newRequest()
	conn1, err := srv.Request(context.Background(), req1)
	assert.Equal(t, "192.168.1.2/32", conn1.Context.IpContext.DstIpAddr)
	assert.Equal(t, "192.168.1.3/32", conn1.Context.IpContext.SrcIpAddr)

	req2 := newRequest()
	conn2, err := srv.Request(context.Background(), req2)
	assert.Nil(t, conn2)
	assert.Error(t, err)
}

func TestAllIPsExcluded(t *testing.T) {
	_, ipnet, _ := net.ParseCIDR("192.168.1.2/31")
	srv, err := NewServer([]*net.IPNet{ipnet})
	assert.NotNil(t, srv)
	assert.NoError(t, err)

	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{
		"192.168.1.2/31",
	}
	conn1, err := srv.Request(context.Background(), req1)
	assert.Nil(t, conn1)
	assert.Error(t, err)
}
