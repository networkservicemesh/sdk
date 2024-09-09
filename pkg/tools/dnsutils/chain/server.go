// Copyright (c) 2022 Cisco Systems, Inc.
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

// Package chain provides a simple file for creating a dnsutils.Handler from a 'chain' of dnsutils.Handler
package chain

import (
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/trace"
)

// NewDNSHandler - chains together a list of dnsutils.Handler with tracing.
func NewDNSHandler(handlers ...dnsutils.Handler) dnsutils.Handler {
	return next.NewDNSHandler(
		next.NewWrappedDNSHandler(trace.NewDNSHandler, handlers...),
	)
}
