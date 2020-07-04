package connect_test

import (
	"errors"
	"fmt"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"testing"
	"time"
)

var notFoundError = errors.New("not found")

type resource interface {
	Close() error
}

func newInterdomainTestingTool(t *testing.T) *interdomainTestingTool {
	return &interdomainTestingTool{
		ports: map[string]int{},
		T:     t,
	}
}

type interdomainTestingTool struct {
	ports     map[string]int
	resources []resource
	*testing.T
}

func (t *interdomainTestingTool) LookupSRV(_ context.Context, service, proto, domain string) (string, []*net.SRV, error) {
	if v, ok := t.ports[domain]; ok {
		return fmt.Sprintf("_%v._%v.%v", service, proto, domain), []*net.SRV{{
			Port:   uint16(v),
			Target: domain,
		}}, nil
	}
	return "", nil, notFoundError
}

func (t *interdomainTestingTool) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	if _, ok := t.ports[host]; ok {
		return []net.IPAddr{{
			IP: net.ParseIP("127.0.0.1"),
		}}, nil
	}
	return nil, notFoundError
}

func (t *interdomainTestingTool) dialDomain(domain string) *grpc.ClientConn {
	_, srvs, err := t.LookupSRV(context.Background(), "", "tcp", domain)
	require.Nil(t, err)
	require.NotEmpty(t, srvs)
	ips, err := t.LookupIPAddr(context.Background(), srvs[0].Target)
	require.Nil(t, err)
	require.NotEmpty(t, ips)
	conn, err := grpc.Dial(fmt.Sprintf("%v:%v", ips[0].IP.String(), srvs[0].Port), grpc.WithInsecure())
	require.Nil(t, err)
	t.resources = append(t.resources, conn)
	return conn
}

func (t *interdomainTestingTool) startNetworkServiceRegistryServerAsync(domain string, server registry.NetworkServiceRegistryServer) {
	t.startServerAsync(domain, func(s *grpc.Server) {
		registry.RegisterNetworkServiceRegistryServer(s, server)
		grpcutils.RegisterHealthServices(s, server)
	})
	t.waitServerReady(domain)
}

func (t *interdomainTestingTool) startNetworkServiceEndpointRegistryServerAsync(domain string, server registry.NetworkServiceEndpointRegistryServer) {
	t.startServerAsync(domain, func(s *grpc.Server) {
		registry.RegisterNetworkServiceEndpointRegistryServer(s, server)
		grpcutils.RegisterHealthServices(s, server)
	})
	t.waitServerReady(domain)
}

func (t *interdomainTestingTool) waitServerReady(domain string) {
	c := grpc_health_v1.NewHealthClient(t.dialDomain(domain))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := grpcutils.WaitStatusServing(ctx, c)
	require.Nil(t, err)
}

func (t *interdomainTestingTool) startServerAsync(domain string, registerFunc func(server *grpc.Server)) {
	s := grpc.NewServer()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	t.ports[domain] = l.Addr().(*net.TCPAddr).Port
	t.resources = append(t.resources, l)
	logrus.Infof("listening domain %v by %v", domain, l.Addr().String())
	registerFunc(s)
	go func() {
		_ = s.Serve(l)
	}()
}

func (t *interdomainTestingTool) cleanup() {
	for _, r := range t.resources {
		_ = r.Close()
	}
	t.resources = nil
}

var _ dnsresolve.Resolver = (*interdomainTestingTool)(nil)
