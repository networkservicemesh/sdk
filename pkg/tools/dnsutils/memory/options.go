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

package memory

import (
	"net"

	"github.com/edwarnicke/genericsync"
)

// Option replaces default behavior for memory dns chain element
type Option func(*memoryHandler)

// WithIPRecords sets IP responses
func WithIPRecords(r *genericsync.Map[string, []net.IP]) Option {
	return func(mh *memoryHandler) {
		mh.ipRecords = r
	}
}

// WithSRVRecords sets SRV responses
func WithSRVRecords(r *genericsync.Map[string, []*net.TCPAddr]) Option {
	return func(mh *memoryHandler) {
		mh.srvRecords = r
	}
}
