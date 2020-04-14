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

package updatepath

import (
	"context"

	"github.com/golang/protobuf/ptypes"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type commonUpdatePath struct {
	name           string
	tokenGenerator token.GeneratorFunc
}

func (u *commonUpdatePath) updatePath(ctx context.Context, conn *networkservice.Connection) error {
	// We need a real path if we don't have one
	if conn == nil {
		return errors.New("NetworkServiceRequest.Connection must not be nil")
	}

	// If we don't have a Path, add one
	if conn.GetPath() == nil {
		conn.Path = &networkservice.Path{}
	}

	path := conn.GetPath()

	// Make sure index isn't out of bound
	if (len(path.GetPathSegments()) > 0) && int(path.GetIndex()) >= len(path.GetPathSegments()) {
		return errors.Errorf("NetworkServiceRequest.Connection.Path.Index(%d) >= len(NetworkServiceRequest.Connection.Path.PathSegments)(%d)",
			path.GetIndex(), len(path.GetPathSegments()))
	}

	// If this isn't already one of our tokens for *this* connectionId ... then update the next one
	if (len(path.GetPathSegments()) > 0) && !(path.GetPathSegments()[path.GetIndex()].GetName() == u.name &&
		path.GetPathSegments()[path.GetIndex()].GetId() == conn.GetId()) {
		path.Index++
	}

	// If we don't already have a PathSegment at the new index... append an empty one
	if int(path.GetIndex()) >= len(path.GetPathSegments()) {
		path.PathSegments = append(path.PathSegments, &networkservice.PathSegment{})
	}

	// Extract the authInfo:
	var authInfo credentials.AuthInfo
	if p, exists := peer.FromContext(ctx); exists {
		authInfo = p.AuthInfo
	}

	// Generate the tok
	tok, expireTime, err := u.tokenGenerator(authInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	// Convert the expireTime to proto
	expires, err := ptypes.TimestampProto(expireTime)
	if err != nil {
		return errors.WithStack(err)
	}

	// Update the PathSegment
	path.GetPathSegments()[path.GetIndex()].Name = u.name
	path.GetPathSegments()[path.GetIndex()].Id = conn.GetId()
	path.GetPathSegments()[path.GetIndex()].Token = tok
	path.GetPathSegments()[path.GetIndex()].Expires = expires
	return nil
}
