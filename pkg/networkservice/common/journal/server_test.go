// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
package journal_test

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

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/journal"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
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

	srv, err := journal.NewServer("foo", conn)
	assert.NoError(t, err)

	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"10.0.0.1/32"},
					DstIpAddrs: []string{"10.0.0.2/32"},
				},
			},
			Path: &networkservice.Path{},
		},
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	clockMock := clockmock.NewMock()
	ctx := clock.WithClock(context.Background(), clockMock)

	ts := time.Now()
	clockMock.Set(ts)

	sub, err := testConn.Subscribe("foo", func(msg *stan.Msg) {
		data := msg.Data
		entry := journal.Entry{}
		subErr := json.Unmarshal(data, &entry)
		assert.NoError(t, subErr)

		assert.Equal(t, entry.Time.Unix(), ts.Unix())
		assert.Equal(t, []string{"10.0.0.1/32"}, entry.Sources)
		assert.Equal(t, []string{"10.0.0.2/32"}, entry.Destinations)
		assert.Equal(t, journal.ActionRequest, entry.Action)
		assert.NotNil(t, entry.Path)

		wg.Done()
	})
	assert.NoError(t, err)
	defer func() {
		_ = sub.Close()
	}()

	_, err = srv.Request(ctx, req)
	assert.NoError(t, err)

	wg.Wait()
}
