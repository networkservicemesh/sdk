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

// Package clock provides tools for accessing time functions
package clock

import (
	"context"
	"time"
)

// Clock is an interface for accessing time functions
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration

	Sleep(d time.Duration)

	Timer(d time.Duration) Timer
	After(d time.Duration) <-chan time.Time
	AfterFunc(d time.Duration, f func()) Timer

	Ticker(d time.Duration) Ticker

	WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc)
	WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc)
}

type clockImpl struct{}

func (c *clockImpl) Now() time.Time {
	return time.Now()
}

func (c *clockImpl) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (c *clockImpl) Until(t time.Time) time.Duration {
	return time.Until(t)
}

func (c *clockImpl) Sleep(d time.Duration) {
	time.Sleep(d)
}

func (c *clockImpl) Timer(d time.Duration) Timer {
	return &realTimer{
		Timer: time.NewTimer(d),
	}
}

func (c *clockImpl) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c *clockImpl) AfterFunc(d time.Duration, f func()) Timer {
	return &realTimer{
		Timer: time.AfterFunc(d, f),
	}
}

func (c *clockImpl) Ticker(d time.Duration) Ticker {
	return &realTicker{
		Ticker: time.NewTicker(d),
	}
}

func (c *clockImpl) WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(parent, deadline)
}

func (c *clockImpl) WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}
