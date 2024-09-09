// Copyright (c) 2022-2023 Cisco Systems, Inc.
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

package trace

import (
	"context"
	"strconv"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/typeutils"
)

type beginTraceHandler struct {
	traced dnsutils.Handler
}

type endTraceHandler struct{}

// NewDNSHandler - wraps tracing around the supplied traced.
func NewDNSHandler(traced dnsutils.Handler) dnsutils.Handler {
	return next.NewDNSHandler(
		&beginTraceHandler{traced: traced},
		&endTraceHandler{},
	)
}

func (t *beginTraceHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	methodName := "ServeDNS"
	operation := typeutils.GetFuncName(t.traced, methodName)
	ctx, finish := withLog(ctx, operation, methodName, strconv.Itoa(int(m.Id)))
	defer finish()

	logRequest(ctx, m, "message")

	wrapper := wrapResponseWriter(rw)
	t.traced.ServeDNS(ctx, wrapper, m)

	logResponse(ctx, &wrapper.responseMsg, "message")
}

func (t *endTraceHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	logRequest(ctx, m, "message")

	wrapper := wrapResponseWriter(rw)
	next.Handler(ctx).ServeDNS(ctx, wrapper, m)

	logResponse(ctx, &wrapper.responseMsg, "message")
}

func wrapResponseWriter(rw dns.ResponseWriter) *traceResponseWriter {
	wrapper, ok := rw.(*traceResponseWriter)
	if !ok {
		wrapper = &traceResponseWriter{
			ResponseWriter: rw,
		}
	}

	return wrapper
}

type traceResponseWriter struct {
	dns.ResponseWriter
	responseMsg dns.Msg
}

func (rw *traceResponseWriter) WriteMsg(m *dns.Msg) error {
	rw.responseMsg = *m.Copy()
	return rw.ResponseWriter.WriteMsg(m)
}
