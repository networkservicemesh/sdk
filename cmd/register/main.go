package main

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
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

	tlsPeer, err := spiffeutils.NewTLSPeer(spiffe.WithWorkloadAPIAddr("unix:/run/spire/sockets/agent.sock"))
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to create spiffeutils.TLSPeer %+v", err)
	}
	tlsPeer.WaitUntilReady(ctx)

	registryServer := newFakeRegistry()
	regCtx := newRegistryServer(ctx, registryServer, &RegistryURL, tlsPeer)
	select {
	case <-regCtx.Done():
	}
	log.Entry(ctx).Println(regCtx.Err())
}

type fakeRegistry struct {
	registrations map[string]*registry.NSERegistration
}

func (f *fakeRegistry) RegisterNSE(ctx context.Context, registration *registry.NSERegistration) (*registry.NSERegistration, error) {
	return registration, nil
}

func (f *fakeRegistry) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return nil
}

func (f *fakeRegistry) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func newFakeRegistry() registry.NetworkServiceRegistryServer {
	reg := next.NewRegistryServer()
	return reg
}

func newRegistryServer(ctx context.Context, r registry.NetworkServiceRegistryServer, url *url.URL, peer spiffeutils.TLSPeer) context.Context {
	server := grpc.NewServer(spiffeutils.SpiffeCreds(peer, 10*time.Minute))
	registry.RegisterNetworkServiceRegistryServer(server, r)

	ctx = grpcutils.ListenAndServe(ctx, url, server)
	return ctx
}

func Flags(f *pflag.FlagSet) {
	// Standard NSM flags
	f.StringVarP(&Name, flags.NameKey, flags.NameShortHand, "icmp-responder", flags.NameUsageDefault)
	f.StringVarP(&BaseDir, flags.BaseDirKey, flags.BaseDirShortHand, flags.BaseDirDefault, flags.BaseDirUsageDefault)
	flags.URLVarP(f, &ListenOnURL, flags.ListenOnURLKey, flags.ListenOnURLShortHand, &url.URL{Scheme: flags.ListenOnURLSchemeDefault, Path: flags.ListenOnURLPathDefault}, flags.ListenOnURLUsageDefault)
	flags.URLVarP(f, &ConnectToURL, flags.ConnectToURLKey, flags.ConnectToURLShortHand, &url.URL{Scheme: flags.ConnectToURLSchemeDefault, Path: flags.ConnectToURLPathDefault}, flags.ConnectToURLUsageDefault)
	flags.URLVarP(f, &RegistryURL, "registry-url", "r", &url.URL{Scheme: "unix", Path: "/registry.socket"}, "path to registry")
}
