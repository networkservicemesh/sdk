// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package mechanismtranslation_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

const (
	clientIfName = "nsm1"
)

func kernelMechanism() *networkservice.Mechanism {
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	}
	_, _ = kernel.NewClient(kernel.WithInterfaceName(clientIfName)).Request(context.Background(), request)
	return request.GetMechanismPreferences()[0]
}

func TestMechanismTranslationClient(t *testing.T) {
	capture := new(captureClient)

	client := next.NewNetworkServiceClient(
		metadata.NewClient(),
		mechanismtranslation.NewClient(),
		capture,
		kernel.NewClient(kernel.WithInterfaceName(clientIfName)),
		adapters.NewServerToClient(
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				kernelmech.MECHANISM: null.NewServer(),
			}),
		),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
			Mechanism: &networkservice.Mechanism{
				Cls:  cls.LOCAL,
				Type: vfio.MECHANISM,
			},
			Context: &networkservice.ConnectionContext{
				ExtraContext: map[string]string{
					"a": "A",
				},
			},
		},
		MechanismPreferences: []*networkservice.Mechanism{{
			Cls:  cls.LOCAL,
			Type: vfio.MECHANISM,
		}},
	}

	// 1. Request

	conn, err := client.Request(context.Background(), request.Clone())
	require.NoError(t, err)
	require.Equal(t, request.GetConnection().String(), conn.String())

	captureRequest := request.Clone()
	captureRequest.MechanismPreferences = nil
	captureRequest.Connection.Mechanism = nil
	require.Equal(t, captureRequest.String(), capture.request.String())

	// 2. Refresh

	conn, err = client.Request(context.Background(), request.Clone())
	require.NoError(t, err)
	require.Equal(t, request.GetConnection().String(), conn.String())

	captureRequest = request.Clone()
	captureRequest.MechanismPreferences = nil
	captureRequest.Connection.Mechanism = kernelMechanism()
	require.Equal(t, captureRequest.String(), capture.request.String())

	// 3. Close

	_, err = client.Close(context.Background(), conn.Clone())
	require.NoError(t, err)

	require.Equal(t, captureRequest.GetConnection().String(), capture.conn.String())
}

func TestMechanismTranslationClient_CloseOnError(t *testing.T) {
	count := 0
	client := next.NewNetworkServiceClient(
		metadata.NewClient(),
		new(afterErrorClient),
		mechanismtranslation.NewClient(),
		checkrequest.NewClient(t, func(t *testing.T, request *networkservice.NetworkServiceRequest) {
			switch count {
			case 0, 2:
				require.Nil(t, request.GetConnection().GetMechanism())
			case 1:
				require.Equal(t, request.GetConnection().GetMechanism().String(), kernelMechanism().String())
			}
			count++
		}),
		kernel.NewClient(kernel.WithInterfaceName(clientIfName)),
		adapters.NewServerToClient(
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				kernelmech.MECHANISM: null.NewServer(),
			}),
		),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "id",
		},
	}

	// 1. Request
	conn, err := client.Request(context.Background(), request.Clone())
	require.NoError(t, err)

	// 2. Refresh Request + Close on error
	request.Connection = conn.Clone()

	_, err = client.Request(context.Background(), request.Clone())
	require.Error(t, err)

	// 3. Refresh Request
	_, err = client.Request(context.Background(), request.Clone())
	require.NoError(t, err)

	require.Equal(t, count, 3)
}

type captureClient struct {
	request *networkservice.NetworkServiceRequest
	conn    *networkservice.Connection
}

func (c *captureClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	c.request = request.Clone()
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *captureClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.conn = conn.Clone()
	return next.Client(ctx).Close(ctx, conn, opts...)
}

type afterErrorClient struct {
	success bool
}

func (c *afterErrorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	c.success = !c.success

	conn, err := next.Client(ctx).Request(ctx, request)
	if err != nil || c.success {
		return conn, err
	}

	_, _ = next.Client(ctx).Close(ctx, conn)

	return nil, errors.New("error")
}

func (c *afterErrorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn)
}
