package nsmgr_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"time"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	api_registry "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/registry"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	interpose_reg "github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	adapters2 "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type Cluster struct {
	Nodes       []*Node
	Registry    registry.Registry
	RegistryURL *url.URL
}

type Node struct {
	Forwarder    endpoint.Endpoint
	ForwarderURL *url.URL
	NSMgr        nsmgr.Nsmgr
	NSMgrURL     *url.URL
}

type NSMGRSuite struct {
	suite.Suite
	resources []context.CancelFunc
	cluster   *Cluster
}

func (t *NSMGRSuite) Cluster() Cluster {
	return *t.cluster
}
func (t *NSMGRSuite) TearDownTest() {
	for _, cancel := range t.resources {
		cancel()
	}
}

func (t *NSMGRSuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.resources = append(t.resources, cancel)
	t.setupCluster(ctx, 2)
}

func (t *NSMGRSuite) SetupSuite() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr))
}

func (t *NSMGRSuite) setupCluster(ctx context.Context, nodesCount int) {
	if ctx == nil {
		panic("ctx should not be nil")
	}
	if nodesCount < 1 {
		panic("nodesCount should be more than 0")
	}
	t.cluster = new(Cluster)
	t.cluster.Registry, t.cluster.RegistryURL = t.NewRegistry(ctx)
	for i := 0; i < nodesCount; i++ {
		var node = new(Node)
		node.NSMgr, node.NSMgrURL = t.NewNSMgr(ctx, t.cluster.RegistryURL)
		t.cluster.Nodes = append(t.cluster.Nodes, node)
		forwarderName := "cross-nse" + uuid.New().String()
		node.Forwarder, node.ForwarderURL = t.NewCrossConnectNSE(ctx, forwarderName, node.NSMgrURL)
		forwarderRegistrationClient := chain.NewNetworkServiceEndpointRegistryClient(
			interpose_reg.NewNetworkServiceEndpointRegistryClient(),
			adapters2.NetworkServiceEndpointServerToClient(node.NSMgr.NetworkServiceEndpointRegistryServer()),
		)
		_, err := forwarderRegistrationClient.Register(context.Background(), &api_registry.NetworkServiceEndpoint{
			Url:  node.ForwarderURL.String(),
			Name: forwarderName,
		})
		t.NoError(err)
	}

}

func (t *NSMGRSuite) NewClient(ctx context.Context, connectTO *url.URL) networkservice.NetworkServiceClient {
	return client.NewClient(ctx, "nsc-1", nil, tokenGenerator, t.DialContext(ctx, connectTO))
}

func (t *NSMGRSuite) NewEndpoint(ctx context.Context, registration *api_registry.NetworkServiceEndpoint, nsmgr nsmgr.Nsmgr) (endpoint.Endpoint, *url.URL) {
	result := endpoint.NewServer(ctx,
		registration.Name,
		authorize.NewServer(),
		tokenGenerator)
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	t.serve(ctx, serveURL, result.Register)
	_, err := nsmgr.NetworkServiceEndpointRegistryServer().Register(context.Background(), &api_registry.NetworkServiceEndpoint{
		Name:                registration.Name,
		Url:                 serveURL.String(),
		NetworkServiceNames: registration.NetworkServiceNames,
	})
	t.NoError(err)
	for _, service := range registration.NetworkServiceNames {
		_, err = nsmgr.NetworkServiceRegistryServer().Register(ctx, &api_registry.NetworkService{
			Name:    service,
			Payload: "IP",
		})
		t.NoError(err)
	}
	log.Entry(ctx).Infof("Endpoint listen on: %v", serveURL)
	return result, serveURL
}

func (t *NSMGRSuite) NewRegistry(ctx context.Context) (registry.Registry, *url.URL) {
	result := registry.NewServer(chain.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer()), chain.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer()))
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	t.serve(ctx, serveURL, result.Register)
	log.Entry(ctx).Infof("Registry listen on: %v", serveURL)
	return result, serveURL
}

func (t *NSMGRSuite) NewCrossConnectNSE(ctx context.Context, name string, connectTo *url.URL) (endpoint.Endpoint, *url.URL) {
	var crossNSE endpoint.Endpoint
	crossNSE = endpoint.NewServer(ctx,
		name,
		authorize.NewServer(),
		tokenGenerator,
		// Statically set the url we use to the unix file socket for the NSMgr
		clienturl.NewServer(connectTo),
		connect.NewServer(
			ctx,
			client.NewClientFactory(
				name,
				// What to call onHeal
				addressof.NetworkServiceClient(adapters.NewServerToClient(crossNSE)),
				tokenGenerator,
			),
			grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		),
	)
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	t.serve(ctx, serveURL, crossNSE.Register)
	log.Entry(ctx).Infof("%v listen on: %v", name, serveURL)
	return crossNSE, serveURL
}

func (t *NSMGRSuite) DialContext(ctx context.Context, u *url.URL) *grpc.ClientConn {
	conn, err := grpc.DialContext(ctx, grpcutils.URLToTarget(u),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	t.resources = append(t.resources, func() {
		_ = conn.Close()
	})
	t.Nil(err, "Can not dial to", u)
	return conn
}

func (t *NSMGRSuite) NewNSMgr(ctx context.Context, registryURL *url.URL) (nsmgr.Nsmgr, *url.URL) {
	var registryCC *grpc.ClientConn
	if registryURL != nil {
		registryCC = t.DialContext(ctx, registryURL)
	}
	listener, err := net.Listen("tcp", ":0")
	t.NoError(err)
	serveURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:" + fmt.Sprint(listener.Addr().(*net.TCPAddr).Port)}
	t.NoError(listener.Close())

	nsmgrReg := &api_registry.NetworkServiceEndpoint{
		Name: "nsmgr-" + uuid.New().String(),
		Url:  serveURL.String(),
	}
	// Serve NSMGR, Use in memory registry server
	nsmgr := nsmgr.NewServer(ctx, nsmgrReg, authorize.NewServer(), tokenGenerator, registryCC, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))

	t.serve(ctx, serveURL, nsmgr.Register)
	log.Entry(ctx).Infof("NSMgr listen on: %v", serveURL)
	return nsmgr, serveURL
}

func (t *NSMGRSuite) serve(ctx context.Context, url *url.URL, register func(server *grpc.Server), opts ...grpc.ServerOption) {
	server := grpc.NewServer(opts...)
	register(server)
	errCh := grpcutils.ListenAndServe(ctx, url, server)
	go func() {
		select {
		case <-ctx.Done():
			log.Entry(ctx).Infof("Stop serve: %v", url.String())
			return
		case err := <-errCh:
			t.NoError(err, "Received error from the serve of url: ", url.String())
		}
	}()
}

func tokenGenerator(_ credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}
