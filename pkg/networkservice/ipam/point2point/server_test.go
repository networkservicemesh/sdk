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

	conn4, err := srv.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "id",
			NetworkService: "ns",
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{},
			},
		},
		MechanismPreferences: nil,
	})
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
