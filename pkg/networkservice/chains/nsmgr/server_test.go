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
	"os"
	"testing"
	"time"

	interpose_reg "github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	adapters2 "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	next_reg "github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func TokenGenerator(peerAuthInfo credentials.AuthInfo) (tokenValue string, expireTime time.Time, err error) {
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


// NewCrossNSE construct a new Cross connect test NSE
func newCrossNSE(ctx context.Context, name string, connectTo *url.URL, tokenGenerator token.GeneratorFunc, clientDialOptions ...grpc.DialOption) endpoint.Endpoint {
	var crossNSe endpoint.Endpoint
	crossNSe = endpoint.NewServer(
		name,
		authorize.NewServer(),
		tokenGenerator,
		// Statically set the url we use to the unix file socket for the NSMgr
		clienturl.NewServer(connectTo),
		connect.NewServer(
			ctx,
			client.NewClientFactory(
				name,
				// What to call onHeal
				addressof.NetworkServiceClient(adapters.NewServerToClient(crossNSe)),
				tokenGenerator,
			),
			clientDialOptions...,
		),
	)
	return crossNSe
}

func TestNSmgrCrossNSETest(t *testing.T) {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()


	nsmgrReg := &registry.NetworkServiceEndpoint{
		Name: "nsmgr",
		Url:  "tcp://127.0.0.1:5001",
	}

	// Serve NSMGR, Use in memory registry server
	mgr := nsmgr.NewServer(ctx, nsmgrReg, authorize.NewServer(), TokenGenerator, nil, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	nsmURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	mgrGrpcSrv, mgrGrpcCancel, mgrErr := serverNSM(ctx, nsmURL, mgr)
	require.NotNil(t, mgrGrpcSrv)
	require.NotNil(t, mgrErr)
	defer mgrGrpcCancel()
	nsmgrReg.Url = nsmURL.String()
	logrus.Infof("NSMGR listenON: %v", nsmURL.String())

	// Serve endpoint
	nseURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	endpoint.Serve(ctx, nseURL,
		endpoint.NewServer(
			"final-endpoint",
			authorize.NewServer(),
			TokenGenerator),
	)
	logrus.Infof("NSE listenON: %v", nseURL.String())

	// Serve Cross Connect NSE
	crossNSEURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	endpoint.Serve(ctx, crossNSEURL, newCrossNSE(ctx, "cross-nse", nsmURL, TokenGenerator, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.WaitForReady(true))))
	logrus.Infof("Cross NSE listenON: %v", crossNSEURL.String())

	// Register network service.
	serviceReg, err := mgr.NetworkServiceRegistryServer().Register(context.Background(), &registry.NetworkService{
		Name: "my-service",
	})
	require.Nil(t, err)

	// Register endpoints

	// Register NSE
	_, err = mgr.NetworkServiceEndpointRegistryServer().Register(context.Background(), &registry.NetworkServiceEndpoint{
		Url:                 nseURL.String(),
		NetworkServiceNames: []string{serviceReg.Name},
	})
	require.Nil(t, err)

	// Register Cross NSE
	crossRegClient := next_reg.NewNetworkServiceEndpointRegistryClient(interpose_reg.NewNetworkServiceEndpointRegistryClient(), adapters2.NetworkServiceEndpointServerToClient(mgr.NetworkServiceEndpointRegistryServer()))
	_, err = crossRegClient.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Url:  crossNSEURL.String(),
		Name: "cross-nse",
	})
	require.Nil(t, err)

	var nsmClient grpc.ClientConnInterface
	nsmClient, err = newClient(ctx, nsmURL)
	require.Nil(t, err)
	cl := client.NewClient(context.Background(), "nsc-1", nil, TokenGenerator, nsmClient)

	var connection *networkservice.Connection

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service",
			Context:        &networkservice.ConnectionContext{},
		},
	}
	connection, err = cl.Request(ctx, request)
	require.Nil(t, err)
	require.NotNil(t, connection)

	require.Equal(t, 5, len(connection.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.GetConnection().Context = connection.Context
	refreshRequest.GetConnection().Mechanism = connection.Mechanism
	refreshRequest.GetConnection().NetworkServiceEndpointName = connection.NetworkServiceEndpointName

	var connection2 *networkservice.Connection
	connection2, err = cl.Request(ctx, refreshRequest)
	require.Nil(t, err)
	require.NotNil(t, connection2)
	require.Equal(t, 5, len(connection.Path.PathSegments))
}
