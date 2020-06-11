// Copyright (c) 2020 Cisco Systems, Inc.
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

package adapters

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type networkServiceRegistryServer struct {
	client registry.NetworkServiceRegistryClient
}

func (n *networkServiceRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEntry) (*registry.NetworkServiceEntry, error) {
	return n.client.Register(ctx, request)
}

func (n *networkServiceRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	client, err := n.client.Find(s.Context(), query)
	if err != nil {
		return err
	}

	for {
		if err := client.Context().Err(); err != nil {
			return err
		}
		msg, err := client.Recv()
		if err != nil {
			return err
		}
		err = s.Send(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *networkServiceRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEntry) (*empty.Empty, error) {
	return n.client.Unregister(ctx, request)
}

// RegistryClientToServer - returns a registry.NetworkServiceRegistryClient wrapped around the supplied client
func RegistryClientToServer(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	return &networkServiceRegistryServer{client: client}
}

var _ registry.NetworkServiceRegistryServer = &networkServiceRegistryServer{}
