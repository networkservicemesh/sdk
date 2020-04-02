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
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientListener - inform server about new client is arrived or disconnected.
type ClientListener interface {
	ClientConnected(id string)
	ClientDisconnected(id string)
}

// Server - a callback server, hold client callback connections and allow to provide client
type Server interface {
	NewClient(ctx context.Context, id string) (grpc.ClientConnInterface, error)

	AddListener(listener ClientListener)
	CallbackServiceServer
}

type serverImpl struct {
	connections      map[string]*serverClientConnImpl
	listeners        []ClientListener
	lock             sync.Mutex
	identityProvider IdentityProvider
}

// IdentityProvider - A function to retrieve identity from grpc connection context and create clients based on it.
type IdentityProvider func(ctx context.Context) string

// IdentityByAuthority - return identity by :authority
func IdentityByAuthority(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		logrus.Errorf("Not metadata provided")
	}
	return md.Get(":authority")[0]
}

// NewServer - creates a new callback server to handle clients, should be used to create client connections back to client.
func NewServer(provider IdentityProvider) Server {
	return &serverImpl{
		identityProvider: provider,
		connections:      map[string]*serverClientConnImpl{},
	}
}

// AddListener - add listener to client, to be informed abount new clients are joined.
func (s *serverImpl) AddListener(listener ClientListener) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.listeners = append(s.listeners, listener)
	for k := range s.connections {
		listener.ClientConnected(k)
	}
}

// HandleCallbacks - main entry point for server, handle client stream here.
func (s *serverImpl) HandleCallbacks(serverClient CallbackService_HandleCallbacksServer) error {
	key := s.identityProvider(serverClient.Context())
	s.addConnection(key, serverClient)
	defer s.removeConnection(key)

	// Wait until context will be complete.
	ctx := serverClient.Context()
	<-ctx.Done()
	return ctx.Err()
}

func (s *serverImpl) addConnection(key string, serverClient CallbackService_HandleCallbacksServer) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.connections[key] = &serverClientConnImpl{
		server:   serverClient,
		id:       key,
		msgID:    0,
		streams:  map[string]responseChannel{},
		messages: make(chan *Request),
	}

	for _, l := range s.listeners {
		l.ClientConnected(key)
	}
}

func (s *serverImpl) removeConnection(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.connections, key)
	for _, l := range s.listeners {
		l.ClientDisconnected(key)
	}
}

type responseChannel chan *Response

type serverClientConnImpl struct {
	server   CallbackService_HandleCallbacksServer
	id       string
	msgID    int64
	lock     sync.Mutex
	streams  map[string]responseChannel
	messages chan *Request
	ctx      context.Context
}

func (s *serverClientConnImpl) handleMessages() {
	// Handle all messages from client and dispatch to required queues.
	for {
		select {
		case <-s.ctx.Done():
		default:
			msg, err := s.server.Recv()
			if err != nil {
				if s.isShowError(err) {
					logrus.Errorf("Server: Error receiving message: %v", err)
				}
				return
			}
			var msgChan responseChannel
			ok := false

			msgChan, ok = s.getStream(msg.Method, msg.OpId)

			if !ok {
				logrus.Errorf("There is no recipient for message: %v", msg)
				continue
			}
			msgChan <- msg
		}
	}
}

func (s *serverClientConnImpl) isShowError(err error) bool {
	return err != nil && s.ctx.Err() == nil && !strings.Contains(err.Error(), "context canceled")
}

func (s *serverClientConnImpl) getStream(method string, id int64) (responseChannel, bool) {
	s.lock.Lock()
	// Forward messages to required channel
	msgChan, ok := s.streams[streamKey(method, id)]
	s.lock.Unlock()
	return msgChan, ok
}

func streamKey(method string, id int64) string {
	return fmt.Sprintf("%s/%v", method, id)
}

func (s *serverClientConnImpl) sendMessages() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg := <-s.messages:
			err := s.server.Send(msg)
			if s.isShowError(err) {
				logrus.Errorf("Failed to send message: %v err: %v", msg, err)
			}
		}
	}
}

// Invoke - perform invoke from server to client.
func (s *serverClientConnImpl) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	//
	msg, ok := args.(proto.Message)
	replyMsg, okReply := reply.(proto.Message)
	if !ok || !okReply {
		return errors.Errorf("Arguments: %v -> %v are not complain to protobuf messages.", args, reply)
	}

	var err error

	var anyArgs *any.Any
	var anyReply *any.Any

	anyArgs, err = ptypes.MarshalAny(msg)
	if err != nil {
		return errors.Errorf("Failed to convert %v to any cause: %v", msg, err)
	}

	anyReply, err = ptypes.MarshalAny(replyMsg)
	if err != nil {
		return errors.Errorf("Failed to convert %v to any cause: %v", msg, err)
	}

	req := &Request{
		Mode:   Mode_Invoke,
		Method: method,
		Args:   anyArgs,
		Reply:  anyReply,
	}

	invokeChan, id := s.registerChannel(method)
	req.OpId = id

	// Remove response channel from list
	defer s.unregisterChannel(method, id)

	// Send message
	s.messages <- req

	var response *Response
	response, err = s.waitResponse(ctx, invokeChan, req.Method, req.OpId)
	if err != nil {
		return err
	}
	err = ptypes.UnmarshalAny(response.Response, replyMsg)
	if err != nil {
		return err
	}
	if response.Status.Code != 0 {
		return errors.New(response.Status.Message)
	}
	return nil
}

