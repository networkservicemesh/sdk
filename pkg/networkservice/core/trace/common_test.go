// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package trace_test has few tests for tracing diffs
package trace_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func TestDiffMechanism(t *testing.T) {
	c1 := newConnection()
	c2 := newConnection()
	c2.MechanismPreferences[1].Type = "MEMIF"
	diffMsg, diff := trace.Diff(c1.ProtoReflect(), c2.ProtoReflect())
	jsonOut, _ := json.Marshal(diffMsg)
	require.Equal(t, `{"mechanism_preferences":{"1":{"type":"MEMIF"}}}`, string(jsonOut))
	require.True(t, diff)
}

func TestDiffLabels(t *testing.T) {
	c1 := newConnection()
	c2 := newConnection()
	c2.MechanismPreferences[1].Parameters = map[string]string{
		"label":  "v3",
		"label2": "v4",
	}
	diffMsg, diff := trace.Diff(c1.ProtoReflect(), c2.ProtoReflect())
	jsonOut, _ := json.Marshal(diffMsg)
	require.Equal(t, `{"mechanism_preferences":{"1":{"parameters":{"+label2":"v4","label":"v3"}}}}`, string(jsonOut))
	require.True(t, diff)
}
func TestDiffPath(t *testing.T) {
	c1 := newConnection()
	c2 := newConnection()

	c1.Connection.Path = &networkservice.Path{
		Index: 0,
		PathSegments: []*networkservice.PathSegment{
			{Id: "id1", Token: "t1"},
		},
	}

	diffMsg, diff := trace.Diff(c1.ProtoReflect(), c2.ProtoReflect())
	jsonOut, _ := json.Marshal(diffMsg)
	require.Equal(t, `{"connection":{"path":{"path_segments":{"-0":{"id":"id1","token":"t1"}}}}}`, string(jsonOut))
	require.True(t, diff)
}

func TestDiffPathAdd(t *testing.T) {
	c1 := newConnection()
	c2 := newConnection()

	c1.Connection.Path = &networkservice.Path{
		Index: 0,
		PathSegments: []*networkservice.PathSegment{
			{Id: "id1", Token: "t1"},
		},
	}
	c2.Connection.Path = &networkservice.Path{
		Index: 0,
		PathSegments: []*networkservice.PathSegment{
			{Id: "id1", Token: "t1"},
			{Id: "id2", Token: "t2"},
		},
	}

	diffMsg, diff := trace.Diff(c1.ProtoReflect(), c2.ProtoReflect())
	jsonOut, _ := json.Marshal(diffMsg)
	require.Equal(t, `{"connection":{"path":{"path_segments":{"+1":{"id":"id2","token":"t2"}}}}}`, string(jsonOut))
	require.True(t, diff)
}

func TestTraceOutput(t *testing.T) {
	// Configure logging
	// Set output to buffer
	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	log.EnableTracing(true)

	// Create a chain with modifying elements
	ch := chain.NewNamedNetworkServiceServer(
		"TestTraceOutput",
		&labelChangerFirstServer{},
		&labelChangerSecondServer{},
	)

	request := newConnection()

	conn, err := ch.Request(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	e, err := ch.Close(context.Background(), conn)
	require.NoError(t, err)
	require.NotNil(t, e)

	expectedOutput :=
		"level=info msg=\"(1) ⎆ sdk/pkg/networkservice/core/trace_test/labelChangerFirstServer.Request()\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(1.1)   request={\\\"connection\\\":{\\\"id\\\":\\\"conn-1\\\",\\\"context\\\":" +
			"{\\\"ip_context\\\":{\\\"src_ip_required\\\":true}}},\\\"mechanism_preferences\\\":[{\\\"cls\\\":\\\"LOCAL\\\"," +
			"\\\"type\\\":\\\"KERNEL\\\"},{\\\"cls\\\":\\\"LOCAL\\\",\\\"type\\\":\\\"KERNEL\\\",\\\"parameters\\\":{\\\"label\\\"" +
			":\\\"v2\\\"}}]}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(1.2)   request-diff={\\\"connection\\\":{\\\"labels\\\":{\\\"+Label\\\":\\\"A\\\"}}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(2)  ⎆ sdk/pkg/networkservice/core/trace_test/labelChangerSecondServer.Request()\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(2.1)    request-diff={\\\"connection\\\":{\\\"labels\\\":{\\\"Label\\\":\\\"B\\\"}}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(2.2)    request-response={\\\"id\\\":\\\"conn-1\\\",\\\"context\\\":{\\\"ip_context\\\":{\\\"src_ip_required\\\":true}}," +
			"\\\"labels\\\":{\\\"Label\\\":\\\"B\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(2.3)    request-response-diff={\\\"labels\\\":{\\\"Label\\\":\\\"C\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(1.3)   request-response-diff={\\\"labels\\\":{\\\"Label\\\":\\\"D\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(1) ⎆ sdk/pkg/networkservice/core/trace_test/labelChangerFirstServer.Close()\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(1.1)   close={\\\"id\\\":\\\"conn-1\\\",\\\"context\\\":{\\\"ip_context\\\":{\\\"src_ip_required\\\":true}}," +
			"\\\"labels\\\":{\\\"Label\\\":\\\"D\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(1.2)   close-diff={\\\"labels\\\":{\\\"Label\\\":\\\"W\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(2)  ⎆ sdk/pkg/networkservice/core/trace_test/labelChangerSecondServer.Close()\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(2.1)    close-diff={\\\"labels\\\":{\\\"Label\\\":\\\"X\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(2.2)    close-response={\\\"id\\\":\\\"conn-1\\\",\\\"context\\\":{\\\"ip_context\\\":{\\\"src_ip_required\\\"" +
			":true}},\\\"labels\\\":{\\\"Label\\\":\\\"X\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(2.3)    close-response-diff={\\\"labels\\\":{\\\"Label\\\":\\\"Y\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n" +
			"level=info msg=\"(1.3)   close-response-diff={\\\"labels\\\":{\\\"Label\\\":\\\"Z\\\"}}\" id=conn-1 name=TestTraceOutput type=NetworkService\n"

	require.Equal(t, expectedOutput, buff.String())
}

func newConnection() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpRequired: true,
				},
			},
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: "KERNEL",
				Cls:  cls.LOCAL,
			},
			{
				Type: "KERNEL",
				Cls:  cls.LOCAL,
				Parameters: map[string]string{
					"label": "v2",
				},
			},
		},
	}
}

type labelChangerFirstServer struct{}

func (c *labelChangerFirstServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Labels = map[string]string{
		"Label": "A",
	}
	rv, err := next.Server(ctx).Request(ctx, request)
	rv.Labels = map[string]string{
		"Label": "D",
	}
	return rv, err
}

func (c *labelChangerFirstServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	connection.Labels = map[string]string{
		"Label": "W",
	}
	r, err := next.Server(ctx).Close(ctx, connection)
	connection.Labels = map[string]string{
		"Label": "Z",
	}
	return r, err
}

type labelChangerSecondServer struct{}

func (c *labelChangerSecondServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	request.Connection.Labels = map[string]string{
		"Label": "B",
	}
	rv, err := next.Server(ctx).Request(ctx, request)
	rv.Labels = map[string]string{
		"Label": "C",
	}
	return rv, err
}

func (c *labelChangerSecondServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	connection.Labels = map[string]string{
		"Label": "X",
	}
	r, err := next.Server(ctx).Close(ctx, connection)
	connection.Labels = map[string]string{
		"Label": "Y",
	}
	return r, err
}
