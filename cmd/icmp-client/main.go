package main

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/flags"
	"github.com/spf13/pflag"
	"github.com/spiffe/go-spiffe/spiffe"
	"net/url"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffeutils"
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
	log.Entry(ctx).Println("cert id: ", svid)

	cc, err := grpc.DialContext(ctx, ConnectToURL.String(), spiffeutils.WithSpiffe(tlsPeer, 10*time.Second), grpc.WithBlock())
	if err != nil {
		log.Entry(ctx).Fatalf("failed to connect on %q: %+v", &ConnectToURL, err)
	}

	defer cc.Close()

	//nsc := networkservice.NewNetworkServiceClient(cc)
	nsc := client.NewClient(ctx, "", nil, spiffeutils.SpiffeJWTTokenGeneratorFunc(tlsPeer.GetCertificate, 10*time.Second), cc)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "icmp-responder",
		},
	}

	nsc.Request(ctx, request)
	//request := &networkservice.NetworkServiceRequest{
	//	Connection:           &networkservice.Connection{
	//		Id:                         "nsc",
	//		NetworkService:             "icmp-responder",
	//		Path: &networkservice.Path{
	//			Index:                0,
	//			PathSegments:         networkservice.PathSegment{
	//				Name:                 "",
	//				Id:                   "",
	//				Token:                "",
	//				Expires:              nil,
	//				Metrics:              nil,
	//				XXX_NoUnkeyedLiteral: struct{}{},
	//				XXX_unrecognized:     nil,
	//				XXX_sizecache:        0,
	//			},
	//		},
	//	},
	//}
	//conn, err := nsc.Request(ctx, request)
	//if err != nil {
	//	log.Entry(ctx).Fatalln("unable to request network service:", err)
	//}
	//defer nsc.Close(ctx, conn)

	if ctx.Err() != nil {
		log.Entry(ctx).Warnf(ctx.Err().Error())
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
