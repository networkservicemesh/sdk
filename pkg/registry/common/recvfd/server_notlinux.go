// Copyright (c) 2021-2024 Cisco and/or its affiliates.
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

//go:build !linux
// +build !linux

// Package recvfd provides an NSE registry server chain element that:
//  1. Receives and fd over a unix file socket if the nse.URL is an inode://${dev}/${inode} url
//  2. Rewrites the nse.URL to unix:///proc/${pid}/fd/${fd} so it can be used by a normal dialer
package recvfd

import (
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
)

// NewNetworkServiceEndpointRegistryServer - creates new NSE registry chain element that will:
//  1. Receive and fd over a unix file socket if the nse.URL is an inode://${dev}/${inode} url
//  2. Rewrite the nse.URL to unix:///proc/${pid}/fd/${fd} so it can be used by a normal dialer
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return null.NewNetworkServiceEndpointRegistryServer()
}
