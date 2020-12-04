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

package refresh_test

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	endpointName     = "endpoint-name"
	connectionMarker = "refresh-marker"
)

type countClient struct {
	t     *testing.T
	count int32
}

func (c *countClient) validator(atLeast int32) func() bool {
	return func() bool {
		if count := atomic.LoadInt32(&c.count); count < atLeast {
			logrus.Warnf("count %v < atLeast %v", count, atLeast)
			return false
		}
		return true
	}
}

func (c *countClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request = request.Clone()
	conn := request.GetConnection()

	// Check that refresh updates the request.Connection field (issue #530).
	if atomic.AddInt32(&c.count, 1) == 1 {
		conn.NetworkServiceEndpointName = endpointName
	} else {
		assert.Equal(c.t, endpointName, conn.NetworkServiceEndpointName)
	}

	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *countClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// refreshTestServer is a helper endpoint to check that the Request()/Close()
// order isn't mixed by refresh (which may happen due to race condition between
// a requests initiated by a client, and a refresh timer), and timer initiated
// request aren't too fast or too slow.
//
// Usage details:
// * Each client Request() should be wrapped in beforeRequest()/afterRequest()
//   calls. Same for Close() and beforeClose()/afterClose().
// * Caveat: parallel client initiated requests aren't supported by this tester.
// * To distinguish between different requests, the value of
//   `Connection.Context.ExtraContext[connectionMarker]` is used as a marker.
type refreshTesterServer struct {
	t           *testing.T
	minDuration time.Duration
	maxDuration time.Duration

	mutex         sync.Mutex
	state         int
	lastSeen      time.Time
	currentMarker string
	nextMarker    string
}

type refreshTesterServerState = int

const (
	testRefreshStateInit = iota
	testRefreshStateWaitRequest
	testRefreshStateDoneRequest
	testRefreshStateRunning
	testRefreshStateWaitClose
	testRefreshStateDoneClose
)

func newRefreshTesterServer(t *testing.T, minDuration, maxDuration time.Duration) *refreshTesterServer {
	return &refreshTesterServer{
		t:           t,
		minDuration: minDuration,
		maxDuration: maxDuration,
		state:       testRefreshStateInit,
	}
}

func (t *refreshTesterServer) beforeRequest(marker string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.checkUnlocked()
	assert.Contains(t.t, []refreshTesterServerState{testRefreshStateInit, testRefreshStateRunning}, t.state, "Unexpected state")
	t.state = testRefreshStateWaitRequest
	t.nextMarker = marker
}

func (t *refreshTesterServer) afterRequest() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.checkUnlocked()
	assert.Equal(t.t, testRefreshStateDoneRequest, t.state, "Unexpected state")
	t.state = testRefreshStateRunning
}

func (t *refreshTesterServer) beforeClose() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.checkUnlocked()
	assert.Equal(t.t, testRefreshStateRunning, t.state, "Unexpected state")
	t.state = testRefreshStateWaitClose
}

func (t *refreshTesterServer) afterClose() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.checkUnlocked()
	assert.Equal(t.t, testRefreshStateDoneClose, t.state, "Unexpected state")
	t.state = testRefreshStateInit
	t.currentMarker = ""
}

func (t *refreshTesterServer) checkUnlocked() {
	if t.state == testRefreshStateDoneRequest || t.state == testRefreshStateRunning {
		delta := time.Now().UTC().Sub(t.lastSeen)
		assert.Lessf(t.t, int64(delta), int64(t.maxDuration), "Duration expired (too slow) delta=%v max=%v", delta, t.maxDuration)
	}
}

func (t *refreshTesterServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	t.mutex.Lock()
	locked := true
	defer func() {
		if locked {
			t.mutex.Unlock()
		}
	}()
	t.checkUnlocked()

	marker := request.Connection.Context.ExtraContext[connectionMarker]
	assert.NotEmpty(t.t, marker, "Marker is empty")

	switch t.state {
	case testRefreshStateWaitRequest:
		assert.Contains(t.t, []string{t.nextMarker, t.currentMarker}, marker, "Unexpected marker")
		if marker == t.nextMarker {
			t.state = testRefreshStateDoneRequest
			t.currentMarker = t.nextMarker
		}
	case testRefreshStateDoneRequest, testRefreshStateRunning, testRefreshStateWaitClose:
		assert.Equal(t.t, t.currentMarker, marker, "Unexpected marker")
		delta := time.Now().UTC().Sub(t.lastSeen)
		assert.GreaterOrEqual(t.t, int64(delta), int64(t.minDuration), "Too fast delta=%v min=%v", delta, t.minDuration)
	default:
		assert.Fail(t.t, "Unexpected state", t.state)
	}

	t.lastSeen = time.Now()

	t.mutex.Unlock()
	locked = false

	return next.Server(ctx).Request(ctx, request)
}

func (t *refreshTesterServer) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	t.mutex.Lock()
	locked := true
	defer func() {
		if locked {
			t.mutex.Unlock()
			locked = false
		}
	}()
	t.checkUnlocked()

	assert.Equal(t.t, testRefreshStateWaitClose, t.state, "Unexpected state")
	t.state = testRefreshStateDoneClose

	t.mutex.Unlock()
	locked = false

	return next.Server(ctx).Close(ctx, connection)
}

func mkRequest(marker string, conn *networkservice.Connection) *networkservice.NetworkServiceRequest {
	if conn == nil {
		conn = &networkservice.Connection{
			Id: "conn-id",
			Context: &networkservice.ConnectionContext{
				ExtraContext: map[string]string{
					connectionMarker: marker,
				},
			},
			NetworkService: "my-service-remote",
		}
	} else {
		conn.Context.ExtraContext[connectionMarker] = marker
	}
	return &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: conn,
	}
}

func generateRequests(t *testing.T, client networkservice.NetworkServiceClient, refreshTester *refreshTesterServer, iterations int, tickDuration time.Duration) {
	//nolint:gosec // Predictable random number generator is OK for testing purposes.
	randSrc := rand.New(rand.NewSource(0))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var oldConn *networkservice.Connection
	for i := 0; i < iterations && !t.Failed(); i++ {
		refreshTester.beforeRequest(strconv.Itoa(i))
		conn, err := client.Request(ctx, mkRequest(strconv.Itoa(i), oldConn))
		refreshTester.afterRequest()
		assert.NotNil(t, conn)
		assert.Nil(t, err)
		oldConn = conn

		if t.Failed() {
			return
		}

		if randSrc.Int31n(10) != 0 {
			time.Sleep(tickDuration)
		}

		if t.Failed() {
			return
		}

		if randSrc.Int31n(10) == 0 {
			refreshTester.beforeClose()
			_, err = client.Close(ctx, oldConn)
			assert.Nil(t, err)
			refreshTester.afterClose()
			oldConn = nil
		}
	}

	if oldConn != nil {
		refreshTester.beforeClose()
		_, _ = client.Close(ctx, oldConn)
		refreshTester.afterClose()
	}
	time.Sleep(tickDuration)
}
