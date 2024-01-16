// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package nsmgrproxy provides chain of networkservice.NetworkServiceServer chain elements to creating NSMgrProxy
package nsmgrproxy

import (
	"context"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clusterinfo"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interdomainbypass"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/swapip"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/switchcase"
	registryauthorize "github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	registryconnect "github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/fs"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	authmonitor "github.com/networkservicemesh/sdk/pkg/tools/monitorconnection/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

func (n *nsmgrProxyServer) Register(s *grpc.Server) {
	grpcutils.RegisterHealthServices(s, n)
	networkservice.RegisterNetworkServiceServer(s, n)
	networkservice.RegisterMonitorConnectionServer(s, n)
}

type nsmgrProxyServer struct {
	endpoint.Endpoint
}

type serverOptions struct {
	name                             string
	mapipFilePath                    string
	listenOn, registryURL            *url.URL
	authorizeServer                  networkservice.NetworkServiceServer
	authorizeMonitorConnectionServer networkservice.MonitorConnectionServer
	authorizeNSRegistryServer        registryapi.NetworkServiceRegistryServer
	authorizeNSERegistryServer       registryapi.NetworkServiceEndpointRegistryServer
	authorizeNSRegistryClient        registryapi.NetworkServiceRegistryClient
	authorizeNSERegistryClient       registryapi.NetworkServiceEndpointRegistryClient
	dialOptions                      []grpc.DialOption
	dialTimeout                      time.Duration
}

func (s *serverOptions) openMapIPChannel(ctx context.Context) <-chan map[string]string {
	var r = make(chan map[string]string)
	var fCh = fs.WatchFile(ctx, s.mapipFilePath)
	go func() {
		defer close(r)
		for data := range fCh {
			var m map[string]string
			if err := yaml.Unmarshal(data, &m); err != nil {
				log.FromContext(ctx).Errorf("An error during umarshal ipmap: %v", err.Error())
				continue
			}
			select {
			case <-ctx.Done():
				return
			case r <- m:
			}
		}
	}()
	return r
}

// Option modifies option value
type Option func(o *serverOptions)

// WithName sets name for the server
func WithName(name string) Option {
	return func(o *serverOptions) {
		o.name = name
	}
}

// WithAuthorizeServer sets authorize server for the server
func WithAuthorizeServer(authorizeServer networkservice.NetworkServiceServer) Option {
	if authorizeServer == nil {
		panic("authorizeServer cannot be nil")
	}

	return func(o *serverOptions) {
		o.authorizeServer = authorizeServer
	}
}

// WithAuthorizeMonitorConnectionServer sets authorization MonitorConnectionServer chain element
func WithAuthorizeMonitorConnectionServer(authorizeMonitorConnectionServer networkservice.MonitorConnectionServer) Option {
	if authorizeMonitorConnectionServer == nil {
		panic("authorizeMonitorConnectionServer cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeMonitorConnectionServer = authorizeMonitorConnectionServer
	}
}

// WithAuthorizeNSRegistryServer sets authorization NetworkServiceRegistry chain element
func WithAuthorizeNSRegistryServer(authorizeNSRegistryServer registryapi.NetworkServiceRegistryServer) Option {
	if authorizeNSRegistryServer == nil {
		panic("authorizeNSRegistryServer cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSRegistryServer = authorizeNSRegistryServer
	}
}

// WithAuthorizeNSERegistryServer sets authorization NetworkServiceEndpointRegistry chain element
func WithAuthorizeNSERegistryServer(authorizeNSERegistryServer registryapi.NetworkServiceEndpointRegistryServer) Option {
	if authorizeNSERegistryServer == nil {
		panic("authorizeNSERegistryServer cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSERegistryServer = authorizeNSERegistryServer
	}
}

// WithAuthorizeNSRegistryClient sets authorization NetworkServiceRegistry chain element
func WithAuthorizeNSRegistryClient(authorizeNSRegistryClient registryapi.NetworkServiceRegistryClient) Option {
	if authorizeNSRegistryClient == nil {
		panic("authorizeNSRegistryClient cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSRegistryClient = authorizeNSRegistryClient
	}
}

// WithAuthorizeNSERegistryClient sets authorization NetworkServiceEndpointRegistry chain element
func WithAuthorizeNSERegistryClient(authorizeNSERegistryClient registryapi.NetworkServiceEndpointRegistryClient) Option {
	if authorizeNSERegistryClient == nil {
		panic("authorizeNSERegistryClient cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSERegistryClient = authorizeNSERegistryClient
	}
}

// WithListenOn sets current listenOn url
func WithListenOn(u *url.URL) Option {
	return func(o *serverOptions) {
		o.listenOn = u
	}
}

// WithMapIPFilePath sets the custom path for the file that contains internal to external IPs information in YAML format
func WithMapIPFilePath(p string) Option {
	return func(o *serverOptions) {
		o.mapipFilePath = p
	}
}

// WithDialOptions sets connect Options for the server
func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(o *serverOptions) {
		o.dialOptions = dialOptions
	}
}

// WithDialTimeout sets dial timeout for the server
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *serverOptions) {
		o.dialTimeout = dialTimeout
	}
}

// WithRegistryURL sets URL to the registry
func WithRegistryURL(u *url.URL) Option {
	return func(o *serverOptions) {
		o.registryURL = u
	}
}

// NewServer creates new proxy NSMgr
func NewServer(ctx context.Context, regURL *url.URL, tokenGenerator token.GeneratorFunc, options ...Option) endpoint.Endpoint {
	rv := new(nsmgrProxyServer)
	opts := &serverOptions{
		name:                             "nsmgr-proxy-" + uuid.New().String(),
		authorizeServer:                  authorize.NewServer(authorize.Any()),
		authorizeMonitorConnectionServer: authmonitor.NewMonitorConnectionServer(authmonitor.Any()),
		authorizeNSRegistryServer:        registryauthorize.NewNetworkServiceRegistryServer(registryauthorize.Any()),
		authorizeNSERegistryServer:       registryauthorize.NewNetworkServiceEndpointRegistryServer(registryauthorize.Any()),
		authorizeNSRegistryClient:        registryauthorize.NewNetworkServiceRegistryClient(registryauthorize.Any()),
		authorizeNSERegistryClient:       registryauthorize.NewNetworkServiceEndpointRegistryClient(registryauthorize.Any()),
		listenOn:                         &url.URL{Scheme: "unix", Host: "listen.on"},
		mapipFilePath:                    "map-ip.yaml",
	}
	for _, opt := range options {
		opt(opts)
	}

	nseClient := chain.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		clientconn.NewNetworkServiceEndpointRegistryClient(),
		dial.NewNetworkServiceEndpointRegistryClient(ctx,
			dial.WithDialOptions(opts.dialOptions...),
			dial.WithDialTimeout(opts.dialTimeout),
		),
		registryconnect.NewNetworkServiceEndpointRegistryClient(),
	)

	regNSEClient := chain.NewNetworkServiceEndpointRegistryClient(
		clienturl.NewNetworkServiceEndpointRegistryClient(opts.registryURL),
		nseClient,
	)

	proxyRegNSEClient := chain.NewNetworkServiceEndpointRegistryClient(
		clienturl.NewNetworkServiceEndpointRegistryClient(regURL),
		nseClient,
	)

	nsClient := chain.NewNetworkServiceRegistryClient(
		begin.NewNetworkServiceRegistryClient(),
		clientconn.NewNetworkServiceRegistryClient(),
		dial.NewNetworkServiceRegistryClient(ctx,
			dial.WithDialOptions(opts.dialOptions...),
		),
		registryconnect.NewNetworkServiceRegistryClient(),
	)

	regNSClient := chain.NewNetworkServiceRegistryClient(
		clienturl.NewNetworkServiceRegistryClient(opts.registryURL),
		nsClient,
	)

	proxyRegNSClient := chain.NewNetworkServiceRegistryClient(
		clienturl.NewNetworkServiceRegistryClient(regURL),
		nsClient,
	)

	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(opts.name),
		endpoint.WithAuthorizeServer(opts.authorizeServer),
		endpoint.WithAuthorizeMonitorConnectionServer(opts.authorizeMonitorConnectionServer),
		endpoint.WithAdditionalFunctionality(
			switchcase.NewServer(&switchcase.ServerCase{
				Condition: func(ctx context.Context, c *networkservice.Connection) bool {
					return interdomain.Is(c.GetNetworkServiceEndpointName())
				},
				Server: discover.NewServer(proxyRegNSClient, proxyRegNSEClient),
			}, &switchcase.ServerCase{
				Condition: switchcase.Default,
				Server:    discover.NewServer(regNSClient, regNSEClient),
			}),
			switchcase.NewServer(&switchcase.ServerCase{
				Condition: func(ctx context.Context, c *networkservice.Connection) bool {
					var u = clienturlctx.ClientURL(ctx)
					return u != nil && u.Scheme != "tcp"
				},
				Server: interdomainbypass.NewServer(),
			}),
			swapip.NewServer(opts.openMapIPChannel(ctx)),
			clusterinfo.NewServer(),
			connect.NewServer(
				client.NewClient(
					ctx,
					client.WithName(opts.name),
					client.WithDialOptions(opts.dialOptions...),
					client.WithDialTimeout(opts.dialTimeout),
					client.WithoutRefresh(),
					client.WithAdditionalFunctionality(
						swapip.NewClient(opts.openMapIPChannel(ctx)),
					),
				),
			),
		),
	)

	return rv
}
