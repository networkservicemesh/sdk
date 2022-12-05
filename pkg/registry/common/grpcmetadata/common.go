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

	"google.golang.org/grpc/metadata"
)

type pathContextkey string

const (
	pathContextKey pathContextkey = "pathContextKey"
)

// PathFromContext returns Path from context if it exists
func PathFromContext(ctx context.Context) *Path {
	if value, ok := ctx.Value(pathContextKey).(*Path); ok {
		return value
	}

	return &Path{}
}

// PathWithContext puts Path to context
func PathWithContext(ctx context.Context, path *Path) context.Context {
	return context.WithValue(ctx, pathContextKey, path)
}

func loadFromMetadata(md metadata.MD) (*Path, error) {
	pathValue, loaded := md["path"]
	if !loaded {
		return nil, errors.New("failed to load path from grpc metadata")
	}

	path := &Path{}
	err := json.Unmarshal([]byte(pathValue[0]), path)
	if err != nil {
		return nil, err
	}

	return path, nil
}

func appendToMetadata(ctx context.Context, path *Path) (context.Context, error) {
	bytes, err := json.Marshal(path)
	if err != nil {
		return nil, err
	}
	ctx = metadata.AppendToOutgoingContext(ctx, pathKey, string(bytes))
	return ctx, nil
}
