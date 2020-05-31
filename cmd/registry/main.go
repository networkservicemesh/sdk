package main

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/flags"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffeutils"
	"github.com/spf13/pflag"
	"github.com/spiffe/go-spiffe/spiffe"
	"google.golang.org/grpc"
	"net/url"
	"os"
	"time"
)

var Name string
var BaseDir string
var ConnectToURL url.URL
var ListenOnURL url.URL
var RegistryURL url.URL

func main() {
	ctx := context.Background()

	flagSet := pflag.FlagSet{}
	Flags(&flagSet)

	populateFromEnv := flags.FromEnv(flags.EnvPrefix, flags.EnvReplacer, &flagSet)
	populateFromEnv()

	err := flagSet.Parse(os.Args)
	if err != nil {
		log.Entry(ctx).Fatalln(err)
	}

	storage := &memory.Storage{}
	nsmReg := memory.NewNSMRegistryServer(storage, "nsm-name")
	nsReg := memory.NewNetworkServiceRegistryServer("nsm-name", storage)
	nsdReg := memory.NewNetworkServiceDiscoveryServer(storage)

	tlsPeer, err := spiffeutils.NewTLSPeer(spiffe.WithWorkloadAPIAddr("unix:/run/spire/sockets/agent.sock"))
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to create spiffeutils.TLSPeer %+v", err)
	}
	tlsPeer.WaitUntilReady(ctx)

	server := grpc.NewServer(spiffeutils.SpiffeCreds(tlsPeer, 10*time.Minute))

	registry.RegisterNsmRegistryServer(server, nsmReg)
	registry.RegisterNetworkServiceRegistryServer(server, nsReg)
	registry.RegisterNetworkServiceDiscoveryServer(server, nsdReg)

	errCh := grpcutils.ListenAndServe(ctx, &RegistryURL, server)

	for err := range errCh {
		log.Entry(ctx).Println(err)
	}
}

//func main() {
//	ctx := context.Background()
//
//	flagSet := pflag.FlagSet{}
//	Flags(&flagSet)
//
//	populateFromEnv := flags.FromEnv(flags.EnvPrefix, flags.EnvReplacer, &flagSet)
//	populateFromEnv()
//
//	err := flagSet.Parse(os.Args)
//	if err != nil {
//		log.Entry(ctx).Fatalln(err)
//	}
//
//	tlsPeer, err := spiffeutils.NewTLSPeer(spiffe.WithWorkloadAPIAddr("unix:/run/spire/sockets/agent.sock"))
//	if err != nil {
//		log.Entry(ctx).Fatalf("Error attempting to create spiffeutils.TLSPeer %+v", err)
//	}
//	tlsPeer.WaitUntilReady(ctx)
//
//	f := &fakeRegistry{
//		registrations: make(map[string]*registry.NSERegistration, 0),
//	}
//	registryServer := newFakeRegistry(f)
//	discoveryServer := newFakeDiscovery(f)
//	regCtx := newRegistryServer(ctx, registryServer, discoveryServer, &RegistryURL, tlsPeer)
//	for err := range regCtx {
//		log.Entry(ctx).Println(err)
//	}
//	log.Entry(ctx).Println("registry server exiting")
//}

//type fakeRegistry struct {
//	registrations map[string]*registry.NSERegistration
//}
//
//func (f *fakeRegistry) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
//	log.Entry(ctx).Printf("request: %+v", request)
//
//	if entry, ok := f.registrations[request.NetworkServiceName]; ok {
//		log.Entry(ctx).Println("entry: %+v", entry)
//		payload := entry.NetworkService.Payload
//		networkService := entry.NetworkService
//		endpoints := []*registry.NetworkServiceEndpoint{entry.NetworkServiceEndpoint}
//		networkServiceManagers := map[string]*registry.NetworkServiceManager{
//			entry.NetworkServiceEndpoint.NetworkServiceManagerName: entry.NetworkServiceManager,
//		}
//		response := &registry.FindNetworkServiceResponse{
//			Payload:                 payload,
//			NetworkService:          networkService,
//			NetworkServiceManagers:  networkServiceManagers,
//			NetworkServiceEndpoints: endpoints,
//		}
//		log.Entry(ctx).Printf("payload: %+v", payload)
//		log.Entry(ctx).Printf("%ns: %+v", networkService)
//		log.Entry(ctx).Printf("nse: %+v", entry.NetworkServiceEndpoint)
//		log.Entry(ctx).Printf("nsms: %+v", entry.NetworkServiceManager)
//		log.Entry(ctx).Printf("response: %+v", request)
//		return response, nil
//	}
//	return nil, status.Error(codes.NotFound, fmt.Sprintf("network service not found: %s", request.GetNetworkServiceName()))
//}
//
//func (f *fakeRegistry) RegisterNSE(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
//	log.Entry(ctx).Printf("Register: %+v", registration)
//	f.registrations[registration.NetworkService.Name] = registration
//	return registration, nil
//}
//
//func (f *fakeRegistry) BulkRegisterNSE(registration registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
//	panic("implement me")
//}
//
//func (f *fakeRegistry) RemoveNSE(ctx context.Context, registration *registry.RemoveNSERequest) (*empty.Empty, error) {
//	// TODO something
//	return &empty.Empty{}, nil
//}
//
//func newFakeRegistry(reg *fakeRegistry) registry.NetworkServiceRegistryServer {
//	next.NewNetworkServiceRegistryServer(reg)
//	return reg
//}
//
//func newRegistryServer(ctx context.Context, r registry.NetworkServiceRegistryServer, d registry.NetworkServiceDiscoveryServer, url *url.URL, peer spiffeutils.TLSPeer) <-chan error {
//	server := grpc.NewServer(spiffeutils.SpiffeCreds(peer, 10*time.Minute))
//	registry.RegisterNetworkServiceRegistryServer(server, r)
//	registry.RegisterNetworkServiceDiscoveryServer(server, d)
//
//	errCh := grpcutils.ListenAndServe(ctx, url, server)
//	return errCh
//}
//
//func newFakeDiscovery(f *fakeRegistry) registry.NetworkServiceDiscoveryServer {
//	return f
//}
//
func Flags(f *pflag.FlagSet) {
	// Standard NSM flags
	f.StringVarP(&Name, flags.NameKey, flags.NameShortHand, "icmp-responder", flags.NameUsageDefault)
	f.StringVarP(&BaseDir, flags.BaseDirKey, flags.BaseDirShortHand, flags.BaseDirDefault, flags.BaseDirUsageDefault)
	flags.URLVarP(f, &ListenOnURL, flags.ListenOnURLKey, flags.ListenOnURLShortHand, &url.URL{Scheme: flags.ListenOnURLSchemeDefault, Path: flags.ListenOnURLPathDefault}, flags.ListenOnURLUsageDefault)
	flags.URLVarP(f, &ConnectToURL, flags.ConnectToURLKey, flags.ConnectToURLShortHand, &url.URL{Scheme: flags.ConnectToURLSchemeDefault, Path: flags.ConnectToURLPathDefault}, flags.ConnectToURLUsageDefault)
	flags.URLVarP(f, &RegistryURL, "registry-url", "r", &url.URL{Scheme: "unix", Path: "/registry.socket"}, "path to registry")
}
