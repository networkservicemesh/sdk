// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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

package mechanisms_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"gonum.org/v1/gonum/stat/combin"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
)

func server() networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
		memif.MECHANISM:  null.NewServer(),
		kernel.MECHANISM: null.NewServer(),
		srv6.MECHANISM:   null.NewServer(),
		vxlan.MECHANISM:  null.NewServer(),
	}))
}

func request() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Cls:        cls.LOCAL,
				Type:       memif.MECHANISM,
				Parameters: make(map[string]string),
			},
			{
				Cls:        cls.LOCAL,
				Type:       kernel.MECHANISM,
				Parameters: make(map[string]string),
			},
			{
				Cls:        cls.REMOTE,
				Type:       srv6.MECHANISM,
				Parameters: make(map[string]string),
			},
			{
				Cls:        cls.REMOTE,
				Type:       vxlan.MECHANISM,
				Parameters: make(map[string]string),
			},
		},
	}
}

func permuteOverMechanismPreferenceOrder(request *networkservice.NetworkServiceRequest) []*networkservice.NetworkServiceRequest {
	var rv []*networkservice.NetworkServiceRequest
	numMechanism := len(request.GetMechanismPreferences())
	for k := numMechanism; k > 0; k-- {
		permutationGenerator := combin.NewPermutationGenerator(numMechanism, numMechanism)
		for permutationGenerator.Next() {
			permutation := permutationGenerator.Permutation(nil)
			req := request.Clone()
			req.MechanismPreferences = nil
			for _, index := range permutation {
				req.MechanismPreferences = append(req.MechanismPreferences, request.GetMechanismPreferences()[index])
			}
			rv = append(rv, req)
		}
	}
	return rv
}

func TestSelectMechanism(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := server()
	for _, request := range permuteOverMechanismPreferenceOrder(request()) {
		assert.Nil(t, request.GetConnection().GetMechanism(), "SelectMechanismContract requires request.GetConnection().GetMechanism() nil")
		assert.Greater(t, len(request.GetMechanismPreferences()), 0, "serverBasicMechanismContract requires len(request.GetMechanismPreferences()) > 0")
		conn, err := server.Request(context.Background(), request)
		assert.Nil(t, err)
		assert.NotNil(t, conn)
		assert.NotNil(t, conn.GetMechanism())
		assert.Equal(t, request.GetMechanismPreferences()[0].GetCls(), conn.GetMechanism().GetCls(), "Unexpected response to request %+v", request)
		assert.Equal(t, request.GetMechanismPreferences()[0].GetType(), conn.GetMechanism().GetType(), "Unexpected response to request %+v", request)
		_, err = server.Close(context.Background(), conn)
		assert.Nil(t, err)
	}
}

func TestDontSelectMechanismIfSet(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := server()
	for _, request := range permuteOverMechanismPreferenceOrder(request()) {
		request.Connection = &networkservice.Connection{Mechanism: request.GetMechanismPreferences()[len(request.GetMechanismPreferences())-1]}
		assert.NotNil(t, request.GetConnection().GetMechanism())
		assert.Greater(t, len(request.GetMechanismPreferences()), 0, "serverBasicMechanismContract requires len(request.GetMechanismPreferences()) > 0")
		conn, err := server.Request(context.Background(), request)
		assert.Nil(t, err)
		assert.NotNil(t, conn)
		assert.Equal(t, request.GetConnection().GetMechanism(), conn.GetMechanism())
	}
}

func TestUnsupportedMechanismPreference(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	request := request()
	request.MechanismPreferences = []*networkservice.Mechanism{
		{Cls: "NOT_A_CLS", Type: "NOT_A_TYPE"},
	}
	conn, err := server().Request(context.Background(), request)
	assert.Nil(t, conn)
	assert.NotNil(t, err)
	_, err = server().Close(context.Background(), &networkservice.Connection{Mechanism: &networkservice.Mechanism{Cls: "NOT_A_CLS", Type: "NOT_A_TYPE"}})
	assert.NotNil(t, err)
}

func TestUnsupportedMechanism(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	request := request()
	request.GetConnection().Mechanism = &networkservice.Mechanism{
		Cls:  "NOT_A_CLS",
		Type: "NOT_A_TYPE",
	}
	conn, err := server().Request(context.Background(), request)
	assert.Nil(t, conn)
	assert.NotNil(t, err)
	_, err = server().Close(context.Background(), &networkservice.Connection{Mechanism: &networkservice.Mechanism{Cls: "NOT_A_CLS", Type: "NOT_A_TYPE"}})
	assert.NotNil(t, err)
}

func TestDownstreamError(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	request := request()
	request.GetConnection().Mechanism = &networkservice.Mechanism{
		Cls:  cls.LOCAL,
		Type: memif.MECHANISM,
	}
	server := chain.NewNetworkServiceServer(mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
		memif.MECHANISM: injecterror.NewServer(),
	}))
	conn, err := server.Request(context.Background(), request)
	assert.Nil(t, conn)
	assert.NotNil(t, err)
	_, err = server.Close(context.Background(), &networkservice.Connection{Mechanism: &networkservice.Mechanism{Cls: "NOT_A_CLS", Type: "NOT_A_TYPE"}})
	assert.NotNil(t, err)
}

func TestFewWrongMechanisms(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	unsupportedErr := errors.New("unsupported")

	server := next.NewNetworkServiceServer(
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			"mech1": injecterror.NewServer(
				injecterror.WithError(unsupportedErr),
			),
			"mech2": injecterror.NewServer(
				injecterror.WithError(unsupportedErr),
			),
			"mech3": null.NewServer(),
		}),
	)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: "mech1",
			},
			{
				Type: "mech2",
			},
			{
				Type: "mech3",
			},
		},
	}

	conn, err := server.Request(context.Background(), request)
	require.Nil(t, err)

	_, err = server.Close(context.Background(), conn)
	require.Nil(t, err)
}

func TestDontCallNextByItself(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ch := make(chan struct{}, 10)
	server := next.NewNetworkServiceServer(
		server(),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			ch <- struct{}{}
		}),
	)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: &networkservice.Mechanism{
				Type: memif.MECHANISM,
			},
		},
	}

	conn, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, 1, len(ch))

	_, err = server.Close(context.Background(), conn)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(ch))
}

//nolint:dupl
func TestMetrics(t *testing.T) {
	s := server()
	metricsKey := "server_interface"

	for _, request := range permuteOverMechanismPreferenceOrder(request()) {
		request.MechanismPreferences[0].Parameters[ifnameKey] = ifname
		request.Connection.Path = &networkservice.Path{
			PathSegments: make([]*networkservice.PathSegment, 1),
			Index:        0,
		}

		conn, err := s.Request(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, conn.GetPath())
		require.Len(t, conn.GetPath().GetPathSegments(), 1)
		require.NotNil(t, conn.GetPath().GetPathSegments()[0].GetMetrics())

		require.Equal(t, fmt.Sprintf("%s/%s", request.GetMechanismPreferences()[0].GetType(), ifname), conn.GetPath().GetPathSegments()[0].GetMetrics()[metricsKey])
	}
}
