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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	clientListener      net.Listener
	streams             map[string]grpc.ClientStream
	sendChannel         chan *Response
	clientClient        *grpc.ClientConn
	scClient            CallbackServiceClient
	clientConnInterface grpc.ClientConnInterface
	clientGRPCServer    *grpc.Server
	socketFile          string
}

func (c *callbackClient) Context() context.Context {
	return c.ctx
}

func (c *callbackClient) Stop() {
	c.cancel()
	close(c.errorChan)
	_ = c.clientListener.Close()
	_ = os.Remove(c.socketFile)
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
		streams:             map[string]grpc.ClientStream{},
		sendChannel:         make(chan *Response),
	}
	return serveCtx
}

const socketMask = 0077

func (c *callbackClient) Serve(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	c.ctx = ctx
	c.cancel = cancel
	var err error
	c.socketFile = TempSocketFile()
	c.clientListener, err = net.Listen("unix", c.socketFile)

	if err != nil {
		c.errorChan <- err
		return
	}

	go func() {
		serveErr := c.clientGRPCServer.Serve(c.clientListener)
		if serveErr != nil {
			c.errorChan <- serveErr
		}
	}()

	go func() {
		addr := fmt.Sprintf("unix:%v", c.socketFile)
		for {
			if ctx.Err() != nil {
				// Perform exit if context is done.
				return
			}
			c.clientClient, err = grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				logrus.Errorf("Failed to start local GRPC %v", err)
				c.errorChan <- err
				continue
			}
			c.scClient = NewCallbackServiceClient(c.clientConnInterface)
			// Client call handle to clientConnInterface
			cl, err := c.scClient.HandleCallbacks(ctx)
			if err != nil {
				logrus.Infof("Error: %v. try again", err)
				continue
			}

			// Send go routine
			go c.handleSendStream(ctx, cl)
			c.handleRecvStream(ctx, cl)
		}
	}()
}

func (c *callbackClient) handleRecvStream(ctx context.Context, cl CallbackService_HandleCallbacksClient) {
	for {
		if ctx.Err() != nil {
			// Perform exit if context is done.
			return
		}
		var req *Request
		req, err := cl.Recv()
		if err != nil {
			if c.isShowError(ctx, err) {
				logrus.Errorf("ClientError receive message: %v", err)
			}
			return
		}
		switch req.Mode {
		case Mode_Invoke:
			c.handleInvoke(req)
		case Mode_Header:
			c.handleSendHeader(req)
		case Mode_Trailer:
			c.handleSendTrailer(req)
		case Mode_NewStream:
			c.handleNewStream(req)
		case Mode_Close:
			c.handleStreamClose(req)
		case Mode_Msg:
			c.handleMsg(req)
		case Mode_Recv:
			c.handleRecv(req)
		}
	}
}

func (c *callbackClient) isShowError(ctx context.Context, err error) bool {
	return ctx.Err() != nil && !strings.Contains(err.Error(), "transport is closing") && !strings.Contains(err.Error(), "context canceled")
}

// TempSocketFile - creates a temp socket file with prefix "callback.socket"
func TempSocketFile() string {
	socketFile, _ := ioutil.TempFile("", "callback.socket")
	_ = socketFile.Close()
	_ = os.Remove(socketFile.Name())
	unix.Umask(socketMask)
	return socketFile.Name()
}

func (c *callbackClient) handleSendStream(ctx context.Context, cl CallbackService_HandleCallbacksClient) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.sendChannel:
			sendErr := cl.Send(msg)
			if sendErr != nil {
				logrus.Errorf("Err: %v", sendErr)
				return
			}
		}
	}
}

func (c *callbackClient) handleSendTrailer(req *Request) {
	stream := c.streams[streamKey(req.Method, req.OpId)]
	if stream == nil {
		c.sendError(req, "Stream not found")
	} else {
		md := stream.Trailer()
		c.sendMD(md, req)
	}
}

func (c *callbackClient) handleSendHeader(req *Request) {
	stream := c.streams[streamKey(req.Method, req.OpId)]
	if stream == nil {
		c.sendError(req, "Stream not found")
	} else {
		md, err := stream.Header()
		if err != nil {
			c.sendError(req, err.Error())
		} else {
			c.sendMD(md, req)
		}
	}
}

func (c *callbackClient) handleNewStream(req *Request) {
	descr := &grpc.StreamDesc{
		StreamName:    req.Descr.StreamName,
		ServerStreams: req.Descr.ServerStreams,
		ClientStreams: req.Descr.ClientStreams,
	}
	stream, err := c.clientClient.NewStream(c.ctx, descr, req.Method)
	if err != nil {
		c.sendError(req, err.Error())
		return
	}
	c.streams[streamKey(req.Method, req.OpId)] = stream
	// Send response with 0 error code.
	c.sendEmptyResponse(req)
}

