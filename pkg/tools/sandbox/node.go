// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package sandbox

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Node is a NSMgr with Forwarder, NSE registry clients
type Node struct {
	t *testing.T

	NSMgr                   *NSMgrEntry
	ForwarderRegistryClient registryapi.NetworkServiceEndpointRegistryClient
	EndpointRegistryClient  registryapi.NetworkServiceEndpointRegistryClient
	NSRegistryClient        registryapi.NetworkServiceRegistryClient

	*Domain
}

// -----------------
// NSMgr
// -----------------

type nsmgrOptions struct {
	unixURL, tcpURL *url.URL
	tokenTimeout    time.Duration
	supplyNSMgr     SupplyNSMgrFunc
}

// NSMgrOption is an option pattern for NewNSMgr
type NSMgrOption func(t *testing.T, nsmgrOpts *nsmgrOptions)

// WithNSMgrURL sets NewNSMgr serve URL
func WithNSMgrURL(u *url.URL) NSMgrOption {
	return func(t *testing.T, nsmgrOpts *nsmgrOptions) {
		switch u.Scheme {
		case "unix":
			nsmgrOpts.unixURL = u
		case "tcp":
			nsmgrOpts.tcpURL = u
		default:
			require.FailNow(t, "expected one of [unix, tcp] URL scheme, got: "+u.Scheme)
		}
	}
}

// WithNSMgrTokenTimeout sets NewNSMgr token timeout
func WithNSMgrTokenTimeout(tokenTimeout time.Duration) NSMgrOption {
	return func(_ *testing.T, nsmgrOpts *nsmgrOptions) {
		nsmgrOpts.tokenTimeout = tokenTimeout
	}
}

// WithNSMgrSupplier sets NewNSMgr NSMgr supplier
func WithNSMgrSupplier(supplyNSMgr SupplyNSMgrFunc) NSMgrOption {
	return func(_ *testing.T, nsmgrOpts *nsmgrOptions) {
		nsmgrOpts.supplyNSMgr = supplyNSMgr
	}
}

// NewNSMgr creates a new NSMgr
func (n *Node) NewNSMgr(
	ctx context.Context,
	name string,
	opts ...NSMgrOption,
) *NSMgrEntry {
	nsmgrOpts := &nsmgrOptions{
		tcpURL:       TCPURL(n.t),
		tokenTimeout: DefaultTokenTimeout,
		supplyNSMgr:  nsmgr.NewServer,
	}
	WithNSMgrURL(n.supplyURL("nsmgr"))(n.t, nsmgrOpts)

	for _, opt := range opts {
		opt(n.t, nsmgrOpts)
	}

	clientTC := n.supplyClientTC() // NSMgr -> (...)
	tokenGenerator := n.supplyTokenGenerator(nsmgrOpts.tokenTimeout)

	options := []nsmgr.Option{
		nsmgr.WithName(name),
		nsmgr.WithURL(nsmgrOpts.tcpURL.String()),
		nsmgr.WithAuthorizeServer(authorize.NewServer(authorize.Any())),
		nsmgr.WithDialOptions(DefaultSecureDialOptions(clientTC, tokenGenerator)...),
	}

	if n.Registry != nil {
		registryCC := dial(ctx, n.t, n.Registry.URL, clientTC, tokenGenerator)
		options = append(options, nsmgr.WithRegistryClientConn(registryCC))
	}

	entry := &NSMgrEntry{
		Nsmgr: nsmgrOpts.supplyNSMgr(ctx, tokenGenerator, options...),
		Name:  name,
	}

	serve(ctx, n.t, nsmgrOpts.tcpURL, n.supplyServerTC(), entry.Register)
	log.FromContext(ctx).Infof("Started listening NSMgr %s on %s", name, nsmgrOpts.tcpURL.String())

	var cc grpc.ClientConnInterface
	if nsmgrOpts.unixURL != nil {
		entry.URL = nsmgrOpts.unixURL

		serve(ctx, n.t, nsmgrOpts.unixURL, n.supplyServerTC(), entry.Register)
		log.FromContext(ctx).Infof("Started listening NSMgr %s on %s", name, nsmgrOpts.unixURL.String())
	} else {
		entry.URL = nsmgrOpts.tcpURL
	}

	cc = dial(ctx, n.t, entry.URL, n.supplyClientTC(), tokenGenerator) // (...) -> NSMgr

	n.NSMgr = entry
	n.ForwarderRegistryClient = registryclient.NewNetworkServiceEndpointRegistryInterposeClient(ctx, cc)
	n.EndpointRegistryClient = registryclient.NewNetworkServiceEndpointRegistryClient(ctx, cc)
	n.NSRegistryClient = registryclient.NewNetworkServiceRegistryClient(cc)

	return entry
}

