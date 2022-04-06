// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

package heal

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// LivenessChecker blocks until the connection is alive or ctx is done.
type LivenessChecker func(ctx context.Context, conn *networkservice.Connection)

type options struct {
	livenessChecker LivenessChecker
	attemptAfter    time.Duration
}

// Option - option for heal.NewClient() chain element
type Option func(o *options)

// WithLivenessChecker - sets the liveness checker for the heal chain element
func WithLivenessChecker(livenessChecker LivenessChecker) Option {
	return func(o *options) {
		o.livenessChecker = livenessChecker
	}
}

// WithAttemptAfter sets the time interval to wait before healing failed connection
func WithAttemptAfter(attemptAfter time.Duration) Option {
	return func(o *options) {
		o.attemptAfter = attemptAfter
	}
}
