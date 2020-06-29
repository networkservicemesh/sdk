// Copyright (c) 2020 Cisco and/or its affiliates.
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

package clientmap

import (
	"context"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
)

type refcountClient struct {
	count  int32
	client networkservice.NetworkServiceClient
}

func (r *refcountClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return r.client.Request(ctx, request)
}

func (r *refcountClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return r.client.Close(ctx, conn)
}

// RefcountMap - clientmap.Map wrapped with refcounting.  Each Store,Load,LoadOrStore increments the count,
//               each Delete decrements the count.  When the count is zero, Delete deletes.
type RefcountMap struct {
	Map
}

// Store sets the value for a key.
// Increments the refcount
func (r *RefcountMap) Store(key string, value networkservice.NetworkServiceClient) {
	client := &refcountClient{client: value}
	r.Map.Store(key, client)
	atomic.AddInt32(&client.count, 1)
}

// LoadOrStore returns the existing value for the key if present. Otherwise, it stores and returns the given value. The loaded result is true if the value was loaded, false if stored.
// Increments the refcount.
func (r *RefcountMap) LoadOrStore(key string, value networkservice.NetworkServiceClient) (networkservice.NetworkServiceClient, bool) {
	client := &refcountClient{client: value}
	rv, loaded := r.Map.LoadOrStore(key, client)
	if client, ok := rv.(*refcountClient); ok {
		atomic.AddInt32(&client.count, 1)
		return client.client, loaded
	}
	return rv, loaded
}

// Load returns the value stored in the map for a key, or nil if no value is present. The ok result indicates whether value was found in the map.
// Increments refcount.
func (r *RefcountMap) Load(key string) (networkservice.NetworkServiceClient, bool) {
	rv, loaded := r.Map.Load(key)
	if client, ok := rv.(*refcountClient); ok {
		atomic.AddInt32(&client.count, 1)
		return client.client, loaded
	}
	return rv, loaded
}

// Delete decrements the refcount and deletes the value for a key if refcount is zero.
func (r *RefcountMap) Delete(key string) {
	rv, _ := r.Map.Load(key)
	if client, ok := rv.(*refcountClient); ok && atomic.AddInt32(&client.count, -1) == 0 {
		r.Map.Delete(key)
	}
}
