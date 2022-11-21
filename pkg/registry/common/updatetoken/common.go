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

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

func updateToken(ctx context.Context, path *grpcmetadata.Path, tokenGenerator token.GeneratorFunc) error {
	// Make sure index isn't out of bound
	if path == nil || int(path.Index) >= len(path.PathSegments) {
		return errors.Errorf("NetworkServiceRequest.Connection.Path.Index(%d) >= len(NetworkServiceRequest.Connection.Path.PathSegments)(%d)",
			path.Index, len(path.PathSegments))
	}

	// Extract the authInfo:
	var authInfo credentials.AuthInfo
	if p, exists := peer.FromContext(ctx); exists {
		authInfo = p.AuthInfo
	}

	// Generate the token
	tok, expireTime, err := tokenGenerator(authInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	// Update the PathSegment
	path.PathSegments[path.Index].Token = tok
	path.PathSegments[path.Index].Expires = expireTime

	return nil
}

func getIDFromToken(tokenString string) (spiffeid.ID, error) {
	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, &claims)
	if err != nil {
		return spiffeid.ID{}, errors.Errorf("failed to parse jwt token: %s", err.Error())
	}

	sub, ok := claims["sub"]
	if !ok {
		return spiffeid.ID{}, errors.New("failed to get field 'sub' from jwt token payload")
	}
	subString, ok := sub.(string)
	if !ok {
		return spiffeid.ID{}, errors.New("failed to convert field 'sub' from jwt token payload to string")
	}
	return spiffeid.FromString(subString)
}

func updatePathIds(pathIds []string, index int, id string) []string {
	if index >= len(pathIds) {
		pathIds = append(pathIds, id)
	} else {
		pathIds[index] = id
	}

	return pathIds
}
