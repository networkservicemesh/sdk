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

package callback

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type clientConnection struct {
	client     connectionStream
	localAddr  *net.UnixAddr
	remoteAddr *net.UnixAddr
	cancel     context.CancelFunc
	buffer     *bytes.Buffer
	ctx        context.Context
}

type connectionStream interface {
	Send(*CallbackData) error
	Recv() (*CallbackData, error)
}

func newConnection(ctx context.Context, cancelFunc context.CancelFunc, stream connectionStream) *clientConnection {
	return &clientConnection{
		ctx:    ctx,
		cancel: cancelFunc,
		client: stream,
		localAddr: &net.UnixAddr{
			Name: "callback",
			Net:  "unix",
		},
		remoteAddr: &net.UnixAddr{
			Name: "callback",
			Net:  "unix",
		},
		buffer: bytes.NewBuffer([]byte{}),
	}
}

func (c *clientConnection) isShowError(err error) bool {
	return err != nil && c.ctx.Err() == nil && !strings.Contains(err.Error(), "context canceled")
}

func (c *clientConnection) Read(b []byte) (n int, err error) {
	for {
		if c.buffer.Len() > 0 {
			n, err = c.buffer.Read(b)
			return
		}
		data, err := c.client.Recv()
		if err != nil {
			if c.isShowError(err) {
				logrus.Errorf("error receive: %v", err)
			}
			return 0, err
		}
		if data.Kind == DataKind_Close {
			return 0, errors.New("connection is closed")
		}
		_, _ = c.buffer.Write(data.Data)
	}
}

func (c *clientConnection) Write(b []byte) (n int, err error) {
	err = c.client.Send(&CallbackData{Data: b, Kind: DataKind_Data})
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *clientConnection) Close() error {
	err := c.client.Send(&CallbackData{Data: nil, Kind: DataKind_Close})
	c.cancel()
	return err
}

func (c *clientConnection) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *clientConnection) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *clientConnection) SetDeadline(t time.Time) error {
	// Not required
	return nil
}

func (c *clientConnection) SetReadDeadline(t time.Time) error {
	// Not required
	return nil
}

func (c *clientConnection) SetWriteDeadline(t time.Time) error {
	// Not required
	return nil
}
