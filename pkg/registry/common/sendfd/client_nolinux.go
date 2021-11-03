// Copyright (c) 2021 Cisco and/or its affiliates.
//
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

// +build !linux

// Package sendfd provides a registry.NetworkServiceEndpointRegistryClient chain element to convert any unix file socket
// endpoint.URLs into 'inode://${dev}/${ino}' urls and send the fd over the unix file socket.
package sendfd

import (
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
)

// NewNetworkServiceEndpointRegistryClient - creates a Client that if endpoint.Url is of scheme "unix" will replace it with an "inode" scheme url and send the FD over the unix socket
func NewNetworkServiceEndpointRegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return null.NewNetworkServiceEndpointRegistryClient()
}
