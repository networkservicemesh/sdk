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

package healclientfixture

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/test/fakes/fakemonitorconnection"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"google.golang.org/grpc"
)

type ClientRequestFunc func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error)

// TestOnHeal test wrapper for 'networkservice.NetworkServiceClient' implementation
type TestOnHeal struct {
	r ClientRequestFunc
}

// Request calls 'r' with passed arguments
func (t *TestOnHeal) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return t.r(ctx, in, opts...)
}

// Close has no implementation yet
func (t *TestOnHeal) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	panic("implement me")
}

// callNotifier sends struct{}{} to notifier channel when h called
func callNotifier(notifier chan struct{}, h ClientRequestFunc) ClientRequestFunc {
	return func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (i *networkservice.Connection, e error) {
		notifier <- struct{}{}
		if h != nil {
			return h(ctx, in, opts...)
		}
		return nil, nil
	}
}

// Fixture is auxiliary class to test 'healClient'
type Fixture struct {
	// Client that we are going to test
	Client networkservice.NetworkServiceClient

	// ServerStream is server stream used to send 'networkservice.ConnectionEvent' to client
	ServerStream networkservice.MonitorConnection_MonitorConnectionsServer

	// CloseStream is function that stops server streaming
	CloseStream func()

	// OnHeal pointer to test chain that will be used in case of healing
	OnHeal *TestOnHeal

	// OnHealNotifierCh receives struct{}{} every time when 'OnHeal' called
	OnHealNotifierCh chan struct{}

	// FakeMonitorServer fake implementation of MonitorConnectionServer, passed to constructor of 'healClient'
	FakeMonitorServer *fakemonitorconnection.FakeMonitorServer

	// Request that passed to Request during setup
	Request *networkservice.NetworkServiceRequest

	// Conn is connection received as a response of Request
	Conn *networkservice.Connection
}

// SetupWithSingleRequest initialize Fixture and calls Request with 'request' passed
func SetupWithSingleRequest(f *Fixture) error {
	f.FakeMonitorServer = fakemonitorconnection.New()

	monitorClient, err := f.FakeMonitorServer.Client(context.Background())
	if err != nil {
		return err
	}

	f.OnHeal = &TestOnHeal{}
	f.OnHealNotifierCh = make(chan struct{})
	f.OnHeal.r = callNotifier(f.OnHealNotifierCh, f.OnHeal.r)

	healClient := heal.NewClient(monitorClient, addressof.NetworkServiceClient(f.OnHeal))
	f.Client = chain.NewNetworkServiceClient(healClient)

	f.Conn, err = f.Client.Request(context.Background(), f.Request)
	if err != nil {
		return err
	}

	f.ServerStream, f.CloseStream, err = f.FakeMonitorServer.Stream(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// TearDown releases resources allocated during SetupWithSingleRequest
func TearDown(f *Fixture) {
	_, _ = f.Client.Close(context.Background(), f.Conn)
	f.CloseStream()
	f.FakeMonitorServer.Close()
}
