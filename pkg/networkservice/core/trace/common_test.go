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
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"

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
	logrus.SetLevel(logrus.DebugLevel)
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

	//nolint:stylecheck // disable ST1018: string literal contains Unicode control characters
	expectedOutput := `[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(1) ⎆ sdk/pkg/networkservice/core/trace_test/labelChangerFirstServer.Request()
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(1.1)   request: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"connection": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"id": "conn-1",
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"context": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"ip_context": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m				"src_ip_required": true
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	},
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"mechanism_preferences": [
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		{
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"cls": "LOCAL",
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"type": "KERNEL"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		},
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		{
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"cls": "LOCAL",
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"type": "KERNEL",
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"parameters": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m				"label": "v2"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	]
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(1.2)   request-diff: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"connection": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"+Label": "A"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(2)  ⎆ sdk/pkg/networkservice/core/trace_test/labelChangerSecondServer.Request()
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(2.1)    request-diff: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"connection": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"Label": "B"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(2.2)    response: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"id": "conn-1",
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"context": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"ip_context": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"src_ip_required": true
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	},
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "B"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(2.3)    response-diff: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "C"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(1.3)   response-diff: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "D"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(1) ⎆ sdk/pkg/networkservice/core/trace_test/labelChangerFirstServer.Close()
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(1.1)   request: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"id": "conn-1",
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"context": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"ip_context": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"src_ip_required": true
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	},
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "D"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(1.2)   request-diff: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "W"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(2)  ⎆ sdk/pkg/networkservice/core/trace_test/labelChangerSecondServer.Close()
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(2.1)    request-diff: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "X"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(2.2)    response: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"id": "conn-1",
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"context": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"ip_context": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m			"src_ip_required": true
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	},
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "X"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(2.3)    response-diff: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "Y"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m(1.3)   response-diff: {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	"labels": {
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m		"Label": "Z"
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m	}
[37m [DEBU] [id:conn-1] [name:TestTraceOutput] [0m}
`

	// Logger created by the trace chain element uses custom formatter, which prints date and time info in each line
	// To check if output matches our expectations, we need to somehow get rid of this info.
	// We have the following options:
	// 1. Configure formatter options on logger creation in trace element
	// 2. Use some global configuration (either set global default formatter
	// 	  instead of creating it in trace element or use global config for our formatter)
	// 3. Remove datetime information from the output
	// Since we are unlikely to need to remove date in any case except these tests,
	// it seems like the third option would be the most convenient.
	result := ""
	datetimeLength := 19
	for _, line := range strings.Split(buff.String(), "\n") {
		if len(line) > datetimeLength {
			result += line[datetimeLength:] + "\n"
		} else {
			result += line
		}
	}

	require.Equal(t, expectedOutput, result)
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
