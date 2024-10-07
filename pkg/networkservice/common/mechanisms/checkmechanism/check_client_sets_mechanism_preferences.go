// Copyright (c) 2020-2023 Cisco and/or its affiliates.
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

// Package checkmechanism provides TestSuites for use with implementations of a mechanism networkservice chain element.
package checkmechanism

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
)

type checkClientSetsMechanismPreferences struct {
	networkservice.NetworkServiceClient
}

// CheckClientSetsMechanismPreferences - returns a NetworkServiceClient that will check to make sure that the
//
//	clientUnderTest correctly sets the MechanismPreferences to include
//	a mechanism of type mechanismType
//	t - *testing.T for checking
//	clientUnderTest - client we are testing
//	mechanismType - Mechanism.Type implemented by clientUnderTest
//	mechanismCheck - function to check for any parameters that should be present in
//	                 the MechanismPreference
func CheckClientSetsMechanismPreferences(t *testing.T, clientUnderTest networkservice.NetworkServiceClient, mechanismType string, mechanismCheck func(*testing.T, *networkservice.Mechanism)) networkservice.NetworkServiceClient {
	rv := &checkClientSetsMechanismPreferences{}
	rv.NetworkServiceClient = chain.NewNetworkServiceClient(
		clientUnderTest,
		// Check after the clientUnderTest under test is run to make sure we have the right MechanismPreferences set
		checkrequest.NewClient(t,
			func(t *testing.T, req *networkservice.NetworkServiceRequest) {
				var found bool
				for _, mechanism := range req.GetMechanismPreferences() {
					if mechanism.GetType() == mechanismType {
						found = true
						// Run any mechanism specific checks
						mechanismCheck(t, mechanism)
					}
				}
				assert.True(t, found, "Did not find %s Mechanism in MechanismPreferences", mechanismType)
			},
		),
		// Its important to have a working 'server' to return a connection to us
		adapters.NewServerToClient(mechanisms.NewServer(
			map[string]networkservice.NetworkServiceServer{
				mechanismType: null.NewServer(),
			}),
		),
	)
	return rv
}

func (m *checkClientSetsMechanismPreferences) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Make sure we are creating a copy before we change it
	request = request.Clone()
	// Make sure no MechanismPreferences are set
	request.MechanismPreferences = nil
	return m.NetworkServiceClient.Request(ctx, request, opts...)
}
