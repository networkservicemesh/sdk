// Copyright (c) 2022-2024 Cisco Systems, Inc.
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
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/r3labs/diff"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func logRequest(ctx context.Context, message *dns.Msg, prefixes ...string) {
	msg := strings.Join(append(prefixes, "request"), "-")
	diffMsg := strings.Join(append(prefixes, "request", "diff"), "-")

	messageInfo, ok := trace(ctx)
	if ok && !cmp.Equal(messageInfo.RequestMsg, message) {
		if messageInfo.RequestMsg != nil {
			messageDiff, _ := diff.Diff(messageInfo.RequestMsg, message)
			if len(messageDiff) > 0 {
				logObjectTrace(ctx, diffMsg, messageDiff)
			}
		} else {
			logObjectTrace(ctx, msg, message)
		}
		messageInfo.RequestMsg = message.Copy()
		return
	}
}

func logResponse(ctx context.Context, message *dns.Msg, prefixes ...string) {
	msg := strings.Join(append(prefixes, "response"), "-")
	diffMsg := strings.Join(append(prefixes, "response", "diff"), "-")

	messageInfo, ok := trace(ctx)
	if ok && !cmp.Equal(messageInfo.ResponseMsg, message) {
		if messageInfo.ResponseMsg != nil {
			messageDiff, _ := diff.Diff(messageInfo.ResponseMsg, message)
			if len(messageDiff) > 0 {
				logObjectTrace(ctx, diffMsg, messageDiff)
			}
		} else {
			logObjectTrace(ctx, msg, message)
		}
		messageInfo.ResponseMsg = message.Copy()
		return
	}
}

func logObjectTrace(ctx context.Context, k, v interface{}) {
	log.FromContext(ctx).Tracef("%v=%s", k, v)
}
