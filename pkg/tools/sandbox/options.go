// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type forwarderOptions struct {
	additionalFunctionalityServer []networkservice.NetworkServiceServer
	additionalFunctionalityClient []networkservice.NetworkServiceClient
}

// ForwarderOption is an option to configure a forwarder for sandbox.
type ForwarderOption func(*forwarderOptions)

// WithForwarderAdditionalFunctionalityServer adds an additionalFunctionality to server chain.
func WithForwarderAdditionalFunctionalityServer(a ...networkservice.NetworkServiceServer) ForwarderOption {
	return func(o *forwarderOptions) {
		o.additionalFunctionalityServer = a
	}
}

// WithForwarderAdditionalFunctionalityClient adds an additionalFunctionality to client chain.
func WithForwarderAdditionalFunctionalityClient(a ...networkservice.NetworkServiceClient) ForwarderOption {
	return func(o *forwarderOptions) {
		o.additionalFunctionalityClient = a
	}
}
