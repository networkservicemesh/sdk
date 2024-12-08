// Copyright (c) 2020-2024 Cisco Systems, Inc.
//
// Copyright (c) 2021-2024 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Nordix Foundation.
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
package traceverbose_test

import (
	"bytes"
	"context"
	"encoding/json"

	"testing"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/testutil"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace/traceverbose"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func TestDiffMechanism(t *testing.T) {
	c1 := testutil.NewConnection()
	c2 := testutil.NewConnection()
	c2.MechanismPreferences[1].Type = "MEMIF"
	diffMsg, diff := traceverbose.Diff(c1.ProtoReflect(), c2.ProtoReflect())
	jsonOut, _ := json.Marshal(diffMsg)
	require.Equal(t, `{"mechanism_preferences":{"1":{"type":"MEMIF"}}}`, string(jsonOut))
	require.True(t, diff)
}

func TestDiffLabels(t *testing.T) {
	c1 := testutil.NewConnection()
	c2 := testutil.NewConnection()
	c2.MechanismPreferences[1].Parameters = map[string]string{
		"label":  "v3",
		"label2": "v4",
	}
	diffMsg, diff := traceverbose.Diff(c1.ProtoReflect(), c2.ProtoReflect())
	jsonOut, _ := json.Marshal(diffMsg)
	require.Equal(t, `{"mechanism_preferences":{"1":{"parameters":{"+label2":"v4","label":"v3"}}}}`, string(jsonOut))
	require.True(t, diff)
}

func TestDiffPath(t *testing.T) {
	c1 := testutil.NewConnection()
	c2 := testutil.NewConnection()

	c1.Connection.Path = &networkservice.Path{
		Index: 0,
		PathSegments: []*networkservice.PathSegment{
			{Id: "id1", Token: "t1"},
		},
	}

	diffMsg, diff := traceverbose.Diff(c1.ProtoReflect(), c2.ProtoReflect())
	jsonOut, _ := json.Marshal(diffMsg)
	require.Equal(t, `{"connection":{"path":{"path_segments":{"-0":{"id":"id1","token":"t1"}}}}}`, string(jsonOut))
	require.True(t, diff)
}

func TestDiffPathAdd(t *testing.T) {
	c1 := testutil.NewConnection()
	c2 := testutil.NewConnection()

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

	diffMsg, diff := traceverbose.Diff(c1.ProtoReflect(), c2.ProtoReflect())
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
	logrus.SetLevel(logrus.TraceLevel)
	log.EnableTracing(true)

	// Create a chain with modifying elements
	ch := chain.NewNetworkServiceServer(
		&testutil.LabelChangerFirstServer{},
		&testutil.LabelChangerSecondServer{},
	)

	request := testutil.NewConnection()

	conn, err := ch.Request(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	e, err := ch.Close(context.Background(), conn)
	require.NoError(t, err)
	require.NotNil(t, e)

	expectedOutput := ` [TRAC] [id:conn-1] [type:networkService] (1) ⎆ testutil/LabelChangerFirstServer.Request()
 [TRAC] [id:conn-1] [type:networkService] (1.1)   request=connection:{id:"conn-1"  context:{ip_context:{src_ip_required:true}}}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"  parameters:{key:"label"  value:"v2"}}
 [TRAC] [id:conn-1] [type:networkService] (1.2)   request-diff=connection:{id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"A"}}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"  parameters:{key:"label"  value:"v2"}}
 [TRAC] [id:conn-1] [type:networkService] (2)  ⎆ testutil/LabelChangerSecondServer.Request()
 [TRAC] [id:conn-1] [type:networkService] (2.1)    request-diff=connection:{id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"B"}}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"  parameters:{key:"label"  value:"v2"}}
 [TRAC] [id:conn-1] [type:networkService] (2.2)    request-response=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"B"}
 [TRAC] [id:conn-1] [type:networkService] (2.3)    request-response-diff=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"C"}
 [TRAC] [id:conn-1] [type:networkService] (1.3)   request-response-diff=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"D"}
 [TRAC] [id:conn-1] [type:networkService] (1) ⎆ testutil/LabelChangerFirstServer.Close()
 [TRAC] [id:conn-1] [type:networkService] (1.1)   close=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"D"}
 [TRAC] [id:conn-1] [type:networkService] (1.2)   close-diff=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"W"}
 [TRAC] [id:conn-1] [type:networkService] (2)  ⎆ testutil/LabelChangerSecondServer.Close()
 [TRAC] [id:conn-1] [type:networkService] (2.1)    close-diff=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"X"}
 [TRAC] [id:conn-1] [type:networkService] (2.2)    close-response=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"X"}
 [TRAC] [id:conn-1] [type:networkService] (2.3)    close-response-diff=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"Y"}
 [TRAC] [id:conn-1] [type:networkService] (1.3)   close-response-diff=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"Z"}
`
	result := testutil.TrimLogTime(&buff)

	result = testutil.Normalize(result)
	expectedOutput = testutil.Normalize(expectedOutput)

	require.Equal(t, expectedOutput, result)
}

func TestErrorOutput(t *testing.T) {
	// Configure logging
	// Set output to buffer
	var buff bytes.Buffer
	logrus.SetOutput(&buff)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logrus.SetLevel(logrus.TraceLevel)
	log.EnableTracing(true)

	// Create a chain with modifying elements
	ch := chain.NewNetworkServiceServer(
		&testutil.LabelChangerFirstServer{},
		&testutil.LabelChangerSecondServer{},
		&testutil.ErrorServer{},
	)

	request := testutil.NewConnection()

	conn, err := ch.Request(context.Background(), request)
	require.Error(t, err)
	require.Nil(t, conn)

	expectedOutput := ` [TRAC] [id:conn-1] [type:networkService] (1) ⎆ testutil/LabelChangerFirstServer.Request()
 [TRAC] [id:conn-1] [type:networkService] (1.1)   request=connection:{id:"conn-1"  context:{ip_context:{src_ip_required:true}}}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"  parameters:{key:"label"  value:"v2"}}
 [TRAC] [id:conn-1] [type:networkService] (1.2)   request-diff=connection:{id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"A"}}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"  parameters:{key:"label"  value:"v2"}}
 [TRAC] [id:conn-1] [type:networkService] (2)  ⎆ testutil/LabelChangerSecondServer.Request()
 [TRAC] [id:conn-1] [type:networkService] (2.1)    request-diff=connection:{id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"B"}}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"}  mechanism_preferences:{cls:"LOCAL"  type:"KERNEL"  parameters:{key:"label"  value:"v2"}}
 [TRAC] [id:conn-1] [type:networkService] (3)   ⎆ testutil/ErrorServer.Request()
 [TRAC] [id:conn-1] [type:networkService] (3.1)     request-response=id:"conn-1"  context:{ip_context:{src_ip_required:true}}  labels:{key:"Label"  value:"B"}
 [ERRO] [id:conn-1] [type:networkService] (3.2)     Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
 [TRAC] [id:conn-1] [type:networkService] (2.2)    request-response-diff=<nil>
 [ERRO] [id:conn-1] [type:networkService] (2.3)    Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
 [ERRO] [id:conn-1] [type:networkService] (1.3)   Error returned from api/pkg/api/networkservice/networkServiceClient.Close;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*beginTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:85;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	github.com/networkservicemesh/sdk/pkg/networkservice/core/trace.(*endTraceClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/trace/client.go:106;	github.com/networkservicemesh/sdk/pkg/networkservice/core/next.(*nextClient).Close;		/root/go/pkg/mod/github.com/networkservicemesh/sdk@v0.5.1-0.20210929180427-ec235de055f1/pkg/networkservice/core/next/client.go:65;	
`
	result := testutil.TrimLogTime(&buff)
	result = testutil.Normalize(result)
	expectedOutput = testutil.Normalize(expectedOutput)

	require.Equal(t, expectedOutput, result)
}
