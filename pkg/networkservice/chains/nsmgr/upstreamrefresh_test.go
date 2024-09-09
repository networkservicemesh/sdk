// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package nsmgr_test

import (
	"context"
	"testing"
	"time"

	"github.com/edwarnicke/genericsync"
	"github.com/google/uuid"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/upstreamrefresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func Test_UpstreamRefreshClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("my-service"))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

	// This NSE will send REFRESH_REQUESTED events if mtu will be changed
	counter := new(count.Server)
	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		newRefreshMTUSenderServer(),
		counter,
	)

	// Create the first client
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(upstreamrefresh.NewClient(ctx)))
	reqCtx, reqClose := context.WithTimeout(ctx, time.Second)
	defer reqClose()

	req := defaultRequest(nsReg.GetName())
	req.Connection.Id = uuid.New().String()
	req.GetConnection().GetContext().MTU = defaultMtu

	conn, err := nsc.Request(reqCtx, req)
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	// Create the second client
	nsc2 := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(upstreamrefresh.NewClient(ctx)))
	reqCtx2, reqClose2 := context.WithTimeout(ctx, time.Second)
	defer reqClose2()

	// Change MTU for the second client
	req2 := defaultRequest(nsReg.GetName())
	req2.Connection.Id = uuid.New().String()
	req2.GetConnection().GetContext().MTU = 1000

	conn2, err := nsc2.Request(reqCtx2, req2)
	require.NoError(t, err)
	require.Equal(t, 2, counter.UniqueRequests())

	// The request from the second client should trigger refresh on the first one
	require.Eventually(t, func() bool { return counter.Requests() == 3 }, timeout, tick)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	_, err = nsc.Close(ctx, conn2)
	require.NoError(t, err)
}

func Test_UpstreamRefreshClient_LocalNotifications(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("my-service"))
	require.NoError(t, err)

	// Create the first NSE
	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint1",
		NetworkServiceNames: []string{nsReg.GetName()},
	}
	counter1 := new(count.Server)
	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		newRefreshMTUSenderServer(),
		counter1,
	)

	// Create the second NSE
	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint2",
		NetworkServiceNames: []string{nsReg.GetName()},
	}
	counter2 := new(count.Server)
	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg2,
		sandbox.GenerateTestToken,
		newRefreshMTUSenderServer(),
		counter2,
	)

	// Create the client
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(upstreamrefresh.NewClient(ctx, upstreamrefresh.WithLocalNotifications())))

	// Send request --> NSE1
	reqCtx, reqClose := context.WithTimeout(ctx, time.Second)
	defer reqClose()

	req := defaultRequest(nsReg.GetName())
	req.Connection.Id = "1"
	req.GetConnection().NetworkServiceEndpointName = nseReg.GetName()
	req.GetConnection().GetContext().MTU = defaultMtu

	conn, err := nsc.Request(reqCtx, req)
	require.NoError(t, err)
	require.Equal(t, 1, counter1.UniqueRequests())

	// Send request2 --> NSE2
	reqCtx2, reqClose2 := context.WithTimeout(ctx, time.Second)
	defer reqClose2()

	req2 := defaultRequest(nsReg.GetName())
	req2.Connection.Id = "2"
	req2.GetConnection().NetworkServiceEndpointName = nseReg2.GetName()
	req2.GetConnection().GetContext().MTU = defaultMtu

	conn2, err := nsc.Request(reqCtx2, req2)
	require.NoError(t, err)
	require.Equal(t, 1, counter2.UniqueRequests())

	// Send request3 --> NSE1 with different MTU
	reqCtx3, reqClose3 := context.WithTimeout(ctx, time.Second)
	defer reqClose3()

	req3 := defaultRequest(nsReg.GetName())
	req3.Connection.Id = "3"
	req3.GetConnection().NetworkServiceEndpointName = nseReg.GetName()
	req3.GetConnection().GetContext().MTU = 1000

	conn3, err := nsc.Request(reqCtx3, req3)
	require.NoError(t, err)
	require.Equal(t, 2, counter1.UniqueRequests())

	// Third request should trigger the first and the second to refresh their connections even if they connected to different endpoints
	require.Eventually(t, func() bool { return counter2.Requests() == 2 }, timeout, tick)
	require.Equal(t, 1, counter2.UniqueRequests())

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
	_, err = nsc.Close(ctx, conn2)
	require.NoError(t, err)
	_, err = nsc.Close(ctx, conn3)
	require.NoError(t, err)
}

