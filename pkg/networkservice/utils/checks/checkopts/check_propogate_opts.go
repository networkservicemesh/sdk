// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package checkopts - provides tools for checking to make sure that grpc.CallOptions are propagated through a client chain elements
package checkopts

import (
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectopts"
)

// CheckPropogateOptsClient - returns NetworkServiceClient that checks to make sure the clientUnderTest is passing through the grpc.CallOptions.
func CheckPropogateOptsClient(t *testing.T, clientUnderTest networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		injectopts.NewClient(),
		clientUnderTest,
		NewClient(t),
	)
}
