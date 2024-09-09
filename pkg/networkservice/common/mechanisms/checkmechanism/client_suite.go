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

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkerror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkopts"
)

// ClientSuite - test suite to check that a NetworkServiceClient implementing a Mechanism meets basic contracts.
type ClientSuite struct {
	suite.Suite
	clientUnderTest      networkservice.NetworkServiceClient
	configureContext     func(ctx context.Context) context.Context
	mechanismType        string
	mechanismCheck       func(*testing.T, *networkservice.Mechanism)
	contextOnReturnCheck func(*testing.T, context.Context)
	contextCheck         func(*testing.T, context.Context)
	Request              *networkservice.NetworkServiceRequest
	ConnClose            *networkservice.Connection
}

// NewClientSuite - returns a ClientTestSuite
//
//	clientUnderTest - the client we are testing to make sure it correctly implements a Mechanism
//	configureContext - a function that is applied to context.Background to make sure anything
//	                   needed by the clientUnderTest is present in the context.Context
//	mechanismType - Mechanism.Type implemented by the clientUnderTest
//	mechanismCheck - function to check that the clientUnderTest has properly added Mechanism.Parameters
//	                 to the Mechanism it has appended to MechanismPreferences
//	contextOnReturnCheck - function to check that the context.Context *after* clientUnderTest has returned are correct
//	contextCheck - function to check that the clientUnderTest introduces no side effects to the context.Context
//	               before calling the next element in the chain
//	request - NetworkServiceRequest to be used for testing
//	connClose - Connection to be used for testing Close(...)
func NewClientSuite(
	clientUnderTest networkservice.NetworkServiceClient,
	configureContext func(ctx context.Context) context.Context,
	mechanismType string,
	mechanismCheck func(*testing.T, *networkservice.Mechanism),
	contextOnReturnCheck,
	contextCheck func(*testing.T, context.Context),

	request *networkservice.NetworkServiceRequest,
	connClose *networkservice.Connection,
) *ClientSuite {
	return &ClientSuite{
		clientUnderTest:      clientUnderTest,
		configureContext:     configureContext,
		mechanismType:        mechanismType,
		mechanismCheck:       mechanismCheck,
		contextOnReturnCheck: contextOnReturnCheck,
		contextCheck:         contextCheck,
		Request:              request,
		ConnClose:            connClose,
	}
}

// TestPropagatesError - tests that the clientUnderTest returns an error when the next element in the chain returned an error to it.
func (m *ClientSuite) TestPropagatesError() {
	check := checkerror.CheckPropagatesErrorClient(m.T(), m.clientUnderTest)
	_, err := check.Request(m.configureContext(context.Background()), m.Request.Clone())
	assert.NotNil(m.T(), err)
	_, err = check.Close(m.configureContext(context.Background()), m.ConnClose)
	assert.NotNil(m.T(), err)
}

// TestPropogatesOpts - tests that the clientUnderTest passes along the grpc.CallOptions to the next client in the chain.
func (m *ClientSuite) TestPropogatesOpts() {
	contract := checkopts.CheckPropogateOptsClient(m.T(), m.clientUnderTest)
	_, err := contract.Request(m.configureContext(context.Background()), m.Request.Clone())
	assert.Nil(m.T(), err)
	_, err = contract.Close(m.configureContext(context.Background()), m.ConnClose)
	assert.Nil(m.T(), err)
}

// TestSetsMechanismPreference - test that the clientUnderTest correctly sets MechanismPreferences for the Mechanism it implements
//
//	before calling the next element in the chain
func (m *ClientSuite) TestSetsMechanismPreference() {
	check := CheckClientSetsMechanismPreferences(m.T(),
		m.clientUnderTest,
		m.mechanismType,
		m.mechanismCheck,
	)
	conn, err := check.Request(m.configureContext(context.Background()), m.Request.Clone())
	assert.Nil(m.T(), err)
	_, err = check.Close(m.configureContext(context.Background()), conn)
	assert.Nil(m.T(), err)
	_, err = check.Close(m.configureContext(context.Background()), m.ConnClose)
	assert.Nil(m.T(), err)
}

// TestContextOnReturn - test that the clientUnderTest has the correct side effects on the context.Context when it returns.
func (m *ClientSuite) TestContextOnReturn() {
	contract := CheckClientContextOnReturn(m.T(),
		m.clientUnderTest,
		m.mechanismType,
		m.contextOnReturnCheck,
	)
	_, err := contract.Request(m.configureContext(context.Background()), m.Request.Clone())
	assert.Nil(m.T(), err)
	_, err = contract.Close(m.configureContext(context.Background()), m.ConnClose)
	assert.Nil(m.T(), err)
}

// TestContextAfter - tests that the clientUnderTest has no side effects on the context.Context before it calls the next element in the chain.
func (m *ClientSuite) TestContextAfter() {
	contract := CheckClientContextAfter(m.T(),
		m.clientUnderTest,
		m.mechanismType,
		m.contextCheck,
	)
	_, err := contract.Request(m.configureContext(context.Background()), m.Request.Clone())
	assert.Nil(m.T(), err)
	_, err = contract.Close(m.configureContext(context.Background()), m.ConnClose)
	assert.Nil(m.T(), err)
}
