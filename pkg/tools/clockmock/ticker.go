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

package clockmock

import (
	"time"

	libclock "github.com/benbjohnson/clock"
)

type mockTicker struct {
	mock   *Mock
	ticker *libclock.Ticker
}

func (t *mockTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t *mockTicker) Stop() {
	t.mock.lock.RLock()
	defer t.mock.lock.RUnlock()

	t.ticker.Stop()
}

func (t *mockTicker) Reset(d time.Duration) {
	t.mock.lock.RLock()
	defer t.mock.lock.RUnlock()

	t.ticker.Reset(d)
}
