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
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/tracehelper"
	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"
)

func logRequest(span spanhelper.SpanHelper, request *networkservice.NetworkServiceRequest) {
	connInfo, ok := tracehelper.FromContext(span.Context())
	if ok && !proto.Equal(request, connInfo.Request) {
		requestDiff := (&networkservice.NetworkServiceRequest{}).ProtoReflect()
		span.LogObject("request diff", diff(connInfo.Request, request, requestDiff).Interface())
		connInfo.Request = request.Clone()
	}
}

func logResponse(span spanhelper.SpanHelper, response *networkservice.Connection) {
	connInfo, ok := tracehelper.FromContext(span.Context())
	if ok && !proto.Equal(response, connInfo.Response) {
		responseDiff := (&networkservice.Connection{}).ProtoReflect()
		span.LogObject("response diff", diff(connInfo.Response, response, responseDiff).Interface())
		connInfo.Response = response.Clone()
		return
	}
	span.LogValue("response", "")
}

func diff(oldMessage, newMessage proto.Message, diffStruct protoreflect.Message) protoreflect.Message {
	if oldMessage == nil || reflect.ValueOf(oldMessage).IsNil() {
		return newMessage.ProtoReflect()
	}

	oldReflectMessage := oldMessage.ProtoReflect()
	newMessage.ProtoReflect().Range(func(descriptor protoreflect.FieldDescriptor, newValue protoreflect.Value) bool {
		oldValue := oldReflectMessage.Get(descriptor)
		if !reflect.DeepEqual(oldValue, newValue) {
			diffStruct.Set(descriptor, newValue)
		}
		return true
	})

	return diffStruct
}
