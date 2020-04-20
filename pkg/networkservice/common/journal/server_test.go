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
package journal

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	stand "github.com/nats-io/nats-streaming-server/server"
	"github.com/nats-io/stan.go"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	streamingServer, err := stand.RunServer("test_journal")
	assert.NoError(t, err)
	defer streamingServer.Shutdown()

	conn, err := stan.Connect(streamingServer.ClusterID(), "clientid")
	assert.NoError(t, err)
	defer func() {
		_ = conn.Close()
	}()

	testConn, err := stan.Connect(streamingServer.ClusterID(), "testid")
	assert.NoError(t, err)
	defer func() {
		_ = testConn.Close()
	}()

	srv, err := NewServer("foo", conn)
	assert.NoError(t, err)

	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddr: "10.0.0.1/32",
					DstIpAddr: "10.0.0.2/32",
				},
			},
			Path: &networkservice.Path{},
		},
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	ts := time.Now()

	sub, err := testConn.Subscribe("foo", func(msg *stan.Msg) {
		data := msg.Data
		entry := Entry{}
		subErr := json.Unmarshal(data, &entry)
		assert.NoError(t, subErr)

		assert.GreaterOrEqual(t, entry.Time.Unix(), ts.Unix())
		assert.Equal(t, "10.0.0.1/32", entry.Source)
		assert.Equal(t, "10.0.0.2/32", entry.Destination)
		assert.Equal(t, ActionRequest, entry.Action)
		assert.NotNil(t, entry.Path)

		wg.Done()
	})
	assert.NoError(t, err)
	defer func() {
		_ = sub.Close()
	}()

	_, err = srv.Request(context.Background(), req)
	assert.NoError(t, err)

	wg.Wait()
}
