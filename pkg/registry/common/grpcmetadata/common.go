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

package grpcmetadata

import (
	"context"
	"encoding/json"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type pathContextkey string

const (
	pathContextKey pathContextkey = "pathContextKey"
)

// PathFromContext returns Path from context. If path doesn't exist fuction returns empty Path
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

func fromMD(md metadata.MD) (*Path, error) {
	pathValue, loaded := md[pathKey]

	if !loaded {
		return nil, errors.New("failed to load path from grpc metadata")
	}

	path := &Path{}

	if len(pathValue) > 0 {
		err := json.Unmarshal([]byte(pathValue[len(pathValue)-1]), path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse a JSON-encoded data and store the result")
		}
	}

	return path, nil
}

func fromContext(ctx context.Context) (*Path, error) {
	md, loaded := metadata.FromIncomingContext(ctx)

	if !loaded {
		return nil, errors.New("failed to load grpc metadata from context")
	}

	return fromMD(md)
}

func appendToMetadata(ctx context.Context, path *Path) (context.Context, error) {
	bytes, err := json.Marshal(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert a provided path into JSON")
	}
	ctx = metadata.AppendToOutgoingContext(ctx, pathKey, string(bytes))
	return ctx, nil
}

func sendPath(ctx context.Context, path *Path) error {
	bytes, err := json.Marshal(path)
	if err != nil {
		return errors.Wrap(err, "failed to convert a provided path into JSON")
	}

	header := metadata.Pairs(pathKey, string(bytes))
	return grpc.SendHeader(ctx, header)
}

func nsFindServerSendPath(server registry.NetworkServiceRegistry_FindServer, path *Path) error {
	bytes, err := json.Marshal(path)
	if err != nil {
		return errors.Wrap(err, "failed to convert a provided path into JSON")
	}

	header := metadata.Pairs(pathKey, string(bytes))
	return server.SendHeader(header)
}

func nseFindServerSendPath(server registry.NetworkServiceEndpointRegistry_FindServer, path *Path) error {
	bytes, err := json.Marshal(path)
	if err != nil {
		return errors.Wrap(err, "failed to convert a provided path into JSON")
	}

	header := metadata.Pairs(pathKey, string(bytes))
	return server.SendHeader(header)
}