func (c *callbackClient) sendEmptyResponse(req *Request) {
	scResponse := &Response{
		Method: req.Method,
		OpId:   req.OpId,
		Status: &ResponseStatus{
			Code:    0,
			Message: "",
		},
		Mode: req.Mode,
	}
	c.sendChannel <- scResponse
}

func (c *callbackClient) sendError(req *Request, errorMsg string) {
	scResponse := &Response{
		Method: req.Method,
		OpId:   req.OpId,
		Status: &ResponseStatus{
			Code:    1,
			Message: errorMsg,
		},
		Mode: Mode_Error,
	}
	c.sendChannel <- scResponse
}

func (c *callbackClient) sendMD(md metadata.MD, req *Request) {
	mdMsg := &MD{}
	for k, v := range md {
		mdMsg.Mds = append(mdMsg.Mds, &MDValue{
			Key:   k,
			Value: v,
		})
	}

	respMsg, err := ptypes.MarshalAny(mdMsg)
	if err != nil {
		c.sendError(req, err.Error())
		return
	}
	scResponse := &Response{
		Method: req.Method,
		OpId:   req.OpId,
		Status: &ResponseStatus{
			Code:    0,
			Message: "",
		},
		Response: respMsg,
		Mode:     req.Mode,
	}
	c.sendChannel <- scResponse
}

func (c *callbackClient) handleInvoke(req *Request) {
	argsPb := &ptypes.DynamicAny{}
	err := ptypes.UnmarshalAny(req.Args, argsPb)
	if err != nil {
		c.sendError(req, err.Error())
		return
	}
	replyPb := &ptypes.DynamicAny{}
	err = ptypes.UnmarshalAny(req.Reply, replyPb)
	if err != nil {
		c.sendError(req, err.Error())
		return
	}

	// We reuse same context
	err = c.clientClient.Invoke(c.ctx, req.Method, argsPb.Message, replyPb.Message)

	if err != nil {
		c.sendError(req, err.Error())
		return
	}

	var respMsg *any.Any
	respMsg, err = ptypes.MarshalAny(replyPb.Message)
	if err != nil {
		c.sendError(req, err.Error())
		return
	}
	resp := &Response{
		Response: respMsg,
		Status: &ResponseStatus{
			Message: "",
			Code:    0,
		},
		Method: req.Method,
		OpId:   req.OpId,
		Mode:   Mode_Invoke,
	}

	// Send result back
	c.sendChannel <- resp
}

func (c *callbackClient) handleStreamClose(req *Request) {
	stream := c.streams[streamKey(req.Method, req.OpId)]
	if stream == nil {
		c.sendError(req, "Stream not found")
		return
	}
	if err := stream.CloseSend(); err != nil {
		c.sendError(req, err.Error())
	} else {
		c.sendEmptyResponse(req)
	}
}

func (c *callbackClient) handleMsg(req *Request) {
	stream := c.streams[streamKey(req.Method, req.OpId)]
	if stream == nil {
		c.sendError(req, "Stream not found")
		return
	}

	argsPb := &ptypes.DynamicAny{}
	err := ptypes.UnmarshalAny(req.Args, argsPb)

	if err != nil {
		c.sendError(req, err.Error())
		return
	}

	if err := stream.SendMsg(argsPb.Message); err != nil {
		c.sendError(req, err.Error())
	} else {
		c.sendEmptyResponse(req)
	}
}

func (c *callbackClient) handleRecv(req *Request) {
	stream := c.streams[streamKey(req.Method, req.OpId)]
	if stream == nil {
		c.sendError(req, "Stream not found")
		return
	}

	argsPb := &ptypes.DynamicAny{}
	err := ptypes.UnmarshalAny(req.Reply, argsPb)

	if err != nil {
		c.sendError(req, err.Error())
		return
	}

	if err := stream.RecvMsg(argsPb.Message); err != nil {
		c.sendError(req, err.Error())
	} else {
		var recvAny *any.Any
		recvAny, err = ptypes.MarshalAny(argsPb.Message)
		if err != nil {
			c.sendError(req, err.Error())
			return
		}
		scResponse := &Response{
			Method: req.Method,
			OpId:   req.OpId,
			Status: &ResponseStatus{
				Code:    0,
				Message: "",
			},
			Response: recvAny,
		}
		logrus.Infof("Send MSG: %v", argsPb.Message)
		c.sendChannel <- scResponse
	}
}
