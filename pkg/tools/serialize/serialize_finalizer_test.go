// Copyright (c) 2020 Cisco Systems, Inc.
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

package serialize

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFinalizer is intentionally the only test in the serialize package proper precisely because
// testing the finalizer requires being able to abuse the knowledge of the internals.
func TestFinalizer(t *testing.T) {
	// Create a serialize Executor
	exec := (NewExecutor()).(*executor)
	// The executors internal finalizeCh in a separate variable, so that when we take exec out of
	// scope we can still access it.  Note: This shouldn't preclude garbage collection of exec
	finalizedCh := exec.finalizedCh
	// Take exec out of scope
	exec = nil
	// Run the garbage collector
	runtime.GC()
	select {
		case <-finalizedCh:
			return
		case <- time.After(30*time.Second):
			assert.Fail(t,"Timeout waiting for finalizer to run")
	}
}
