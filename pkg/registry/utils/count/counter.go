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

package count

import "sync/atomic"

// CallCounter - stores Register / Unregister / Find calls count.
type CallCounter struct {
	totalRegisterCalls   int32
	totalUnregisterCalls int32
	totalFindCalls       int32
}

// Registers returns Register call count.
func (c *CallCounter) Registers() int {
	return int(atomic.LoadInt32(&c.totalRegisterCalls))
}

// Unregisters returns Unregister count.
func (c *CallCounter) Unregisters() int {
	return int(atomic.LoadInt32(&c.totalUnregisterCalls))
}

// Finds returns Find count.
func (c *CallCounter) Finds() int {
	return int(atomic.LoadInt32(&c.totalFindCalls))
}
