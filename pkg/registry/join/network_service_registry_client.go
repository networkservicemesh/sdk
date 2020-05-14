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

package join

import (
	"context"
	"google.golang.org/grpc/metadata"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
)

type networkServiceRegistryClient struct {
	clients []registry.NetworkServiceRegistryClient
}

func (n *networkServiceRegistryClient) RegisterNSE(ctx context.Context, in *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	var result *registry.NSERegistration

	for _, c := range n.clients {
		r, err := c.RegisterNSE(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
		if result == nil {
			result = r
		}
	}

	return result, nil
}

type joinedServiceRegistryBulkNSEClient struct {
	clients []registry.NetworkServiceRegistry_BulkRegisterNSEClient
}

func (j *joinedServiceRegistryBulkNSEClient) Send(r *registry.NSERegistration) error {
	for _, c := range j.clients {
		if err := c.Send(r); err != nil {
			return err
		}
	}
	return nil
}

func (j *joinedServiceRegistryBulkNSEClient) Recv() (*registry.NSERegistration, error) {
	var r *registry.NSERegistration
	for _, c := range j.clients {
		if msg, err := c.Recv(); err != nil {
			return nil, err
		} else if r == nil {
			r = msg
		}
	}
	return r, nil
}

func (j *joinedServiceRegistryBulkNSEClient) Header() (metadata.MD, error) {
	if len(j.clients) > 0 {
		return j.clients[0].Header()
	}
	return j.clients[0].Header()
}

func (j *joinedServiceRegistryBulkNSEClient) Trailer() metadata.MD {
	if len(j.clients) > 0 {
		return j.clients[0].Trailer()
	}
	return nil
}

func (j *joinedServiceRegistryBulkNSEClient) CloseSend() error {
	for _, c := range j.clients {
		if err := c.CloseSend(); err != nil {
			return err
		}
	}
	return nil
}

func (j *joinedServiceRegistryBulkNSEClient) Context() context.Context {
	var contexts []context.Context
	for _, c := range j.clients {
		contexts = append(contexts, c.Context())
	}
	return Context(contexts...)
}

func (j *joinedServiceRegistryBulkNSEClient) SendMsg(m interface{}) error {
	for _, c := range j.clients {
		if err := c.SendMsg(m); err != nil {
			return err
		}
	}
	return nil
}

func (j *joinedServiceRegistryBulkNSEClient) RecvMsg(m interface{}) error {
	for _, c := range j.clients {
		if err := c.RecvMsg(m); err != nil {
			return err
		}
	}
	return nil
}

func (n *networkServiceRegistryClient) BulkRegisterNSE(ctx context.Context, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	var clients []registry.NetworkServiceRegistry_BulkRegisterNSEClient
	for _, c := range n.clients {
		r, err := c.BulkRegisterNSE(ctx, opts...)
		if err != nil {
			return nil, err
		}
		clients = append(clients, r)
	}
	return &joinedServiceRegistryBulkNSEClient{clients: clients}, nil
}

func (n *networkServiceRegistryClient) RemoveNSE(ctx context.Context, in *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	for _, c := range n.clients {
		_, err := c.RemoveNSE(ctx, in, opts...)
		if err != nil {
			return nil, err
		}
	}

	return new(empty.Empty), nil
}

func NewNetworkServiceRegistryClient(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return &networkServiceRegistryClient{clients: clients}
}

var _ registry.NetworkServiceRegistryClient = &networkServiceRegistryClient{}
