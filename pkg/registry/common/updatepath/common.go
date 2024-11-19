// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022-2024 Cisco Systems, Inc.
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

// Package updatepath provides a chain element that sets the id of an incoming or outgoing request
package updatepath

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

func updatePath(path *grpcmetadata.Path, peerTok, tok string) (*grpcmetadata.Path, uint32, error) {
	if path == nil {
		return nil, 0, errors.New("updatePath cannot be called with a nil path")
	}

	// Empty path from client means first request
	// Create path segments for client and for current server
	if len(path.PathSegments) == 0 {
		path.PathSegments = append(path.PathSegments,
			&grpcmetadata.PathSegment{Token: peerTok},
			&grpcmetadata.PathSegment{Token: tok})
		path.Index = 1

		return path, path.Index - 1, nil
	}

	// Check if path.Index is correct
	currentIndex := int(path.Index) + 1
	if currentIndex > len(path.PathSegments) {
		return nil, 0, errors.Errorf("Path.Index+1==%d should be less or equal len(Path.PathSegments)==%d",
			currentIndex, len(path.PathSegments))
	}

	// Get previous and current PathSegment
	prev := path.GetCurrentPathSegment()
	path.Index++
	curr := path.GetCurrentPathSegment()

	// If we don't have current PathSegment, we're on first request
	// Add current PathSegment and update token in previous segment
	if curr == nil {
		prev.Token = peerTok
		path.PathSegments = append(path.PathSegments, &grpcmetadata.PathSegment{
			Token: tok,
		})

		return path, path.Index - 1, nil
	}

	// Current PathSegment exists. It means refresh
	// Update tokens in previous and current PathSegments
	prev.Token = peerTok
	curr.Token = tok
	return path, path.Index - 1, nil
}

func generateToken(ctx context.Context, tokenGenerator token.GeneratorFunc) (string, time.Time, error) {
	// Extract the authInfo:
	var authInfo credentials.AuthInfo
	if p, exists := peer.FromContext(ctx); exists {
		authInfo = p.AuthInfo
	}

	// Generate the token
	return tokenGenerator(authInfo)
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

func updatePathIDs(pathIDs []string, index int, id string) []string {
	if index >= len(pathIDs) {
		pathIDs = append(pathIDs, id)
	} else {
		pathIDs[index] = id
	}

	return pathIDs
}
