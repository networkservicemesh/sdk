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

package grpcmetadata

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc/metadata"
)

type pathContextkey string

const (
	pathContextKey pathContextkey = "pathContextKey"
)

// PathFromContext returns registry.Path from context if it exists
func PathFromContext(ctx context.Context) (*registry.Path, error) {
	if value, ok := ctx.Value(pathContextKey).(*registry.Path); ok {
		return value, nil
	}

	return nil, errors.New("failed to get registry.Path from context")
}

// PathWithContext puts registry.Path to context
func PathWithContext(ctx context.Context, path *registry.Path) context.Context {
	return context.WithValue(ctx, pathContextKey, path)
}

func loadFromMetadata(md metadata.MD) (*registry.Path, error) {
	pathValue, loaded := md["path"]
	if !loaded {
		return nil, errors.New("failed to load path from grpc metadata")
	}

	path := &registry.Path{}
	err := json.Unmarshal([]byte(pathValue[0]), path)
	if err != nil {
		return nil, err
	}

	return path, nil
}

func appendToMetadata(ctx context.Context, path *registry.Path) (context.Context, error) {
	bytes, err := json.Marshal(path)
	if err != nil {
		return nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, pathKey, string(bytes))
	return ctx, nil
}
