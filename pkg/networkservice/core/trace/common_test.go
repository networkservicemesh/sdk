// Copyright (c) 2020 Cisco Systems, Inc.
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
	"encoding/json"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
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
