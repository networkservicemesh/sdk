// Copyright (c) 2020 Doc.ai and/or its affiliates.
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
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vfio"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func kernelMechanism() *networkservice.Mechanism {
	request := new(networkservice.NetworkServiceRequest)
	_, _ = kernel.NewClient().Request(context.TODO(), request)
	return request.MechanismPreferences[0]
}

func TestMechanismTranslationClient(t *testing.T) {
	capture := new(captureClient)

	client := next.NewNetworkServiceClient(
		mechanismtranslation.NewClient(),
		capture,
		kernel.NewClient(),
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

	conn, err := client.Request(context.TODO(), request.Clone())
	require.NoError(t, err)
	require.Equal(t, request.Connection.String(), conn.String())

	captureRequest := request.Clone()
	captureRequest.MechanismPreferences = nil
	captureRequest.Connection.Mechanism = nil
	require.Equal(t, captureRequest.String(), capture.request.String())

	// 2. Refresh

	conn, err = client.Request(context.TODO(), request.Clone())
	require.NoError(t, err)
	require.Equal(t, request.Connection.String(), conn.String())

	captureRequest = request.Clone()
	captureRequest.MechanismPreferences = nil
	captureRequest.Connection.Mechanism = kernelMechanism()
	require.Equal(t, captureRequest.String(), capture.request.String())

	// 3. Close

	_, err = client.Close(context.TODO(), conn.Clone())
	require.NoError(t, err)

	require.Equal(t, captureRequest.Connection.String(), capture.conn.String())
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