// -----------------
// Forwarder
// -----------------

type forwarderOptions struct {
	tokenTimeout                  time.Duration
	additionalServerFunctionality []networkservice.NetworkServiceServer
	additionalClientFunctionality []networkservice.NetworkServiceClient
}

// ForwarderOption is an option pattern for NewForwarder
type ForwarderOption func(forwarderOpts *forwarderOptions)

// WithForwarderTokenTimeout sets NewForwarder token timeout
func WithForwarderTokenTimeout(tokenTimeout time.Duration) ForwarderOption {
	return func(forwarderOpts *forwarderOptions) {
		forwarderOpts.tokenTimeout = tokenTimeout
	}
}

// WithForwarderAdditionalServerFunctionality sets NewForwarder additional server functionality
func WithForwarderAdditionalServerFunctionality(servers ...networkservice.NetworkServiceServer) ForwarderOption {
	return func(forwarderOpts *forwarderOptions) {
		forwarderOpts.additionalServerFunctionality = servers
	}
}

// WithForwarderAdditionalClientFunctionality sets NewForwarder additional client functionality
func WithForwarderAdditionalClientFunctionality(clients ...networkservice.NetworkServiceClient) ForwarderOption {
	return func(forwarderOpts *forwarderOptions) {
		forwarderOpts.additionalClientFunctionality = clients
	}
}

// NewForwarder starts a new forwarder and registers it on the node NSMgr
func (n *Node) NewForwarder(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	opts ...ForwarderOption,
) *EndpointEntry {
	forwarderOpts := &forwarderOptions{
		tokenTimeout: DefaultTokenTimeout,
	}

	for _, opt := range opts {
		opt(forwarderOpts)
	}

	if nse.Url == "" {
		nse.Url = n.supplyURL("forwarder").String()
	}

	clientTC := n.supplyClientTC()
	tokenGenerator := n.supplyTokenGenerator(forwarderOpts.tokenTimeout)

	entry := new(EndpointEntry)
	*entry = *n.newEndpoint(ctx, nse, tokenGenerator, n.ForwarderRegistryClient, append(forwarderOpts.additionalServerFunctionality,
		clienturl.NewServer(n.NSMgr.URL),
		heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(entry))),
		connect.NewServer(ctx,
			client.NewCrossConnectClientFactory(
				client.WithName(nse.Name),
				client.WithAdditionalFunctionality(forwarderOpts.additionalClientFunctionality...)),
			connect.WithDialOptions(DefaultSecureDialOptions(clientTC, tokenGenerator)...)),
	)...)

	return entry
}

// -----------------
// Endpoint
// -----------------

type endpointOptions struct {
	tokenTimeout            time.Duration
	additionalFunctionality []networkservice.NetworkServiceServer
}

// EndpointOption is an option pattern for NewEndpoint
type EndpointOption func(endpointOpts *endpointOptions)

// WithEndpointTokenTimeout sets NewEndpoint token timeout
func WithEndpointTokenTimeout(tokenTimeout time.Duration) EndpointOption {
	return func(endpointOpts *endpointOptions) {
		endpointOpts.tokenTimeout = tokenTimeout
	}
}

// WithEndpointAdditionalFunctionality sets NewEndpoint additional functionality
func WithEndpointAdditionalFunctionality(servers ...networkservice.NetworkServiceServer) EndpointOption {
	return func(endpointOpts *endpointOptions) {
		endpointOpts.additionalFunctionality = servers
	}
}

