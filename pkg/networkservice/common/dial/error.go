// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package dial will dial up a grpc.ClientConnInterface if a client *url.URL is provided in the ctx, retrievable by
// clienturlctx.ClientURL(ctx) and put the resulting grpc.ClientConnInterface into the ctx using clientconn.Store(..)
// where it can be retrieved by other chain elements using clientconn.Load(...)
package dial

import (
	"context"
	"errors"
	"fmt"
)

type dialError struct {
	err error
}

func (e *dialError) Error() string {
	return fmt.Sprintf("dial error: %v", e.err)
}

func (e *dialError) Unwrap() error {
	return e.err
}

func wrapDialError(ctx context.Context, err error) error {
	// if request context has error,
	// then any dial error likely has nothing to do with actual connection issues
	if ctx.Err() != nil {
		fmt.Println("nkloqjr: wrapDialError ctx skip")
		return err
	}
	// in case we use a blocking dial, connection error will result in dial ctx deadline
	if errors.Is(err, context.DeadlineExceeded) {
		return &dialError{err}
	}
	return err
}

func IsDialError(err error) bool {
	for err != nil {
		if _, ok := err.(*dialError); ok {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
