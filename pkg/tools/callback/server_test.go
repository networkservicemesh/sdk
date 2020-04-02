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

package callback_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/callback"
)

type nsmTestImpl struct {
}

func (n nsmTestImpl) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.GetConnection().Labels = map[string]string{"LABEL": "Demo1"}
	// Send Back
	return request.GetConnection(), nil
}

func (n nsmTestImpl) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return nil, nil
}

type idHolder struct {
	ids chan string
}

func (i *idHolder) ClientConnected(id string) {
	i.ids <- id
}

func (i *idHolder) ClientDisconnected(id string) {
	i.ids <- "-" + id
}

func TestServerClientInvoke(t *testing.T) {
	// Construct a Server GRPC server
	server := grpc.NewServer()

	listener, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer func() { _ = listener.Close() }()
	defer server.Stop()

	// Register callback GRPC endpoint and callback.Server will hold callback clients to be callable.
	callbackServer := callback.NewServer(callback.IdentityByAuthority)
	callback.RegisterCallbackServiceServer(server, callbackServer)
	go func() {
		_ = server.Serve(listener)
	}()

	// Client stuff setup, construct a client GRPC server a standard way
	nseTest := &nsmTestImpl{}

	clientGRPC := grpc.NewServer()
	defer clientGRPC.Stop()
	networkservice.RegisterNetworkServiceServer(clientGRPC, nseTest)

	// Connect from Client to Server
	var client *grpc.ClientConn

	client, err = grpc.DialContext(context.Background(), listener.Addr().String(), grpc.WithBlock(), grpc.WithInsecure(), grpc.WithAuthority("my_client"))
	require.Nil(t, err)
	// Construct a callback client.
	callbackClient := callback.NewClient(client, clientGRPC)
	callbackClient.Serve(context.Background())

	ids := &idHolder{
		ids: make(chan string),
	}
	callbackServer.AddListener(ids)

	clientID := ""
	select {
	case errValue := <-callbackClient.ErrChan():
		require.Errorf(t, errValue, "Unexpected error")
	case <-time.After(1 * time.Second):
		require.Fail(t, "Timeout waiting for client")
	case clientID = <-ids.ids:
		require.Equal(t, "my_client", clientID)
	}

	nsmClientGRPC, err3 := callbackServer.NewClient(context.Background(), clientID)
	require.Nil(t, err3)
	nsmClient := networkservice.NewNetworkServiceClient(nsmClientGRPC)
	resp, err2 := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "qwe",
		},
	})
	require.Nil(t, err2)
	require.NotNil(t, resp)
	require.Equal(t, 1, len(resp.Labels))
	require.Equal(t, "Demo1", resp.Labels["LABEL"])
}

func BenchmarkServerClientInvoke(b *testing.B) {
	// Construct a Server GRPC server
	server := grpc.NewServer()
	sockFile := callback.TempSocketFile()
	listener, err := net.Listen("unix", sockFile)
	defer func() { _ = os.Remove(sockFile) }()
	require.Nil(b, err)
	defer func() { _ = listener.Close() }()
	defer server.Stop()

	// Register callback GRPC endpoint and callback.Server will hold callback clients to be callable.
	callbackServer := callback.NewServer(callback.IdentityByAuthority)
	callback.RegisterCallbackServiceServer(server, callbackServer)
	go func() {
		_ = server.Serve(listener)
	}()

	// Client stuff setup, construct a client GRPC server a standard way
	nseTest := &nsmTestImpl{}
	clientGRPC := grpc.NewServer()
	defer clientGRPC.Stop()
	networkservice.RegisterNetworkServiceServer(clientGRPC, nseTest)
	// Connect from Client to Server
	var client *grpc.ClientConn
	client, err = grpc.DialContext(context.Background(), "unix:"+sockFile, grpc.WithInsecure(), grpc.WithAuthority("my_client"))
	require.Nil(b, err)
	// Construct a callback client.
	callbackClient := callback.NewClient(client, clientGRPC)
	callbackClient.Serve(context.Background())
	ids := &idHolder{
		ids: make(chan string),
	}
	callbackServer.AddListener(ids)

	clientID := ""
	select {
	case errValue := <-callbackClient.ErrChan():
		require.Errorf(b, errValue, "Unexpected error")
	case <-time.After(1 * time.Second):
		require.Fail(b, "Timeout waiting for client")
	case clientID = <-ids.ids:
		require.Equal(b, "my_client", clientID)
	}

	nsmClientGRPC, err3 := callbackServer.NewClient(context.Background(), clientID)
	require.Nil(b, err3)
	nsmClient := networkservice.NewNetworkServiceClient(nsmClientGRPC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err2 := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "qwe",
			},
		})
		require.Nil(b, err2)
		require.NotNil(b, resp)
		require.Equal(b, 1, len(resp.Labels))
		require.Equal(b, "Demo1", resp.Labels["LABEL"])
	}
}

