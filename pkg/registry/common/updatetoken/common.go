// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package updatetoken provides chain elements to update token and expiration time of NetworkService.Path and NetworkServiceEndpoint.Path
package updatetoken

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// TODO: move to API (path_helper functions)
func GetPrevPathSegment(path *registry.Path) *registry.PathSegment {
	if path == nil {
		return nil
	}
	if len(path.GetPathSegments()) == 0 {
		return nil
	}
	if int(path.GetIndex()) == 0 {
		return nil
	}
	if int(path.GetIndex())-1 > len(path.GetPathSegments()) {
		return nil
	}
	return path.GetPathSegments()[path.GetIndex()-1]
}

func updateToken(ctx context.Context, path *registry.Path, tokenGenerator token.GeneratorFunc) error {
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

	// Generate the token
	token, expireTime, err := tokenGenerator(authInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	// Update the PathSegment
	path.GetPathSegments()[path.GetIndex()].Token = token
	path.GetPathSegments()[path.GetIndex()].Expires = timestamppb.New(expireTime)
	return nil
}
