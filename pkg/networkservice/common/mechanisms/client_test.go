// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
)

func client() networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
		memif.MECHANISM:  null.NewClient(),
		kernel.MECHANISM: null.NewClient(),
		srv6.MECHANISM:   null.NewClient(),
		vxlan.MECHANISM:  null.NewClient(),
	}))
}

func Test_Client_DontSelectMechanismIfSet(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	c := client()
	for _, request := range permuteOverMechanismPreferenceOrder(request()) {
		request.Connection = &networkservice.Connection{Mechanism: request.GetMechanismPreferences()[len(request.GetMechanismPreferences())-1]}
		assert.NotNil(t, request.GetConnection().GetMechanism())
		assert.Greater(t, len(request.GetMechanismPreferences()), 0, "serverBasicMechanismContract requires len(request.GetMechanismPreferences()) > 0")
		conn, err := c.Request(context.Background(), request)
		assert.Nil(t, err)
		assert.NotNil(t, conn)
	}
}

func Test_Client_UnsupportedMechanismPreference(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	request := request()
	request.MechanismPreferences = []*networkservice.Mechanism{
		{Cls: "NOT_A_CLS", Type: "NOT_A_TYPE"},
	}
	conn, err := client().Request(context.Background(), request)
	assert.Nil(t, conn)
	assert.NotNil(t, err)
	_, err = client().Close(context.Background(), &networkservice.Connection{Mechanism: &networkservice.Mechanism{Cls: "NOT_A_CLS", Type: "NOT_A_TYPE"}})
	assert.NotNil(t, err)
}

func Test_Client_UnsupportedMechanism(t *testing.T) {
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

func Test_Client_DownstreamError(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	request := request()
	request.GetConnection().Mechanism = &networkservice.Mechanism{
		Cls:  cls.LOCAL,
		Type: memif.MECHANISM,
	}
	client := chain.NewNetworkServiceClient(mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
		memif.MECHANISM: injecterror.NewClient(),
	}))
	conn, err := client.Request(context.Background(), request)
	assert.Nil(t, conn)
	assert.NotNil(t, err)
	_, err = client.Close(context.Background(), &networkservice.Connection{Mechanism: &networkservice.Mechanism{Cls: "NOT_A_CLS", Type: "NOT_A_TYPE"}})
	assert.NotNil(t, err)
}

func Test_Client_FewWrongMechanisms(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	unsupportedErr := errors.New("unsupported")

	c := next.NewNetworkServiceClient(
		mechanisms.NewClient(map[string]networkservice.NetworkServiceClient{
			"mech1": injecterror.NewClient(
				injecterror.WithError(unsupportedErr),
			),
			"mech2": injecterror.NewClient(
				injecterror.WithError(unsupportedErr),
			),
			"mech3": null.NewClient(),
		}),
		adapters.NewServerToClient(mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			"mech3": null.NewServer(),
		})),
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

	conn, err := c.Request(context.Background(), request)
	require.Nil(t, err)

	_, err = c.Close(context.Background(), conn)
	require.Nil(t, err)
}

func Test_Client_DontCallNextByItself(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ch := make(chan struct{}, 10)
	c := next.NewNetworkServiceClient(
		client(),
		checkcontext.NewClient(t, func(t *testing.T, ctx context.Context) {
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

	conn, err := c.Request(context.Background(), request, grpc.WaitForReady(true))
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	assert.Equal(t, 1, len(ch))

	_, err = c.Close(context.Background(), conn, grpc.WaitForReady(true))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(ch))
}

const (
	ifnameKey = "name"
	ifname    = "nsm-1"
)

//nolint:dupl
func Test_Client_Metrics(t *testing.T) {
	c := client()
	metricsKey := "client_interface"

	for _, request := range permuteOverMechanismPreferenceOrder(request()) {
		request.MechanismPreferences[0].Parameters[ifnameKey] = ifname
		request.Connection.Path = &networkservice.Path{
			PathSegments: make([]*networkservice.PathSegment, 1),
			Index:        0,
		}

		conn, err := c.Request(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, conn.GetPath())
		require.Len(t, conn.GetPath().GetPathSegments(), 1)
		require.NotNil(t, conn.GetPath().GetPathSegments()[0].GetMetrics())
		require.Equal(t, fmt.Sprintf("%s/%s", request.GetMechanismPreferences()[0].GetType(), ifname), conn.GetPath().GetPathSegments()[0].GetMetrics()[metricsKey])
	}
}
