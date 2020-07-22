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

// Package nsmgr_test define a tests for NSMGR chain element.
package nsmgr_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/setextracontext"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func TokenGenerator(peerAuthInfo credentials.AuthInfo) (token string, expireTime time.Time, err error) {
	return "TestToken", time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC), nil
}

func serverNSM(ctx context.Context, listenOn *url.URL, server nsmgr.Nsmgr) (*grpc.Server, context.CancelFunc, <-chan error) {
	s := grpc.NewServer()
	server.Register(s)

	cancelCtx, cancel := context.WithCancel(ctx)
	errorChan := grpcutils.ListenAndServe(cancelCtx, listenOn, s)
	return s, cancel, errorChan
}

func newClient(ctx context.Context, u *url.URL) (*grpc.ClientConn, error) {
	clientCtx, clientCancelFunc := context.WithTimeout(ctx, 10*time.Second)
	defer clientCancelFunc()
	return grpc.DialContext(clientCtx, grpcutils.URLToTarget(u),
		grpc.WithInsecure(),
		grpc.WithBlock())
}
func TestNSmgrEndpointCallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Serve endpoint
	nseURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	_ = endpoint.Serve(ctx, nseURL, endpoint.NewServer(ctx, "test-nse", authorize.NewServer(), TokenGenerator, setextracontext.NewServer(map[string]string{"perform": "ok"})))
	logrus.Infof("NSE listenON: %v", nseURL.String())

	nsmgrReg := &registry.NetworkServiceEndpoint{
		Name: "nsmgr",
		Url:  "tcp://127.0.0.1:5001",
	}

	// Server NSMGR, Use in memory registry server
	mgr := nsmgr.NewServer(ctx, nsmgrReg, authorize.NewServer(), TokenGenerator, nil, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	nsmURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	mgrGrpcSrv, mgrGrpcCancel, mgrErr := serverNSM(ctx, nsmURL, mgr)
	require.NotNil(t, mgrGrpcSrv)
	require.NotNil(t, mgrErr)
	defer mgrGrpcCancel()
	nsmgrReg.Url = nsmURL.String()
	logrus.Infof("NSMGR listenON: %v", nsmURL.String())

	// Register network service.
	nsService, err := mgr.NetworkServiceRegistryServer().Register(context.Background(), &registry.NetworkService{
		Name: "my-service",
	})
	require.Nil(t, err)

	// Register Endpoint
	var nseReg *registry.NetworkServiceEndpoint
	nseReg, err = mgr.NetworkServiceEndpointRegistryServer().Register(context.Background(), &registry.NetworkServiceEndpoint{
		NetworkServiceNames: []string{nsService.Name},
		Url:                 nseURL.String(),
	})
	require.Nil(t, err)
	require.NotEqual(t, "", nseReg.Name)
	require.NotNil(t, nseReg)

	var nsmClient grpc.ClientConnInterface
	nsmClient, err = newClient(ctx, nsmURL)
	require.Nil(t, err)
	cl := client.NewClient(context.Background(), "nsc-1", nil, TokenGenerator, nsmClient)

	var connection *networkservice.Connection

	connection, err = cl.Request(ctx, &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service",
			Context:        &networkservice.ConnectionContext{},
		},
	})
	require.Nil(t, err)
	require.NotNil(t, connection)
	require.Equal(t, 3, len(connection.Path.PathSegments))
}
