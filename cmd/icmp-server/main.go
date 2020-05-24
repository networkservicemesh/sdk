package main

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/tools/flags"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/spf13/pflag"
	"github.com/spiffe/go-spiffe/spiffe"
	"google.golang.org/grpc/health"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffeutils"

	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)
var Name string
var BaseDir string
var ListenOnURL url.URL
var ConnectToURL url.URL

var CidrPrefix string

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
	log.Entry(ctx).Printf("ListenOnURL: %s", ListenOnURL)
	log.Entry(ctx).Printf("ConnectToURL: %s", ConnectToURL)
	log.Entry(ctx).Printf("CIDR Prefix: %s", CidrPrefix)

	log.Entry(ctx).Println()
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
	//tlsPeer, err := spiffeutils.NewTLSPeer(spiffeutils.tim)
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to create spiffeutils.TLSPeer %+v", err)
	}
	tlsPeer.WaitUntilReady(ctx)
	cert, err := tlsPeer.GetCertificate()
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to create spiffeutils.TLSPeer %+v", err)
	}
	svid, err := spiffeutils.SpiffeIDFromTLS(cert)
	//if err != nil {
	//	log.Entry(ctx).Fatalf("Error attempting to create spiffeutils.TLSPeer %+v", err)
	//}
	log.Entry(ctx).Println("svid: ", svid)
	//log.Entry(ctx).Println("tlsPeer: ", tlsPeer)
	//log.Entry(ctx).Println("tlsPeer.GetCertificate()", cert)

	server := grpc.NewServer(spiffeutils.SpiffeCreds(tlsPeer, 10 * time.Minute))

	_, ipnet, err := net.ParseCIDR(CidrPrefix)
	if err != nil {
		log.Entry(ctx).Fatalf("Error parsing cidr: %+v", err)
	}
	prefixes := []*net.IPNet{
		ipnet,
	}

	ipamServer, err := point2pointipam.NewServer(prefixes)
	if err != nil {
		log.Entry(ctx).Fatalf("Error attempting to build ipam server %+v", err)
	}

	//tlsPeer.WaitUntilReady(ctx)
	//cert, _ := tlsPeer.GetCertificate()
	//svid, _ := spiffeutils.SpiffeIDFromTLS(cert)
	log.Entry(ctx).Println("cert id: ", svid)

	endpoint := endpoint.NewServer("icmp-server", &reg, spiffeutils.SpiffeJWTTokenGeneratorFunc(tlsPeer.GetCertificate, 10*time.Minute),
		ipamServer,
		kernel.NewServer(),
	)
	endpoint.Register(server)

	cc, err := grpc.DialContext(ctx,ConnectToURL.String(),spiffeutils.WithSpiffe(tlsPeer,10 * time.Second), grpc.WithBlock())
	//cc, err := grpc.Dial("unix://" + ConnectToURL.String())
	if err != nil {
		log.Entry(ctx).Fatalf("failed to connect on %q: %+v", &ConnectToURL, err)
	}

	defer cc.Close()

	ListenOnURL.Scheme = "unix"

	srvCtx := grpcutils.ListenAndServe(ctx, &ListenOnURL, server)

	registryClient := registry.NewNetworkServiceRegistryClient(cc)

	nseRegistration := &registry.NSERegistration{
		NetworkService:         &registry.NetworkService{
			Name:                 "icmp-responder",
			Payload:              "IP",
		},
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name:                      podName,
			Payload:                   "IP",
			NetworkServiceName:        "icmp-responder",
			NetworkServiceManagerName: "",
			Labels:                   nil,
			State:                     "",
		},
	}

	log.Entry(ctx).Printf("Attempting to register NSE")
	registerNSERet, err := registryClient.RegisterNSE(ctx, nseRegistration)
	log.Entry(ctx).Printf("nse ret: %q", registerNSERet)
	if err != nil {
		log.Entry(ctx).Fatalf("failed to register nse on %q: %+v", &nseRegistration, err)
	}


	monitor, err := grpc.DialContext(ctx,ListenOnURL.String(),spiffeutils.WithSpiffe(tlsPeer,10 * time.Second), grpc.WithBlock())
	if err != nil {
		log.Entry(ctx).Fatalf("failed to connect monitor on %q: %+v", &monitor, err)
	}

	monitorClient := networkservice.NewMonitorConnectionClient(monitor)

	health.NewServer()

	hC := healthgrpc.NewHealthClient(monitor)

	go func() {
		for {
			resp, err := hC.Check(ctx, &healthpb.HealthCheckRequest{
				Service: "connection.MonitorConnection",
			})
			if err != nil {
				log.Entry(ctx).Fatalf("hC check failed", &hC, err)
			}
			log.Entry(ctx).Println("resp status", resp)

			time.Sleep(5 * time.Second)
		}
	}()

	mCC, err := monitorClient.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{
		PathSegments:         nil,
	})
	if err != nil {
		log.Entry(ctx).Fatalf("mcc failed", &mCC, err)
	}

	go func() {
		log.Entry(ctx).Println("attempting to monitor")
		log.Entry(ctx).Println(mCC.Recv())
	}()

	for err = range srvCtx {
		log.Entry(ctx).Println("server err:", err)
		time.Sleep(5 * time.Second)
	}

	log.Entry(ctx).Warnf("complete!")
}


func Flags(f *pflag.FlagSet) {
	// Standard NSM flags
	f.StringVarP(&Name, flags.NameKey, flags.NameShortHand, "icmp-responder", flags.NameUsageDefault)
	f.StringVarP(&BaseDir, flags.BaseDirKey, flags.BaseDirShortHand, flags.BaseDirDefault, flags.BaseDirUsageDefault)
	flags.URLVarP(f, &ListenOnURL, flags.ListenOnURLKey, flags.ListenOnURLShortHand, &url.URL{Scheme: flags.ListenOnURLSchemeDefault, Path: flags.ListenOnURLPathDefault}, flags.ListenOnURLUsageDefault)
	flags.URLVarP(f, &ConnectToURL, flags.ConnectToURLKey, flags.ConnectToURLShortHand, &url.URL{Scheme: flags.ConnectToURLSchemeDefault, Path: flags.ConnectToURLPathDefault}, flags.ConnectToURLUsageDefault)

	// icmp-server specific flags
	f.StringVarP(&CidrPrefix, "CIDR_PREFIX", "p", "169.254.0.0/16", "CIDR Prefix to assign IPs from")
}


// TODO Remove endpointServer, NewServer and Register when nsmgr is updated with timeout

//type endpointServer struct {
//	networkservice.NetworkServiceServer
//	networkservice.MonitorConnectionServer
//}

//// NewServer - returns a NetworkServiceMesh client as a chain of the standard Client pieces plus whatever
////             additional functionality is specified
////             - name - name of the NetworkServiceServer
////             - tokenGenerator - token.GeneratorFunc - generates tokens for use in Path
////             - additionalFunctionality - any additional NetworkServiceServer chain elements to be included in the chain
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
