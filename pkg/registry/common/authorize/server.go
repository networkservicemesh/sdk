// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2021 Cisco Systems, Inc.
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

// Package authorize provides authz checks for incoming or returning connections.
package authorize

import (
	"context"
	"errors"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

type authorizeNSEServer struct {
	registerPolicies   policiesList
	unregisterPolicies policiesList
	spiffieIDNSEsMap   *spiffieIDNSEsMap
}

// NewNetworkServiceEndpointRegistryServer - returns a new authorization registry.NetworkServiceEndpointRegistryServer
// Authorize registry server checks spiffieID of NSE.
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	var s = &authorizeNSEServer{
		registerPolicies:   policiesList{},
		unregisterPolicies: policiesList{},
		spiffieIDNSEsMap:   new(spiffieIDNSEsMap),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *authorizeNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	spiffieID, err := getSpiffeID(ctx)
	if err != nil {
		return nil, err
	}

	rawMap := make(map[string][]string)
	s.spiffieIDNSEsMap.Range(func(key string, value []string) bool {
		rawMap[key] = value
		return true
	})

	input := RegistryOpaInput{
		SpiffieID:        spiffieID,
		NSEName:          nse.Name,
		SpiffieIDNSEsMap: rawMap,
	}
	if err := s.registerPolicies.check(ctx, input); err != nil {
		return nil, err
	}

	nses, ok := s.spiffieIDNSEsMap.Load(spiffieID)
	if !ok {
		nses = make([]string, 0)
	}
	nses = append(nses, nse.Name)
	s.spiffieIDNSEsMap.Store(spiffieID, nses)

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *authorizeNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *authorizeNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	// id, err := getSpiffeID(ctx)
	// if err != nil {
	// 	return nil, err
	// }

	// if err := s.policies.check(ctx, id); err != nil {
	// 	return nil, err
	// }

	// TODO(nikita): one endpoint can't delete another (add check)

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func getSpiffeID(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", errors.New("fail to get peer from context")
	}

	cert := opa.ParseX509Cert(p.AuthInfo)
	if cert != nil {
		spiffeID, err := x509svid.IDFromCert(cert)
		if err == nil {
			return spiffeID.String(), nil
		}
		return "", errors.New("fail to get Spiffe ID from certificate")
	}
	return "", errors.New("fail to get certificate from peer")
}
