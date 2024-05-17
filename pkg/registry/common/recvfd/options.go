// Copyright (c) 2024 Cisco and/or its affiliates.
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

package recvfd

import "context"

type options struct {
	chainContext context.Context
}

// Option applies additional configuration for the recvfd chain element
type Option func(*options)

// WithChainContext sets chain context for the recvfd chain element that can be used to prevent goroutines leaks
func WithChainContext(ctx context.Context) Option {
	return func(rn *options) {
		rn.chainContext = ctx
	}
}
