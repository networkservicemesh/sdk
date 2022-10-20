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
	"encoding/json"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// TODO: Rename ResourceSpiffeID field
// RegistryOpaInput represents input for policies in authorizNSEServer and authorizeNSServer
type RegistryOpaInput struct {
	ResourceSpiffeID     string                  `json:"resource_spiffe_id"`
	ResourceName         string                  `json:"resource_name"`
	SpiffeIDResourcesMap map[string][]string     `json:"spiffe_id_resources_map"`
	PathSegments         []*registry.PathSegment `json:"path_segments"`
	Index                uint32                  `json:"index"`
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
	logger := log.FromContext(ctx)
	for _, policy := range *l {
		if policy == nil {
			continue
		}

		if err := policy.Check(ctx, input); err != nil {
			logger.Info("Policy failed")
			return err
		}

		logger.Info("Policy passed")
	}
	return nil
}

func getRawMap(m *ResourcePathMap) map[string][]string {
	rawMap := make(map[string][]string)
	m.Range(func(key string, value []string) bool {
		rawMap[key] = value
		return true
	})

	return rawMap
}

func getSpiffeIDFromPath(path *registry.Path) (spiffeid.ID, error) {
	tokenString := path.PathSegments[0].Token

	tokenSegments := strings.Split(tokenString, ".")
	if len(tokenSegments) < 3 {
		return spiffeid.ID{}, errors.New("token is invalid. Should have 3 segments separated by dot")
	}
	b, err := jwt.DecodeSegment(tokenSegments[1])
	if err != nil {
		return spiffeid.ID{}, errors.Errorf("failed to decode payload from jwt token: %s", err.Error())
	}

	var payload struct {
		Sub string   `json:"sub"`
		Aud []string `json:"aud"`
	}
	err = json.Unmarshal(b, &payload)
	if err != nil {
		return spiffeid.ID{}, errors.Errorf("failed to parse payload from json: %s", err.Error())
	}

	return spiffeid.FromString(payload.Sub)
}

func printPath(ctx context.Context, path *registry.Path) {
	logger := log.FromContext(ctx)

	for i, s := range path.PathSegments {
		logger.Infof("Segment: %d, Value: %v", i, s)
	}
}
