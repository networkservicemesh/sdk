// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package nsmgr

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	null_registry "github.com/networkservicemesh/sdk/pkg/registry/common/null"
)

// newRecvFD - construct a recvfd server
func newRecvFD() networkservice.NetworkServiceServer {
	return null.NewServer()
}

// newSendFDServer - construct a sendfd server
func newSendFDServer() networkservice.NetworkServiceServer {
	return null.NewServer()
}

// newSendFDClient - construct a sendfd server
func newSendFDClient() networkservice.NetworkServiceClient {
	return null.NewClient()
}

// newRecvFDEndpointRegistry - construct a registry server
func newRecvFDEndpointRegistry() registry.NetworkServiceEndpointRegistryServer {
	return null_registry.NewNetworkServiceEndpointRegistryServer()
}
