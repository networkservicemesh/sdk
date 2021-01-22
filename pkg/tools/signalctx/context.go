// Copyright (c) 2020 Cisco and/or its affiliates.
//
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

// Package signalctx provides a context which is canceled when an os.Signal is received
package signalctx

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/networkservicemesh/sdk/pkg/tools/logger/logruslogger"
)

// WithSignals - returns a context that is canceled when on of the specified signals 'sig' is received.
//               if sig is not provided, defaults to listening for signals:
//               os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT
func WithSignals(parent context.Context, sig ...os.Signal) context.Context {
	// If no signals are specified, go with the defaults
	if len(sig) == 0 {
		sig = []os.Signal{
			os.Interrupt,
			// More Linux signals here
			syscall.SIGHUP,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		}
	}
	// Catch the signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig...)
	// Prepare a master context for the command, and be prepared to cancel it when we get a signal
	ctx, cancel := context.WithCancel(parent)
	_, log := logruslogger.New(ctx)
	go func() {
		select {
		case s := <-c:
			log.Warnf("Caught signal %s, exiting...", s)
			cancel()
		case <-parent.Done():
		}
	}()
	return ctx
}
