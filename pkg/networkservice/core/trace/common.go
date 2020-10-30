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

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/tracehelper"
	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"
)

func logRequest(span spanhelper.SpanHelper, request proto.Message) {
	connInfo, ok := tracehelper.FromContext(span.Context())
	if ok && !proto.Equal(connInfo.Request, request) {
		requestDiff, hadChanges := diff(connInfo.Request, request)
		if hadChanges {
			span.LogObject("request", requestDiff.Interface())
		}
		connInfo.Request = proto.Clone(request)
	}
}

func logResponse(span spanhelper.SpanHelper, response proto.Message) {
	connInfo, ok := tracehelper.FromContext(span.Context())
	if ok && !proto.Equal(connInfo.Response, response) {
		responseDiff, changed := diff(connInfo.Response, response)
		if changed {
			span.LogObject("response", responseDiff.Interface())
		}
		connInfo.Response = proto.Clone(response)
		return
	}
}

func diff(oldMessage, newMessage proto.Message) (protoreflect.Message, bool) {
	if oldMessage == nil || reflect.ValueOf(oldMessage).IsNil() {
		return newMessage.ProtoReflect(), true
	}

	diffMessage := oldMessage.ProtoReflect().New()
	oldReflectMessage := oldMessage.ProtoReflect()

	// Marker we had any changes
	fieldChanged := false

	newMessage.ProtoReflect().Range(func(descriptor protoreflect.FieldDescriptor, newValue protoreflect.Value) bool {
		oldValue := oldReflectMessage.Get(descriptor)
		if descriptor.Kind() == protoreflect.MessageKind {
			// A pointer to message, we do not need to compare
			if fieldMessage, ok := newValue.Interface().(protoreflect.Message); ok {
				fieldDiff, childFieldChanged := diff(
					oldValue.Interface().(protoreflect.Message).Interface(),
					fieldMessage.Interface(),
				)
				newValue = protoreflect.ValueOf(fieldDiff)
				if childFieldChanged {
					diffMessage.Set(descriptor, newValue)
					fieldChanged = true
				}
			}
		}
		if !reflect.DeepEqual(oldValue, newValue) {
			// Primitive value is not equal, set new value
			diffMessage.Set(descriptor, newValue)
		}
		return true
	})

	return diffMessage, fieldChanged
}
