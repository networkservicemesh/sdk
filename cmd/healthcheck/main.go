package main

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/flags"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffeutils"
	"github.com/spf13/pflag"
	"github.com/spiffe/go-spiffe/spiffe"
	"google.golang.org/grpc"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"net/url"
	"os"
	"sync"
	"time"
)

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

	monitor, err := grpc.DialContext(ctx,"unix://" + ListenOnURL.String(),spiffeutils.WithSpiffe(tlsPeer,10 * time.Second), grpc.WithBlock())

	hC := healthgrpc.NewHealthClient(monitor)

	wg := &sync.WaitGroup{}
	wg.Add(1)

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

	monitorClient := networkservice.NewMonitorConnectionClient(monitor)

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
	wg.Wait()
}

var Name string
var BaseDir string
var ListenOnURL url.URL
var ConnectToURL url.URL

func Flags(f *pflag.FlagSet) {
	// Standard NSM flags
	f.StringVarP(&Name, flags.NameKey, flags.NameShortHand, "icmp-responder", flags.NameUsageDefault)
	f.StringVarP(&BaseDir, flags.BaseDirKey, flags.BaseDirShortHand, flags.BaseDirDefault, flags.BaseDirUsageDefault)
	flags.URLVarP(f, &ListenOnURL, flags.ListenOnURLKey, flags.ListenOnURLShortHand, &url.URL{Scheme: flags.ListenOnURLSchemeDefault, Path: flags.ListenOnURLPathDefault}, flags.ListenOnURLUsageDefault)
	flags.URLVarP(f, &ConnectToURL, flags.ConnectToURLKey, flags.ConnectToURLShortHand, &url.URL{Scheme: flags.ConnectToURLSchemeDefault, Path: flags.ConnectToURLPathDefault}, flags.ConnectToURLUsageDefault)
}
