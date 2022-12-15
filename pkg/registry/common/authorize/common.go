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

package authorize

import (
	"context"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// RegistryOpaInput represents input for policies in authorizNSEServer and authorizeNSServer
type RegistryOpaInput struct {
	ResourceID         string                      `json:"resource_id"`
	ResourceName       string                      `json:"resource_name"`
	ResourcePathIdsMap map[string][]string         `json:"resource_path_ids_map"`
	PathSegments       []*grpcmetadata.PathSegment `json:"path_segments"`
	Index              uint32                      `json:"index"`
}

// Policy represents authorization policy for network service.
type Policy interface {
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
			log.FromContext(ctx).Infof("policy failed %v", policy)
			return err
		}

		log.FromContext(ctx).Infof("policy passed")
	}
	return nil
}

func getRawMap(m *PathIdsMap) map[string][]string {
	rawMap := make(map[string][]string)
	m.Range(func(key string, value []string) bool {
		rawMap[key] = value
		return true
	})

	return rawMap
}

func getSpiffeIDFromPath(path *grpcmetadata.Path) (spiffeid.ID, error) {
	tokenString := path.PathSegments[0].Token

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
