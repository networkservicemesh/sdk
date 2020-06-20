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
	"github.com/sirupsen/logrus"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	//expireTimeout        = 100 * time.Millisecond
	expireTimeout        = 50 * time.Millisecond
	waitForTimeout       = 3 * expireTimeout
	tickTimeout          = 10 * time.Millisecond
	refreshCount         = 5
	expectAbsenceTimeout = 5 * expireTimeout

	requestNumber contextKeyType = "RequestNumber"
)

type contextKeyType string

func withRequestNumber(parent context.Context, number int) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, requestNumber, number)
}

func getRequestNumber(ctx context.Context) int {
	if rv, ok := ctx.Value(requestNumber).(int); ok {
		return rv
	}
	return -1
}

type testRefresh struct {
	RequestFunc func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error)
}

func (t *testRefresh) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return t.RequestFunc(ctx, in, opts...)
}

func (t *testRefresh) Close(context.Context, *networkservice.Connection, ...grpc.CallOption) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func setExpires(conn *networkservice.Connection, expireTimeout time.Duration) {
	expireTime := time.Now().Add(expireTimeout)
	expires := &timestamp.Timestamp{
		Seconds: expireTime.Unix(),
		Nanos:   int32(expireTime.Nanosecond()),
	}
	conn.Path = &networkservice.Path{
		Index: 0,
		PathSegments: []*networkservice.PathSegment{
			{
				Expires: expires,
			},
		},
	}
}

func hasValue(c <-chan struct{}) bool {
	select {
	case <-c:
		logrus.Info("taken value from <-requestCh")
		return true
	default:
		return false
	}
}

func firstGetsValueEarlier(c1, c2 <-chan struct{}) bool {
	select {
	case <-c2:
		logrus.Info("taken value from <-requestCh")
		return false
	case <-c1:
		logrus.Info("taken value from <-absence")
		return true
	}
}

func TestNewClient_RunTests100Times(t *testing.T) {
	for i := 0; i < 400; i++ {
		logrus.Info("-------------------------------------------------------------")
		logrus.Infof("Iteration %d", i)
		//TestNewClient_StopRefreshAtAnotherRequest(t)
		TestNewClient_StopRefreshAtClose(t)
	}
}

func TestNewClient_StopRefreshAtClose(t *testing.T) {
	defer goleak.VerifyNone(t)

	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.999999999"
	logrus.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	requestCh := make(chan struct{}, 1)
	testRefresh := &testRefresh{
		RequestFunc: func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (connection *networkservice.Connection, err error) {
			setExpires(in.GetConnection(), expireTimeout)
			if refreshCtx := refresh.RefreshContext(ctx); refreshCtx != nil {
				refreshNumber := refresh.GetRefreshNumber(refreshCtx)
				logrus.Infof("pushed refresh #%d value to requestCh<-", refreshNumber.Number)
			} else {
				logrus.Info("pushed initial request value to requestCh<-")
			}
			requestCh <- struct{}{}
			return in.GetConnection(), nil
		},
	}

	client := next.NewNetworkServiceClient(refresh.NewClient(), testRefresh)
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
		},
	}
	logrus.Info("Initial Request()")
	conn, err := client.Request(context.Background(), request)
	assert.Nil(t, err)

	assert.True(t, hasValue(requestCh)) // receive value from initial request
	logrus.Info("Got value from initial request")
	for i := 0; i < refreshCount; i++ {
		require.Eventually(t, func() bool { return hasValue(requestCh) }, waitForTimeout, tickTimeout)
		logrus.Infof("Got value from POTENTIAL refresh %d", i+1)
	}

	logrus.Info("Close()")
	_, err = client.Close(context.Background(), conn)
	assert.Nil(t, err)

	absence := make(chan struct{})
	time.AfterFunc(expectAbsenceTimeout, func() {
		absence <- struct{}{}
	})
	logrus.Info("expect no refreshes...")
	assert.True(t, firstGetsValueEarlier(absence, requestCh))
}

func TestNewClient_StopRefreshAtAnotherRequest(t *testing.T) {
	defer goleak.VerifyNone(t)

	requestCh := make(chan struct{}, 1)
	testRefresh := &testRefresh{
		RequestFunc: func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (connection *networkservice.Connection, err error) {
			setExpires(in.GetConnection(), expireTimeout)
			if getRequestNumber(ctx) == 1 {
				requestCh <- struct{}{}
			}
			return in.GetConnection(), nil
		},
	}

	client := next.NewNetworkServiceClient(refresh.NewClient(), testRefresh)

	request1 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
		},
	}
	_, err := client.Request(withRequestNumber(context.Background(), 1), request1)
	assert.Nil(t, err)

	assert.True(t, hasValue(requestCh)) // receive value from initial request
	for i := 0; i < refreshCount; i++ {
		require.Eventually(t, func() bool { return hasValue(requestCh) }, waitForTimeout, tickTimeout)
	}

	request2 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn-1",
		},
	}
	conn, err := client.Request(withRequestNumber(context.Background(), 2), request2)
	assert.Nil(t, err)

	absence := make(chan struct{})
	time.AfterFunc(expectAbsenceTimeout, func() {
		absence <- struct{}{}
	})
	assert.True(t, firstGetsValueEarlier(absence, requestCh))

	_, err = client.Close(context.Background(), conn)
	assert.Nil(t, err)
}
