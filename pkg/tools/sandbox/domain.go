// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
)

// Domain contains attached to domain nodes, registry
type Domain struct {
	t *testing.T

	Nodes         []*Node
	NSMgrProxy    *NSMgrEntry
	Registry      *RegistryEntry
	RegistryProxy *RegistryEntry

	DNSResolver dnsresolve.Resolver
	Name        string

	supplyURL            func(prefix string) *url.URL
	supplyTokenGenerator SupplyTokenGeneratorFunc
}

// NewNSRegistryClient creates new NS registry client for the domain
func (d *Domain) NewNSRegistryClient(ctx context.Context, tokenTimeout time.Duration) registryapi.NetworkServiceRegistryClient {
	var registryURL *url.URL
	switch {
	case d.Registry != nil:
		registryURL = d.Registry.URL
	case len(d.Nodes) != 0:
		registryURL = d.Nodes[0].URL()
	default:
		return nil
	}

	return registryclient.NewNetworkServiceRegistryClient(ctx, registryURL,
		registryclient.WithDialOptions(d.DefaultDialOptions(tokenTimeout)...))
}

// DefaultDialOptions returns default dial options for the domain
func (d *Domain) DefaultDialOptions(tokenTimeout time.Duration) []grpc.DialOption {
	return DefaultDialOptions(d.supplyTokenGenerator(tokenTimeout))
}