func BenchmarkServerClientInvokePure(b *testing.B) {
	// Construct a Server GRPC server
	server := grpc.NewServer()
	sockFile := callback.TempSocketFile()
	listener, err := net.Listen("unix", sockFile)
	defer func() { _ = os.Remove(sockFile) }()
	require.Nil(b, err)
	defer func() { _ = listener.Close() }()
	defer server.Stop()

	// Register callback GRPC endpoint and callback.Server will hold callback clients to be callable.
	// Client stuff setup, construct a client GRPC server a standard way
	nseTest := &nsmTestImpl{}
	networkservice.RegisterNetworkServiceServer(server, nseTest)

	go func() {
		_ = server.Serve(listener)
	}()

	// Connect from Client to Server
	var client *grpc.ClientConn
	client, err = grpc.DialContext(context.Background(), "unix:"+sockFile, grpc.WithInsecure(), grpc.WithAuthority("my_client"), grpc.WithBlock())
	require.Nil(b, err)
	nsmClient := networkservice.NewNetworkServiceClient(client)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err2 := nsmClient.Request(context.Background(), &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "qwe",
			},
		})
		require.Nil(b, err2)
		require.NotNil(b, resp)
		require.Equal(b, 1, len(resp.Labels))
		require.Equal(b, "Demo1", resp.Labels["LABEL"])
	}
}

type nsmTestMonImpl struct {
}

func (n nsmTestMonImpl) MonitorConnections(scope *networkservice.MonitorScopeSelector, server networkservice.MonitorConnection_MonitorConnectionsServer) error {
	logrus.Infof("Begin monitor connections: %v", scope)
	for i := 0; i < 50; i++ {
		err := server.Send(&networkservice.ConnectionEvent{
			Type: 0,
			Connections: map[string]*networkservice.Connection{
				fmt.Sprintf("id-%v", i): {
					Id:    fmt.Sprintf("id-%v", i),
					State: networkservice.State_DOWN,
				},
			},
		})
		if err != nil {
			logrus.Errorf("Err %v", err)
		}
	}
	logrus.Infof("End event send: %v. Wait for server close", scope)
	<-server.Context().Done()
	logrus.Infof("End monitor connections: %v.", scope)
	return nil
}

func TestServerClientConnectionMonitor(t *testing.T) {
	// Construct a Server GRPC server
	server := grpc.NewServer()
	listener, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer func() { _ = listener.Close() }()
	defer server.Stop()
	// Register callback GRPC endpoint and callback.Server will hold callback clients to be callable.
	callbackServer := callback.NewServer(callback.IdentityByAuthority)
	callback.RegisterCallbackServiceServer(server, callbackServer)
	go func() {
		_ = server.Serve(listener)
	}()
	// Client stuff setup, construct a client GRPC server a standard way
	monTest := &nsmTestMonImpl{}

	clientGRPC := grpc.NewServer()
	defer clientGRPC.Stop()
	networkservice.RegisterMonitorConnectionServer(clientGRPC, monTest)

	// Connect from Client to Server
	var client *grpc.ClientConn
	client, err = grpc.DialContext(context.Background(), listener.Addr().String(), grpc.WithInsecure(), grpc.WithAuthority("my_client"), grpc.WithBlock())
	require.Nil(t, err)
	// Construct a callback client.
	callbackClient := callback.NewClient(client, clientGRPC)
	callbackClient.Serve(context.Background())
	ids := &idHolder{
		ids: make(chan string),
	}
	callbackServer.AddListener(ids)
	clientID := waitClientConnected(callbackClient, t, ids)
	nsmClientGRPC, err3 := callbackServer.NewClient(context.Background(), clientID)
	require.Nil(t, err3)
	nsmClient := networkservice.NewMonitorConnectionClient(nsmClientGRPC)
	monCtx, cancelMon := context.WithCancel(context.Background())
	monClient, err2 := nsmClient.MonitorConnections(monCtx, createScopeSelector())
	require.Nil(t, err2)
	events := []*networkservice.ConnectionEvent{}
	logrus.Infof("----------")
	for {
		conEvent, err4 := monClient.Recv()
		if err4 == nil {
			events = append(events, conEvent)
		} else {
			require.Equal(t, err4.Error(), "msg")
			break
		}
		if len(events) == 50 {
			break
		}
	}
	require.Equal(t, 50, len(events))
	cancelMon()
}

func waitClientConnected(callbackClient callback.Client, t *testing.T, ids *idHolder) string {
	clientID := ""
	select {
	case errValue := <-callbackClient.ErrChan():
		require.Errorf(t, errValue, "Unexpected error")
	case <-time.After(1 * time.Second):
		require.Fail(t, "Timeout waiting for client")
	case clientID = <-ids.ids:
		require.Equal(t, "my_client", clientID)
	}
	return clientID
}

func createScopeSelector() *networkservice.MonitorScopeSelector {
	return &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			{
				Name:  "segm",
				Token: "tok",
			},
		},
	}
}
