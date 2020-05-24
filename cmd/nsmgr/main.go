package main

import (
	"context"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/tools/flags"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffeutils"
	"github.com/open-policy-agent/opa/rego"
	"github.com/spf13/pflag"
	"github.com/spiffe/go-spiffe/spiffe"
	"google.golang.org/grpc"
	"net/url"
	"os"
	"strings"
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

	podName := os.Getenv("HOSTNAME")

	log.Entry(ctx).Printf("Args: %s", os.Args)
	log.Entry(ctx).Printf("Name: %s", Name)
	log.Entry(ctx).Printf("BaseDir: %s", BaseDir)
	log.Entry(ctx).Printf("ConnectToURL: %s", ConnectToURL)
	log.Entry(ctx).Printf("ListenOnURL: %s", ListenOnURL)
	log.Entry(ctx).Printf("RegistryURL: %s", RegistryURL)

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		log.Entry(ctx).Printf("ENV: %q", pair)
	}

	reg, err := rego.New(
		rego.Query("true = true"),
	).PrepareForEval(ctx)
	if err != nil {
		log.Entry(ctx).Fatalln(err)
	}

	tlsPeer, err := spiffeutils.NewTLSPeer(spiffe.WithWorkloadAPIAddr("unix:/run/spire/sockets/agent.sock"))
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to create spiffeutils.TLSPeer %+v", err)
	}
	tlsPeer.WaitUntilReady(ctx)
	cert, err := tlsPeer.GetCertificate()
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to create spiffeutils.TLSPeer %+v", err)
	}
	svid, err := spiffeutils.SpiffeIDFromTLS(cert)
	log.Entry(ctx).Println("svid: ", svid)

	// set up registry client
	registryCC, err := grpc.DialContext(ctx, RegistryURL.String(), spiffeutils.WithSpiffe(tlsPeer, 10*time.Minute), grpc.WithBlock())
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to build ipam server %+v", err)
	}
	ep := nsmgr.NewServer("nsmgr-"+podName, &reg, spiffeutils.SpiffeJWTTokenGeneratorFunc(tlsPeer.GetCertificate, 10*time.Minute), registryCC)
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to build ipam server %+v", err)
	}
	log.Entry(ctx).Println("cert id: ", svid)

	server := grpc.NewServer(spiffeutils.SpiffeCreds(tlsPeer, 10*time.Minute))
	ep.Register(server)

	nsmgrCtx := grpcutils.ListenAndServe(ctx, &ConnectToURL, server)

	//healthServer := health.NewServer()
	//grpc_health_v1.RegisterHealthServer(server, healthServer)
	//for _, service := range api.ServiceNames(ep) {
	//	log.Entry(ctx).Println("service: ", service)
	//	healthServer.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_SERVING)
	//}

	for err = range nsmgrCtx {
		log.Entry(ctx).Println("error running nsmgr: ", err)
	}

	log.Entry(ctx).Println("nsmgr exiting")
}


//type endpointServer struct {
//	networkservice.NetworkServiceServer
//	networkservice.MonitorConnectionServer
//}
//
//func NewServer(name string, authzPolicy *rego.PreparedEvalQuery, tokenGenerator token.GeneratorFunc, additionalFunctionality ...networkservice.NetworkServiceServer) endpoint.Endpoint {
//	rv := &endpointServer{}
//	rv.NetworkServiceServer = chain.NewNetworkServiceServer(
//		append([]networkservice.NetworkServiceServer{
//			authorize.NewServer(authzPolicy),
//			setid.NewServer(name),
//			monitor.NewServer(&rv.MonitorConnectionServer),
//			updatepath.NewServer(name, tokenGenerator),
//		}, additionalFunctionality...)...)
//	return rv
//}
//func (e *endpointServer) Register(s *grpc.Server) {
//	networkservice.RegisterNetworkServiceServer(s, e)
//	networkservice.RegisterMonitorConnectionServer(s, e)
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
