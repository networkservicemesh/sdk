// Copyright (c) 2022 Cisco and/or its affiliates.
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

package vl3mtu_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"

	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/mtu/vl3mtu"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

const (
	// certPem is a X.509 certificate with spiffeId = "spiffe://test.com/workload"
	certPem = `-----BEGIN CERTIFICATE-----
MIIBvjCCAWWgAwIBAgIQbnFakUhzr52nHoLGltZDyDAKBggqhkjOPQQDAjAdMQsw
CQYDVQQGEwJVUzEOMAwGA1UEChMFU1BJUkUwHhcNMjAwMTAxMDEwMTAxWhcNMzAw
MTAxMDEwMTAxWjAdMQswCQYDVQQGEwJVUzEOMAwGA1UEChMFU1BJUkUwWTATBgcq
hkjOPQIBBggqhkjOPQMBBwNCAASlFpbASv+NIyVdFwTp22JR5gx7D6LJ01Z8Wz0S
ZiBneWRAcYUBBQY6zKwr/RQtCDxUcFfFyq4zEfUD29a5Phnoo4GGMIGDMA4GA1Ud
DwEB/wQEAwIDqDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0T
AQH/BAIwADAdBgNVHQ4EFgQUJJpYlJa1eNEcks+zJcwKClopSAowJQYDVR0RBB4w
HIYac3BpZmZlOi8vdGVzdC5jb20vd29ya2xvYWQwCgYIKoZIzj0EAwIDRwAwRAIg
Dk6tlURSF8ULhNbnyUxFQ33rDic2dX8jOIstV2dWErwCIDRH2yw0swTcUMQWYgHy
aMp+T747AZGjOEfwHb9/w+7m
-----END CERTIFICATE-----
`
)

func getContextWithTLSCert(t *testing.T) (context.Context, context.CancelFunc) {
	block, _ := pem.Decode([]byte(certPem))
	x509cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	authInfo := &credentials.TLSInfo{
		State: tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{x509cert},
		},
	}
	return context.WithTimeout(peer.NewContext(context.Background(), &peer.Peer{AuthInfo: authInfo}), time.Second)
}

func Test_vl3MtuServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	// Put peer Certificate to context

	// Specify pathSegments to test
	segmentNames := []string{"local-nsm", "remote-nsm"}
	ctx, cancel := getContextWithTLSCert(t)
	defer cancel()
	// Create monitorServer
	var monitorServer networkservice.MonitorConnectionServer
	server := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		monitor.NewServer(ctx, &monitorServer),
		vl3mtu.NewServer(),
	)
	monitorClient := adapters.NewMonitorServerToClient(monitorServer)

	// Create maps to hold returned connections and receivers
	connections := make(map[string]*networkservice.Connection)
	receivers := make(map[string]networkservice.MonitorConnection_MonitorConnectionsClient)

	// Get Empty initial state transfers
	for _, segmentName := range segmentNames {
		monitorCtx, cancelMonitor := context.WithCancel(ctx)
		defer cancelMonitor()

		var monitorErr error
		receivers[segmentName], monitorErr = monitorClient.MonitorConnections(monitorCtx, &networkservice.MonitorScopeSelector{
			PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
		})
		require.NoError(t, monitorErr)
		event, err := receivers[segmentName].Recv()
		require.NoError(t, err)

		require.NotNil(t, event)
		require.Equal(t, networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, event.GetType())
		require.Empty(t, event.GetConnections()[segmentName].GetPath().GetPathSegments())
	}

	// Send requests
	var err error
	connections[segmentNames[0]], err = server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: segmentNames[0],
			Path: &networkservice.Path{
				Index:        0,
				PathSegments: []*networkservice.PathSegment{{Name: segmentNames[0]}},
			},
		},
	})
	require.NoError(t, err)

	// Send requests with different mtu
	connections[segmentNames[1]], err = server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: segmentNames[1],
			Path: &networkservice.Path{
				Index:        1,
				PathSegments: []*networkservice.PathSegment{{Name: segmentNames[1]}},
			},
			Context: &networkservice.ConnectionContext{MTU: 1500},
		},
	})
	require.NoError(t, err)

	// Get Updates and insure we've properly filtered by segmentName
	for _, segmentName := range segmentNames {
		var event *networkservice.ConnectionEvent
		event, err = receivers[segmentName].Recv()
		require.NoError(t, err)

		require.NotNil(t, event)
		require.Equal(t, networkservice.ConnectionEventType_UPDATE, event.GetType())
		require.Len(t, event.GetConnections()[segmentName].GetPath().GetPathSegments(), 1)
		require.Equal(t, segmentName, event.GetConnections()[segmentName].GetPath().GetPathSegments()[0].GetName())
	}

	// The first client should receive REFRESH_REQUESTED state, because MTU was updated
	event, err := receivers[segmentNames[0]].Recv()
	require.NoError(t, err)

	require.NotNil(t, event)
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, event.GetType())
	require.Equal(t, networkservice.State_REFRESH_REQUESTED, event.GetConnections()[segmentNames[0]].State)
	require.Len(t, event.GetConnections()[segmentNames[0]].GetPath().GetPathSegments(), 1)
	require.Equal(t, segmentNames[0], event.GetConnections()[segmentNames[0]].GetPath().GetPathSegments()[0].GetName())

	// Close Connections
	for _, conn := range connections {
		_, err := server.Close(ctx, conn)
		require.NoError(t, err)
	}

	// Get deleteMonitorClientCC Events and insure we've properly filtered by segmentName
	for _, segmentName := range segmentNames {
		event, err := receivers[segmentName].Recv()
		require.NoError(t, err)

		require.NotNil(t, event)
		require.Equal(t, networkservice.ConnectionEventType_DELETE, event.GetType())
		require.Len(t, event.GetConnections()[segmentName].GetPath().GetPathSegments(), 1)
		require.Equal(t, segmentName, event.GetConnections()[segmentName].GetPath().GetPathSegments()[0].GetName())
	}
}
