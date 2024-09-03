// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkerror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
)

// ServerSuite - test suite to check that a NetworkServiceServer implementing a Mechanism meets basic contracts.
type ServerSuite struct {
	suite.Suite
	serverUnderTest  networkservice.NetworkServiceServer
	configureContext func(ctx context.Context) context.Context
	mechanismType    string
	mechanismCheck   func(*testing.T, *networkservice.Mechanism)
	contextCheck     func(*testing.T, context.Context)
	Request          *networkservice.NetworkServiceRequest
	ConnClose        *networkservice.Connection
}

// NewServerSuite - returns a ServerSuite
//
//	serverUnderTest - the server being tested to make sure it correctly implements a Mechanism
//	configureContext - a function that is applied to context.Background to make sure anything
//	                   needed by the serverUnderTest is present in the context.Context
//	mechanismType - Mechanism.Type implemented by the serverUnderTest
//	contextCheck - function to check that the serverUnderTest introduces all needed side effects to the context.Context
//	               before calling the next element in the chain
//	request - NetworkServiceRequest to be used for testing
//	connClose - Connection to be used for testing Close(...)
func NewServerSuite(
	serverUnderTest networkservice.NetworkServiceServer,
	configureContext func(ctx context.Context) context.Context,
	mechanismType string,
	mechanismCheck func(*testing.T, *networkservice.Mechanism),
	contextCheck func(*testing.T, context.Context),
	request *networkservice.NetworkServiceRequest,
	connClose *networkservice.Connection,
) *ServerSuite {
	return &ServerSuite{
		serverUnderTest:  serverUnderTest,
		configureContext: configureContext,
		mechanismType:    mechanismType,
		mechanismCheck:   mechanismCheck,
		contextCheck:     contextCheck,
		Request:          request,
		ConnClose:        connClose,
	}
}

// TestPropagatesError - tests that the serverUnderTest returns an error when the next element in the chain returned an error to it.
func (m *ServerSuite) TestPropagatesError() {
	contract := checkerror.CheckPropogatesErrorServer(m.T(), m.serverUnderTest)
	_, err := contract.Request(m.configureContext(context.Background()), m.Request.Clone())
	assert.NotNil(m.T(), err)
	_, err = contract.Close(m.configureContext(context.Background()), m.ConnClose)
	assert.NotNil(m.T(), err)
}

// TestContextAfter - tests that the serverUnderTest has all needed side effects on the context.Context before it calls the next element in the chain.
func (m *ServerSuite) TestContextAfter() {
	contract := CheckContextAfterServer(m.T(),
		m.serverUnderTest,
		m.mechanismType,
		m.contextCheck,
	)
	_, err := contract.Request(m.configureContext(context.Background()), m.Request.Clone())
	assert.Nil(m.T(), err)
	_, err = contract.Close(m.configureContext(context.Background()), m.ConnClose)
	assert.Nil(m.T(), err)
}

// TestMechanismAfter - test to make sure that any changes the server should have made to the Request.Connection.Mechanism
//
//	before calling the next chain element have been made
func (m *ServerSuite) TestMechanismAfter() {
	contract := chain.NewNetworkServiceServer(
		mechanisms.NewServer(
			map[string]networkservice.NetworkServiceServer{
				m.mechanismType: m.serverUnderTest,
			},
		),
		checkrequest.NewServer(m.T(), func(t *testing.T, request *networkservice.NetworkServiceRequest) {
			mechanism := request.GetConnection().GetMechanism().Clone()
			m.mechanismCheck(t, mechanism)
		}),
	)
	_, err := contract.Request(m.configureContext(context.Background()), m.Request.Clone())
	assert.Nil(m.T(), err)
	_, err = contract.Close(m.configureContext(context.Background()), m.ConnClose)
	assert.Nil(m.T(), err)
}