// NewEndpoint starts a new endpoint and registers it on the node NSMgr
func (n *Node) NewEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	opts ...EndpointOption,
) *EndpointEntry {
	endpointOpts := &endpointOptions{
		tokenTimeout: DefaultTokenTimeout,
	}

	for _, opt := range opts {
		opt(endpointOpts)
	}

	if nse.Url == "" {
		nse.Url = n.supplyURL("nse").String()
	}

	tokenGenerator := n.supplyTokenGenerator(endpointOpts.tokenTimeout)

	return n.newEndpoint(ctx, nse, tokenGenerator, n.EndpointRegistryClient, endpointOpts.additionalFunctionality...)
}

func (n *Node) newEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	tokenGenerator token.GeneratorFunc,
	registryClient registryapi.NetworkServiceEndpointRegistryClient,
	additionalFunctionality ...networkservice.NetworkServiceServer,
) *EndpointEntry {
	name := nse.Name
	entry := endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(name),
		endpoint.WithAdditionalFunctionality(additionalFunctionality...),
	)

	serveURL, err := url.Parse(nse.Url)
	require.NoError(n.t, err)

	serve(ctx, n.t, serveURL, n.supplyServerTC(), entry.Register)

	n.registerEndpoint(ctx, nse, registryClient)

	log.FromContext(ctx).Infof("Started listening endpoint %s on %s", nse.Name, serveURL.String())

	return &EndpointEntry{
		Endpoint: entry,
		Name:     name,
		URL:      serveURL,
	}
}

// RegisterForwarder registers forwarder on the node NSMgr
func (n *Node) RegisterForwarder(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) {
	n.registerEndpoint(ctx, nse, n.ForwarderRegistryClient)
}

// RegisterEndpoint registers endpoint on the node NSMgr
func (n *Node) RegisterEndpoint(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) {
	n.registerEndpoint(ctx, nse, n.EndpointRegistryClient)
}

func (n *Node) registerEndpoint(
	ctx context.Context,
	nse *registryapi.NetworkServiceEndpoint,
	registryClient registryapi.NetworkServiceEndpointRegistryClient,
) {
	for _, nsName := range nse.NetworkServiceNames {
		_, err := n.NSRegistryClient.Register(ctx, &registryapi.NetworkService{
			Name:    nsName,
			Payload: payload.IP,
		})
		require.NoError(n.t, err)
	}

	reg, err := registryClient.Register(ctx, nse)
	require.NoError(n.t, err)

	nse.Name = reg.Name
	nse.ExpirationTime = reg.ExpirationTime
}

// -----------------
// Client
// -----------------

type clientOptions struct {
	tokenTimeout            time.Duration
	additionalFunctionality []networkservice.NetworkServiceClient
}

// ClientOption is an option pattern for NewClient
type ClientOption func(endpointOpts *clientOptions)

// WithClientTokenTimeout sets NewClient token timeout
func WithClientTokenTimeout(tokenTimeout time.Duration) ClientOption {
	return func(clientOpts *clientOptions) {
		clientOpts.tokenTimeout = tokenTimeout
	}
}

// WithClientAdditionalFunctionality sets NewClient additional functionality
func WithClientAdditionalFunctionality(clients ...networkservice.NetworkServiceClient) ClientOption {
	return func(clientOpts *clientOptions) {
		clientOpts.additionalFunctionality = clients
	}
}

// NewClient starts a new client and connects it to the node NSMgr
func (n *Node) NewClient(
	ctx context.Context,
	opts ...ClientOption,
) networkservice.NetworkServiceClient {
	clientOpts := &clientOptions{
		tokenTimeout: DefaultTokenTimeout,
	}

	for _, opt := range opts {
		opt(clientOpts)
	}

	return client.NewClient(
		ctx,
		dial(ctx, n.t, n.NSMgr.URL, n.supplyClientTC(), n.supplyTokenGenerator(clientOpts.tokenTimeout)),
		client.WithAuthorizeClient(authorize.NewClient(authorize.Any())),
		client.WithAdditionalFunctionality(clientOpts.additionalFunctionality...),
	)
}
