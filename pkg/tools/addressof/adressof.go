// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2022 Cisco Systems, Inc.
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

// Package addressof provides convenient functions for converting implementations of an interface to
// pointers to implementations of the interface. Tt turns a concrete struct which implements an interface iface cannot be converted to
//
//	*iface - go complains about it.
//	This does not work:
//	     impl := &interfaceImpl{}
//	     var ptr *iface = &cl
//	This also doesn't work:
//	     impl := &interfaceImpl{}
//	     var ptr *iface = &(cl.(iface)
package addressof

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

// NetworkServiceClient - convenience function to help in converting things from networkservice.NetworkServiceClient to *networkservice.NetworkServiceClient
//
//	it turns a concrete struct which implements networkservice.NetworkServiceClient cannot be converted to
//	*networkservice.NetworkServiceClient - go complains about it.
//	This does not work:
//	     impl := &clientImpl{}
//	     var onHeal *networkservice.NetworkServiceClient = &cl
//	This also doesn't work:
//	     impl := &clientImpl{}
//	     var ptr *networkservice.NetworkServiceClient = &(cl.(networkservice.NetworkServiceClient)
//	Using this function does:
//	     impl := &clientImpl{}
//	     var ptr *networkservice.NetworkServiceClient =NetworkServiceClient(cl)
func NetworkServiceClient(client networkservice.NetworkServiceClient) *networkservice.NetworkServiceClient {
	return &client
}

// NetworkServiceEndpointRegistryClient converts client to *registry.NetworkServiceEndpointRegistryClient.
func NetworkServiceEndpointRegistryClient(client registry.NetworkServiceEndpointRegistryClient) *registry.NetworkServiceEndpointRegistryClient {
	return &client
}

// NetworkServiceRegistryClient converts client to *registry.NetworkServiceRegistryClient.
func NetworkServiceRegistryClient(client registry.NetworkServiceRegistryClient) *registry.NetworkServiceRegistryClient {
	return &client
}
