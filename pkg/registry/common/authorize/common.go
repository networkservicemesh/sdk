// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package authorize

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// RegistryOpaInput represents input for policies in authorizNSEServer and authorizeNSServer.
type RegistryOpaInput struct {
	ResourceID         string                      `json:"resource_id"`
	ResourceName       string                      `json:"resource_name"`
	ResourcePathIdsMap map[string][]string         `json:"resource_path_ids_map"`
	PathSegments       []*grpcmetadata.PathSegment `json:"path_segments"`
	Index              uint32                      `json:"index"`
}

// Policy represents authorization policy for network service.
type Policy interface {
	// Name returns policy name
	Name() string
	// Check checks authorization
	Check(ctx context.Context, input interface{}) error
}

type policiesList []Policy

func (l *policiesList) check(ctx context.Context, input RegistryOpaInput) error {
	if l == nil {
		return nil
	}
	for _, policy := range *l {
		if policy == nil {
			continue
		}

		if err := policy.Check(ctx, input); err != nil {
			log.FromContext(ctx).Errorf("policy failed: %v", policy.Name())
			return errors.Wrap(err, "registry: an error occurred during authorization policy check")
		}
	}
	return nil
}

func getRawMap(m *genericsync.Map[string, []string]) map[string][]string {
	rawMap := make(map[string][]string)
	m.Range(func(key string, value []string) bool {
		rawMap[key] = value
		return true
	})

	return rawMap
}

func getSpiffeIDFromPath(ctx context.Context, path *grpcmetadata.Path) spiffeid.ID {
	if len(path.PathSegments) == 0 {
		log.FromContext(ctx).Warn("can't get spiffe id from empty path")
		return spiffeid.ID{}
	}
	tokenString := path.PathSegments[0].Token

	claims := jwt.MapClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(tokenString, &claims)
	if err != nil {
		log.FromContext(ctx).Warnf("failed to parse jwt token: %s", err.Error())
		return spiffeid.ID{}
	}

	sub, ok := claims["sub"]
	if !ok {
		log.FromContext(ctx).Warn("failed to get field 'sub' from jwt token payload")
		return spiffeid.ID{}
	}
	subString, ok := sub.(string)
	if !ok {
		log.FromContext(ctx).Warn("failed to convert field 'sub' from jwt token payload to string")
		return spiffeid.ID{}
	}

	id, err := spiffeid.FromString(subString)
	if err != nil {
		log.FromContext(ctx).Warnf("failed to parse spiffeid from string: %s", err.Error())
		return spiffeid.ID{}
	}
	return id
}

func getLeftSideOfPath(path *grpcmetadata.Path) *grpcmetadata.Path {
	if len(path.PathSegments) == 0 {
		return &grpcmetadata.Path{
			Index:        0,
			PathSegments: []*grpcmetadata.PathSegment{},
		}
	}
	return &grpcmetadata.Path{
		Index:        path.Index,
		PathSegments: path.PathSegments[:path.Index+1],
	}
}
