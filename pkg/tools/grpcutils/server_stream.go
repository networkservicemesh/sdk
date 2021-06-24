// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package grpcutils

import (
	"context"

	"google.golang.org/grpc"
)

// ServerStream is a grpc.ServerStream wrapper with changed context
type ServerStream struct {
	ctx context.Context

	grpc.ServerStream
}

// Context return ss.ctx
func (ss *ServerStream) Context() context.Context {
	return ss.ctx
}

// WrapServerStreamContext wraps given grpc.ServerStream into ServerStream with ctx = wrapper(ss.Context())
func WrapServerStreamContext(ss grpc.ServerStream, wrapper func(ctx context.Context) context.Context) *ServerStream {
	return &ServerStream{
		ctx:          wrapper(ss.Context()),
		ServerStream: ss,
	}
}
