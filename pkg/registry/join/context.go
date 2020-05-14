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

package join

import (
	"context"
	"time"
)

type joinedContext struct {
	contexts []context.Context
}

func (j joinedContext) Deadline() (deadline time.Time, ok bool) {
	var result time.Time
	for _, c := range j.contexts {
		d, o := c.Deadline()
		if !o {
			return d, o
		}
		if result.After(deadline) {
			result = deadline
		}
	}
	return result, true
}

func (j joinedContext) Done() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		for _, ctx := range j.contexts {
			<-ctx.Done()
		}
		close(ch)
	}()
	return ch
}

func (j joinedContext) Err() error {
	for _, c := range j.contexts {
		if err := c.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (j joinedContext) Value(key interface{}) interface{} {
	for _, c := range j.contexts {
		if v := c.Value(key); v != nil {
			return v
		}
	}
	return nil
}

func Context(ctx ...context.Context) context.Context {
	return &joinedContext{
		contexts: ctx,
	}
}
