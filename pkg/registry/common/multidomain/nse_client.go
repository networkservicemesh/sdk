// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package multidomain TODO
package multidomain

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type multidomainNSEClient struct{}

func (n *multidomainNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	var expiratiomTime *timestamppb.Timestamp
	var registrationTime *timestamppb.Timestamp
	for _, domainNse := range splitByDoamin(nse) {
		resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, domainNse, opts...)
		if err != nil {
			return nil, err
		}
		if expiratiomTime == nil || expiratiomTime.AsTime().Local().After(resp.GetExpirationTime().AsTime().Local()) {
			expiratiomTime = resp.GetExpirationTime()
			registrationTime = resp.GetInitialRegistrationTime()
		}
	}

	nse.ExpirationTime = expiratiomTime
	nse.InitialRegistrationTime = registrationTime

	return nse, nil
}

func (n *multidomainNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (n *multidomainNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	for _, domainNse := range splitByDoamin(nse) {
		_, err := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, domainNse, opts...)
		if err != nil {
			return nil, err
		}
	}
	return new(empty.Empty), nil
}

// NewNetworkServiceEndpointRegistryClient - TODO
func NewNetworkServiceEndpointRegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return new(multidomainNSEClient)
}

func splitByDoamin(in *registry.NetworkServiceEndpoint) []*registry.NetworkServiceEndpoint {
	var result []*registry.NetworkServiceEndpoint

	var domains = make(map[string][]string)

	for _, service := range in.GetNetworkServiceNames() {
		var domain = interdomain.Domain(service)
		domains[domain] = append(domains[domain], service)
	}

	for domain, services := range domains {
		var candidate = in.Clone()
		if domain != "" {
			candidate.Name = interdomain.Target(candidate.Name)
			candidate.Name = interdomain.Join(candidate.Name, domain)
		}
		candidate.NetworkServiceNames = services

		for _, unusedService := range diff(in.NetworkServiceNames, services) {
			delete(candidate.NetworkServiceLabels, unusedService)
		}

		result = append(result, candidate)
	}

	return result
}

func diff(a, b []string) []string {
	var collision = make(map[string]struct{})
	for _, item := range b {
		collision[item] = struct{}{}
	}
	var result []string
	for _, item := range a {
		if _, found := collision[item]; !found {
			result = append(result, item)
		}
	}
	return result
}
