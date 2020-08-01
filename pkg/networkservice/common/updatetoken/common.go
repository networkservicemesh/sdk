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

package updatetoken

import (
	"context"

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/golang/protobuf/ptypes"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

func updateToken(ctx context.Context, conn *networkservice.Connection, tokenGenerator token.GeneratorFunc) error {
	path := conn.GetPath()

	// Make sure index isn't out of bound
	if path == nil || int(path.GetIndex()) >= len(path.GetPathSegments()) {
		return errors.Errorf("NetworkServiceRequest.Connection.Path.Index(%d) >= len(NetworkServiceRequest.Connection.Path.PathSegments)(%d)",
			path.GetIndex(), len(path.GetPathSegments()))
	}

	// Extract the authInfo:
	var authInfo credentials.AuthInfo
	if p, exists := peer.FromContext(ctx); exists {
		authInfo = p.AuthInfo
	}

	// Generate the tok
	tok, expireTime, err := tokenGenerator(authInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	// Convert the expireTime to proto
	var expires *timestamp.Timestamp
	expires, err = ptypes.TimestampProto(expireTime)
	if err != nil {
		return errors.WithStack(err)
	}

	// Update the PathSegment
	path.GetPathSegments()[path.GetIndex()].Token = tok
	path.GetPathSegments()[path.GetIndex()].Expires = expires
	return nil
}
