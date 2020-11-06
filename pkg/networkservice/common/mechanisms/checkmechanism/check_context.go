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

package checkmechanism

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontextonreturn"
)

// CheckClientContextOnReturn - returns a NetworkServiceClient that will check the state of the context.Context
//                              after the clientUnderTest has returned
//                              t - *testing.T for checks
//                              clientUnderTest - client we are testing - presumed to implement a mechanism
//                              mechanismType - Mechanism.Type implemented by the clientUnderTest
//                              check - function to check the state of the context.Context after the clientUnderTest has returned
func CheckClientContextOnReturn(t *testing.T, clientUnderTest networkservice.NetworkServiceClient, mechanismType string, check func(*testing.T, context.Context)) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		checkcontextonreturn.NewClient(t, check),
		clientUnderTest,
		adapters.NewServerToClient(mechanisms.NewServer(
			map[string]networkservice.NetworkServiceServer{
				mechanismType: null.NewServer(),
			})),
	)
}

// CheckClientContextAfter - returns a NetworkServiceClient that will check the state of the context.Context after it has left
//                           the clientUnderTest.  Note: it should almost always be the case that there are no side effects
//                           related to implementing the Mechanism at this point, as the clientUnderTest can't know what to
//                           do until after the elements after it have returned a fully complete Connection.
//                           t - *testing.T for checks
//                           clientUnderTest - client we are testing - presumed to implement a mechanism
//                           mechanismType - Mechanism.Type implemented by the clientUnderTest
//                           check - function to check that the clientUnderTest has not taken action to implement the Mechanism
//                                   (as it cannot know what to do at this stage)
func CheckClientContextAfter(t *testing.T, client networkservice.NetworkServiceClient, mechanismType string, check func(*testing.T, context.Context)) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		client,
		checkcontext.NewClient(t, check),
		adapters.NewServerToClient(mechanisms.NewServer(
			map[string]networkservice.NetworkServiceServer{
				mechanismType: null.NewServer(),
			})),
	)
}

// CheckContextAfterServer - returns a NetworkServiceServer that will check the state of the context.Context after it has
//                           left the serverUnderTest.  At this time the context.Context should contain any side effects needed
//                           to implement the Mechanism, as it should have a complete Connection to work with.
//                           t - *testing.T for checks
//                           serverUnderTest - server we are testing - presumed to implement a mechanism
//                           mechanismType - Mechanism.Type implemented by the serverUnderTest
//                           check - function to check that the serverUnderTest has taken action to implement the Mechanism
func CheckContextAfterServer(t *testing.T, serverUnderTest networkservice.NetworkServiceServer, mechanismType string, check func(*testing.T, context.Context)) networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		mechanisms.NewServer(
			map[string]networkservice.NetworkServiceServer{
				mechanismType: serverUnderTest,
			},
		),
		checkcontext.NewServer(t, check),
	)
}
