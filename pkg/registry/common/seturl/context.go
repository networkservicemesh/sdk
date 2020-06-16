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

package seturl

import (
	"context"
	"net/url"
)

const (
	endpointURL endpointURLKeyType = "EndpointURL"
)

type endpointURLKeyType string

// WithEndpointURL -
//    Wraps 'parent' in a new Context that has the endpoint URL.
func WithEndpointURL(parent context.Context, u *url.URL) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, endpointURL, u)
}

// EndpointURL -
//   Returns the EndpointURL
func EndpointURL(ctx context.Context) *url.URL {
	if rv, ok := ctx.Value(endpointURL).(*url.URL); ok {
		return rv
	}
	return nil
}