// This test shows a case when the event monitor is faster than the backward request.
func Test_UpstreamRefreshClientDelay(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("my-service"))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.GetName())

	// This NSE will send REFRESH_REQUESTED events
	// Channels coordinate the test to send the event at the right time
	counter := new(count.Server)
	ch1 := make(chan struct{}, 1)
	ch2 := make(chan struct{}, 1)
	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		newRefreshSenderServer(ch1, ch2),
		counter,
	)

	// Create the client that has slow request processing
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(
		upstreamrefresh.NewClient(ctx),
		newSlowHandlerClient(ch1, ch2)),
	)

	reqCtx, reqClose := context.WithTimeout(ctx, time.Second)
	defer reqClose()

	req := defaultRequest(nsReg.GetName())
	req.Connection.Id = uuid.New().String()

	conn, err := nsc.Request(reqCtx, req)
	require.NoError(t, err)
	require.Equal(t, 1, counter.UniqueRequests())

	// Eventually we should see a refresh
	require.Eventually(t, func() bool { return counter.Requests() == 2 }, timeout, tick)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}

type refreshMTUSenderServer struct {
	m   *genericsync.Map[string, *networkservice.Connection]
	mtu uint32
}

const defaultMtu = 9000

func newRefreshMTUSenderServer() *refreshMTUSenderServer {
	return &refreshMTUSenderServer{
		m:   new(genericsync.Map[string, *networkservice.Connection]),
		mtu: defaultMtu,
	}
}

func (r *refreshMTUSenderServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	if conn.GetContext().GetMTU() != r.mtu {
		if _, ok := r.m.Load(conn.GetId()); ok {
			return conn, err
		}
		ec, _ := monitor.LoadEventConsumer(ctx, false)

		connectionsToSend := make(map[string]*networkservice.Connection)
		r.m.Range(func(k string, v *networkservice.Connection) bool {
			connectionsToSend[k] = v.Clone()
			connectionsToSend[k].State = networkservice.State_REFRESH_REQUESTED
			return true
		})

		_ = ec.Send(&networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_UPDATE,
			Connections: connectionsToSend,
		})
	}
	r.m.Store(conn.GetId(), conn.Clone())

	return conn, err
}

func (r *refreshMTUSenderServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

type refreshSenderServer struct {
	waitSignalCh <-chan struct{}
	eventSentCh  chan<- struct{}

	eventWasSent bool
}

func newRefreshSenderServer(waitSignalCh <-chan struct{}, eventSentCh chan<- struct{}) *refreshSenderServer {
	return &refreshSenderServer{
		waitSignalCh: waitSignalCh,
		eventSentCh:  eventSentCh,
	}
}

func (r *refreshSenderServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if !r.eventWasSent {
		ec, _ := monitor.LoadEventConsumer(ctx, false)

		c := conn.Clone()
		c.State = networkservice.State_REFRESH_REQUESTED

		go func() {
			<-r.waitSignalCh
			_ = ec.Send(&networkservice.ConnectionEvent{
				Type:        networkservice.ConnectionEventType_UPDATE,
				Connections: map[string]*networkservice.Connection{c.GetId(): c},
			})
			time.Sleep(time.Millisecond * 100)
			r.eventWasSent = true
			r.eventSentCh <- struct{}{}
		}()
	} else {
		r.eventSentCh <- struct{}{}
	}
	return conn, err
}

func (r *refreshSenderServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

type slowHandlerClient struct {
	startSlowHandlingCh  chan<- struct{}
	finishSlowHandlingCh <-chan struct{}
}

func newSlowHandlerClient(startSlowHandlingCh chan<- struct{}, finishSlowHandlingCh <-chan struct{}) networkservice.NetworkServiceClient {
	return &slowHandlerClient{
		startSlowHandlingCh:  startSlowHandlingCh,
		finishSlowHandlingCh: finishSlowHandlingCh,
	}
}

func (s *slowHandlerClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	s.startSlowHandlingCh <- struct{}{}
	<-s.finishSlowHandlingCh
	return conn, err
}

func (s *slowHandlerClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
