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

package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type DiffReporter struct {
	path  cmp.Path
	diffs []string
}

func (r *DiffReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

func (r *DiffReporter) Report(rs cmp.Result) {
	if !rs.Equal() {
		vx, vy := r.path.Last().Values()
		r.diffs = append(r.diffs, fmt.Sprintf("%#v:\n\t-: %+v\n\t+: %+v\n", r.path, vx, vy))
	}
}

func (r *DiffReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

func (r *DiffReporter) String() string {
	return strings.Join(r.diffs, "\n")
}

func logMessage(ctx context.Context, message *dns.Msg, prefixes ...string) {
	msg := strings.Join(append(prefixes, "response"), "-")
	diffMsg := strings.Join(append(prefixes, "response", "diff"), "-")

	messageInfo, ok := trace(ctx)
	if ok && !cmp.Equal(messageInfo.Message, message) {
		if messageInfo.Message != nil {
			messageDiff := cmp.Diff(messageInfo.Message, message, cmpopts.EquateEmpty())
			if messageDiff != "" {
				logObjectTrace(ctx, diffMsg, messageDiff)
			}
		} else {
			logObjectTrace(ctx, msg, message)
		}
		messageInfo.Message = message.Copy()
		return
	}
}

func logObjectTrace(ctx context.Context, k, v interface{}) {
	s := log.FromContext(ctx)
	msg := ""
	cc, err := json.Marshal(v)
	if err == nil {
		msg = string(cc)
	} else {
		msg = fmt.Sprint(v)
	}
	s.Tracef("%v=%s", k, msg)
}
