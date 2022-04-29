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

const (
	livenessCheckInterval = 200 * time.Millisecond
	livenessCheckTimeout  = 100 * time.Millisecond
)

// LivenessChecker - function that returns true of conn is 'live' and false otherwise
type LivenessChecker func(deadlineCtx context.Context, conn *networkservice.Connection) bool

type options struct {
	dataPlaneLivenessChecker LivenessChecker
	livenessCheckInterval    time.Duration
	livenessCheckTimeout     time.Duration
}

// Option - option for heal.NewClient() chain element
type Option func(o *options)

// WithLivenessChecker - sets the data plane liveness checker
func WithLivenessChecker(livenessChecker LivenessChecker) Option {
	return func(o *options) {
		o.dataPlaneLivenessChecker = livenessChecker
	}
}

// WithLivenessCheckInterval - sets livenessCheckInterval
func WithLivenessCheckInterval(livenessCheckInterval time.Duration) Option {
	return func(o *options) {
		o.livenessCheckInterval = livenessCheckInterval
	}
}

// WithLivenessCheckTimeout - sets livenessCheckTimeout
func WithLivenessCheckTimeout(livenessCheckTimeout time.Duration) Option {
	return func(o *options) {
		o.livenessCheckTimeout = livenessCheckTimeout
	}
}
