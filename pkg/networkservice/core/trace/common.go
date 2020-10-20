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

package trace

import (
	"github.com/google/go-cmp/cmp"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/tracehelper"
	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"
)

func logRequest(span spanhelper.SpanHelper, request *networkservice.NetworkServiceRequest) {
	connInfo, ok := tracehelper.FromContext(span.Context())
	if ok && !cmp.Equal(request.String(), connInfo.Request) {
		//fmt.Printf("diff: %v", cmp.Diff(request.String(), connInfo.Request))
		span.LogObject("request updated", request)
		connInfo.Request = request.String()
	}
}

func logResponse(span spanhelper.SpanHelper, response *networkservice.Connection) {
	connInfo, ok := tracehelper.FromContext(span.Context())
	if ok && !cmp.Equal(response.String(), connInfo.Response) {
		//fmt.Printf("diff: %v", cmp.Diff(response.String(), connInfo.Response))
		span.LogObject("response updated", response)
		connInfo.Response = response.String()
		return
	}
	span.LogValue("response", "")
}
