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

// +build linux

package netnsmonitor_test

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netns"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/netnsmonitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

func TestNetNSMonitor(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	netNSName := uuid.New().String()
	ns, err := func() (netns.NsHandle, error) {
		baseHandle, err := netns.Get()
		require.NoError(t, err)

		defer func() {
			_ = netns.Set(baseHandle)
			_ = baseHandle.Close()
		}()
		return netns.NewNamed(netNSName)
	}()
	require.NoError(t, err)
	defer func() {
		_ = ns.Close()
		_ = netns.DeleteNamed(netNSName)
	}()

	inode, err := getInodeByHandle(ns)
	require.NoError(t, err)

	ctx := context.Background()
	counter := newConnectionCounter()
	srv := next.NewNetworkServiceServer(
		netnsmonitor.NewServer(ctx),
		counter)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:    "test-connection",
			State: networkservice.State_UP,
			Mechanism: &networkservice.Mechanism{
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					kernel.NetNSInodeKey: fmt.Sprint(inode),
				},
			},
		},
	}

	conn, err := srv.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 1, counter.Count())

	require.NoError(t, ns.Close())
	require.NoError(t, netns.DeleteNamed(netNSName))

	<-time.After(time.Second)
	require.Equal(t, 0, counter.Count())

	_, err = srv.Close(ctx, conn)
	require.NoError(t, err)
}

type connectionCounter struct {
	connections map[string]*networkservice.Connection
}

func newConnectionCounter() *connectionCounter {
	return &connectionCounter{
		connections: make(map[string]*networkservice.Connection),
	}
}

func (m *connectionCounter) Count() int {
	return len(m.connections)
}

func (m *connectionCounter) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return conn, err
	}

	m.connections[conn.GetId()] = conn.Clone()
	return conn, err
}

func (m *connectionCounter) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	delete(m.connections, conn.GetId())
	return next.Server(ctx).Close(ctx, conn)
}

func getInodeByHandle(handle netns.NsHandle) (uintptr, error) {
	file := os.NewFile(uintptr(handle), "netns")
	info, err := file.Stat()
	if err != nil {
		return 0, errors.Wrap(err, "error stat file")
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("not a stat_t")
	}

	return uintptr(stat.Ino), nil
}
