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

// Package injectpeer is used to inject its peer object into the context if needed.
// It  is desirable to know the grpc.Peer information of your server, so for example, you can add
// a proper 'aud' claim to a JWT Token.
// golang grpc doesn't provide the grpc.Peer information to clients unless
// (1)  They ask for it with the grpc.Peer grpc.CallOption
// (2)  Even then, only on return.
// This chain element compensates for that by:
// (1)  Adding the grpc.Peer grpc.CallOption to its call to next.Client(ctx)
// (2)  Keeping a running record on the return of the grpc.Peer
// (3)  Injecting that grpc.Peer into the context used to call next.Client(ctx) once it has a grpc.Peer on record
// (4)  If and only if it it doesn't havea grpc.Peer and its call to next.Client(ctx) return an error, tries doing
//      the call again *with* the grpc.Peer added to the context to see if that clears up the error.
package injectpeer

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type injectPeerClient struct {
	peer *peer.Peer
	serialize.Executor
}

// NewClient - returns a NetworkServiceClient chain element that injects the grpc.Peer into the context for the next call
func NewClient() networkservice.NetworkServiceClient {
	return &injectPeerClient{}
}

func (i *injectPeerClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Add Peer to context
	ctx = i.addPeerToContext(ctx)
	// Make sure we get the new Peer information on return (in case we need it)
	var newPeer peer.Peer
	opts = append(opts, grpc.Peer(&newPeer))

	// Call the next
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		// If we didn't have a peer in the last call, and we got one back, try again,once
		if _, exists := peer.FromContext(ctx); !exists && newPeer != (peer.Peer{}) {
			<-i.updatePeer(&newPeer)
			return i.Request(ctx, request, opts...)
		}
		return nil, err
	}
	i.updatePeer(&newPeer)
	return rv, nil
}

func (i *injectPeerClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// Add Peer to context
	ctx = i.addPeerToContext(ctx)
	// Make sure we get the new Peer information on return (in case we need it)
	var newPeer peer.Peer
	opts = append(opts, grpc.Peer(&newPeer))

	// Call the next
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		// If we didn't have a peer in the last call, and we got one back, try again,once
		if _, exists := peer.FromContext(ctx); !exists && newPeer != (peer.Peer{}) {
			<-i.updatePeer(&newPeer)
			return i.Close(ctx, conn, opts...)
		}
		return nil, err
	}
	i.updatePeer(&newPeer)
	return rv, nil
}

func (i *injectPeerClient) addPeerToContext(ctx context.Context) context.Context {
	<-i.AsyncExec(func() {
		if i.peer != nil && *i.peer != (peer.Peer{}) {
			ctx = peer.NewContext(ctx, i.peer)
		}
	})
	return ctx
}

func (i *injectPeerClient) updatePeer(p *peer.Peer) <-chan struct{} {
	return i.AsyncExec(func() {
		if (peer.Peer{}) != *p {
			i.peer = p
		}
	})
}
