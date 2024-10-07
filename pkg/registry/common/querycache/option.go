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

package querycache

import "time"

// NSCacheOption is an option for NS cache
type NSCacheOption func(c *nsCache)

// NSECacheOption is an option for NSE cache
type NSECacheOption func(c *nseCache)

// WithNSExpireTimeout sets NS cache expire timeout
func WithNSExpireTimeout(expireTimeout time.Duration) NSCacheOption {
	return func(c *nsCache) {
		c.expireTimeout = expireTimeout
	}
}

// WithNSEExpireTimeout sets NSE cache expire timeout
func WithNSEExpireTimeout(expireTimeout time.Duration) NSECacheOption {
	return func(c *nseCache) {
		c.expireTimeout = expireTimeout
	}
}
