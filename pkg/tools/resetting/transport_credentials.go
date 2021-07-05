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

// Package resetting provides grpc credentials.TransportCredentials wrapper that can init re-dial prodcudere for grpc connection on event receiving from the channel.
// This might be useful for updating long-living connections with expired certificates on the server and client-side.
// See at https://github.com/grpc/grpc-go/blob/45549242f79aacb850de77336a76777bef8bbe01/clientconn.go#L1138
package resetting

import (
	"context"
	"net"

	"github.com/edwarnicke/serialize"
	"google.golang.org/grpc/credentials"
)

type transportCredentails struct {
	credentials.TransportCredentials
	exec  serialize.Executor
	conns map[*net.Conn]net.Conn
}

type subscribableConn struct {
	net.Conn
	onClosing func()
}

func (c *subscribableConn) Close() error {
	if c.onClosing != nil {
		c.onClosing()
	}
	return c.Conn.Close()
}

// NewCredentials adds to passed credentials.TransportCredentials a possible to reset all connections on receiving a update from the resetCh.
func NewCredentials(creds credentials.TransportCredentials, resetCh <-chan struct{}) credentials.TransportCredentials {
	if resetCh == nil {
		panic("updateCh cannot be nil")
	}
	var s = &transportCredentails{
		TransportCredentials: creds,
		conns:                make(map[*net.Conn]net.Conn),
	}

	go func() {
		for range resetCh {
			s.exec.AsyncExec(func() {
				for _, conn := range s.conns {
					_ = conn.Close()
				}
				s.conns = make(map[*net.Conn]net.Conn)
			})
		}
	}()

	return s
}

func (u *transportCredentails) ClientHandshake(ctx context.Context, authority string, conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	cc, authInfo, err := u.TransportCredentials.ClientHandshake(ctx, authority, conn)

	if err != nil {
		return cc, authInfo, err
	}

	u.exec.AsyncExec(func() {
		u.conns[&cc] = cc
	})

	return &subscribableConn{
		Conn: cc,
		onClosing: func() {
			u.exec.AsyncExec(func() {
				delete(u.conns, &cc)
			})
		},
	}, authInfo, err
}
