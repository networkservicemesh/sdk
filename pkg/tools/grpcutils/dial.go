// Copyright (c) 2020 Doc.ai and/or its affiliates.
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
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// DialContext tries to create a client connection to the given target during 1 minute.
// This function may be useful in the environments when not guarantee that the server socket is immediately provided.
func DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	return DialContextTimeout(ctx, target, time.Minute, opts...)
}

// DialContextTimeout tries to create a client connection to the given target during passed timeout.
// This function may be useful in the environments when not guarantee that the server socket is immediately provided.
func DialContextTimeout(ctx context.Context, target string, duration time.Duration, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	timeout := time.After(duration)
	for {
		select {
		case <-ctx.Done():
			return conn, errors.Wrap(err, ctx.Err().Error())
		case <-timeout:
			return conn, errors.Wrapf(err, "cannot connect to %v during %v", target, duration)
		default:
			conn, err = grpc.DialContext(ctx, target, opts...)
			if err == nil {
				return
			}
			<-time.After(time.Millisecond * 50)
		}
	}
}
