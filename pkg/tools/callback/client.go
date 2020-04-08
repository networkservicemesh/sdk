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

// Package callback - provide GRPC API to perform client callbacks.
package callback

import (
	"context"
	"errors"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Client - a callback server client, main functionality is to serve local grpc and perform operations on it.
type Client interface {
	// Context - return context
	Context() context.Context
	// Stop - stop all operations
	Stop()
	// Server - perform opertaions on local GRPC server and call for remote server.
	Serve(ctx context.Context)
	// ErrChan - return error stream
	ErrChan() chan error
}

type callbackClient struct {
	errorChan           chan error
	ctx                 context.Context
	cancel              context.CancelFunc
	clientListener      *clientListener
	scClient            CallbackServiceClient
	clientConnInterface grpc.ClientConnInterface
	clientGRPCServer    *grpc.Server
}

func (c *callbackClient) Context() context.Context {
	return c.ctx
}

func (c *callbackClient) Stop() {
	c.cancel()
	close(c.errorChan)
	_ = c.clientListener.Close()
}

func (c *callbackClient) ErrChan() chan error {
	return c.errorChan
}

// NewClient - construct a new callback client. It is used to connect server and transfer all call back to client.
func NewClient(clientConnInterface grpc.ClientConnInterface, clientServer *grpc.Server) Client {
	serveCtx := &callbackClient{
		clientConnInterface: clientConnInterface,
		clientGRPCServer:    clientServer,
		errorChan:           make(chan error, 10),
	}
	return serveCtx
}

type clientListener struct {
	addr        net.Addr
	connections chan net.Conn
}

func (c *clientListener) Accept() (net.Conn, error) {
	result := <-c.connections
	if result == nil {
		return nil, errors.New("connection is closed")
	}
	return result, nil
}

func (c *clientListener) Close() error {
	close(c.connections)
	return nil
}

func (c *clientListener) Addr() net.Addr {
	return c.addr
}

func (c *callbackClient) Serve(ctx context.Context) {
	c.clientListener = &clientListener{
		connections: make(chan net.Conn),
		addr: &net.UnixAddr{
			Name: "callback",
			Net:  "unix",
		},
	}

	go func() {
		serveErr := c.clientGRPCServer.Serve(c.clientListener)
		if serveErr != nil {
			c.errorChan <- serveErr
		}
	}()

	go func() {
		for {
			if ctx.Err() != nil {
				// Perform exit if context is done.
				return
			}

			var cancelFunc context.CancelFunc
			ctx, cancelFunc = context.WithCancel(ctx)
			defer cancelFunc()
			c.scClient = NewCallbackServiceClient(c.clientConnInterface)
			// Client call handle to clientConnInterface
			cl, err := c.scClient.HandleCallbacks(ctx)
			if err != nil {
				logrus.Infof("Error: %v. try again", err)
				c.errorChan <- err
				continue
			}

			c.clientListener.connections <- newConnection(ctx, cancelFunc, cl)
			<-ctx.Done()
			return
		}
	}()
}