func (s *serverClientConnImpl) waitResponse(ctx context.Context, invokeChan responseChannel, method string, id int64) (*Response, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case response := <-invokeChan:
			if response.OpId != id && response.Method != method {
				logrus.Infof("Receive some old invoke msg: %v continue to wait", response)
				break
			}
			return response, nil
		}
	}
}

func (s *serverClientConnImpl) unregisterChannel(method string, id int64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.streams, streamKey(method, id))
}

func (s *serverClientConnImpl) registerChannel(method string) (resultChan chan *Response, msgID int64) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.msgID++

	resultChan = make(responseChannel)
	msgID = s.msgID

	s.streams[streamKey(method, msgID)] = resultChan
	return
}

type clientStream struct {
	resultChan responseChannel
	id         int64
	desc       *grpc.StreamDesc
	method     string
	ctx        context.Context
	s          *serverClientConnImpl
}

// Header - perform request for stream Header metadata.
func (c *clientStream) Header() (metadata.MD, error) {
	return c.requestMD(Mode_Header)
}

func (c *clientStream) requestMD(mode Mode) (metadata.MD, error) {
	req := &Request{
		Mode:   mode,
		Method: c.method,
		OpId:   c.id,
	}
	c.s.messages <- req

	resp, err := c.s.waitResponse(c.ctx, c.resultChan, req.Method, req.OpId)
	if err != nil {
		c.s.unregisterChannel(req.Method, req.OpId)
		return nil, err
	}
	// Decode response.
	md := &MD{}
	err = ptypes.UnmarshalAny(resp.Response, md)
	if err != nil {
		return nil, err
	}

	resultMd := metadata.New(nil)

	for _, v := range md.Mds {
		resultMd.Append(v.Key, v.Value...)
	}
	return resultMd, nil
}

// Trailer - perform request for stream Trailer
func (c *clientStream) Trailer() metadata.MD {
	resp, _ := c.requestMD(Mode_Trailer)
	return resp
}

// CloseSend - send CloseSend to stream
func (c *clientStream) CloseSend() error {
	req := &Request{
		Mode:   Mode_Close,
		Method: c.method,
		OpId:   c.id,
	}
	c.s.messages <- req

	_, err := c.s.waitResponse(c.ctx, c.resultChan, c.method, c.id)
	return err
}

// Context - return a current client stream context
func (c *clientStream) Context() context.Context {
	return c.ctx
}

// SendMsg - send message from server to client
func (c *clientStream) SendMsg(m interface{}) error {
	msg, ok := m.(proto.Message)
	if !ok {
		return errors.Errorf("Failed to cast %v to proto.Message", m)
	}
	msgAny, err := ptypes.MarshalAny(msg)
	if err != nil {
		return errors.Errorf("Failed to convert %v to any cause: %v", msg, err)
	}

	req := &Request{
		Mode:   Mode_Msg,
		Method: c.method,
		OpId:   c.id,
		Args:   msgAny,
	}
	c.s.messages <- req

	_, err = c.s.waitResponse(c.ctx, c.resultChan, c.method, c.id)
	if err != nil {
		return err
	}

	return nil
}

// RecvMsg - receive message from client stream
func (c *clientStream) RecvMsg(m interface{}) error {
	replyMsg, ok := m.(proto.Message)
	if !ok {
		return errors.Errorf("Failed to cast %v to proto.Message", m)
	}
	msgAny, err := ptypes.MarshalAny(replyMsg)
	if err != nil {
		return errors.Errorf("Failed to convert %v to any cause: %v", replyMsg, err)
	}

	req := &Request{
		Mode:   Mode_Recv,
		Method: c.method,
		OpId:   c.id,
		Reply:  msgAny,
	}
	c.s.messages <- req
	response, err := c.s.waitResponse(c.ctx, c.resultChan, c.method, c.id)
	if err != nil {
		return err
	}
	err = ptypes.UnmarshalAny(response.Response, replyMsg)
	if err != nil {
		return err
	}
	if response.Status.Code != 0 {
		return errors.New(response.Status.Message)
	}
	return nil
}

// NewStream -
func (s *serverClientConnImpl) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	resultChan, id := s.registerChannel(method)

	result := &clientStream{
		resultChan: resultChan,
		id:         id,
		desc:       desc,
		method:     method,
		ctx:        ctx,
		s:          s,
	}

	descr := &StreamDescriptor{
		ClientStreams: desc.ClientStreams,
		ServerStreams: desc.ServerStreams,
		StreamName:    desc.StreamName,
	}

	// Need to send new Stream request to start a real stream.
	req := &Request{
		Mode:   Mode_NewStream,
		Method: method,
		OpId:   id,
		Descr:  descr,
	}
	s.messages <- req

	_, err := s.waitResponse(ctx, resultChan, req.Method, req.OpId)
	if err != nil {
		s.unregisterChannel(req.Method, req.OpId)
		return nil, err
	}
	return result, nil
}

func (s *serverImpl) NewClient(ctx context.Context, id string) (grpc.ClientConnInterface, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	srv, ok := s.connections[id]
	if ok {
		if srv.msgID > 0 {
			return nil, errors.New("Client is already created")
		}
		srv.msgID++
		srv.ctx = ctx
		go srv.sendMessages()
		go srv.handleMessages()
		return srv, nil
	}
	return nil, errors.New("Client not found")
}
