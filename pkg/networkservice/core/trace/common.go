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
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/networkservicemesh/sdk/pkg/tools/spanhelper"
)

func logRequest(span spanhelper.SpanHelper, request proto.Message) {
	connInfo, ok := trace(span.Context())
	if ok && !proto.Equal(connInfo.Request, request) {
		if connInfo.Request != nil {
			requestDiff, hadChanges := Diff(connInfo.Request.ProtoReflect(), request.ProtoReflect())
			if hadChanges {
				span.LogObject("request-diff", requestDiff)
			}
		} else {
			span.LogObject("request", request)
		}
		connInfo.Request = proto.Clone(request)
	}
}

func logResponse(span spanhelper.SpanHelper, response proto.Message) {
	connInfo, ok := trace(span.Context())
	if ok && !proto.Equal(connInfo.Response, response) {
		if connInfo.Response != nil {
			responseDiff, changed := Diff(connInfo.Response.ProtoReflect(), response.ProtoReflect())
			if changed {
				span.LogObject("response-diff", responseDiff)
			}
		} else {
			span.LogObject("response", response)
		}
		connInfo.Response = proto.Clone(response)
		return
	}
}

// Diff - calculate a protobuf messge diff
func Diff(oldMessage, newMessage protoreflect.Message) (map[string]interface{}, bool) {
	if oldMessage == nil || reflect.ValueOf(oldMessage).IsNil() {
		return nil, true
	}

	diffMessage := map[string]interface{}{}
	oldReflectMessage := oldMessage

	// Marker we had any changes
	changes := 0

	newMessage.Range(func(descriptor protoreflect.FieldDescriptor, newRefValue protoreflect.Value) bool {
		rawOldValue := oldReflectMessage.Get(descriptor)
		oldValue := rawOldValue.Interface()
		newValue := newRefValue.Interface()

		if descriptor.Cardinality() == protoreflect.Repeated {
			if newList, ok := newValue.(protoreflect.List); ok {
				oldList := oldValue.(protoreflect.List)

				originMap := map[string]protoreflect.Value{}
				for i := 0; i < oldList.Len(); i++ {
					originMap[fmt.Sprintf("%d", i)] = oldList.Get(i)
				}
				targetMap := map[string]protoreflect.Value{}
				for i := 0; i < newList.Len(); i++ {
					targetMap[fmt.Sprintf("%d", i)] = newList.Get(i)
				}
				if resultMap, mapChanged := mapDiff(descriptor, originMap, targetMap); mapChanged {
					changes++
					diffMessage[string(descriptor.Name())] = resultMap
				}
			}
			if newMap, ok := newValue.(protoreflect.Map); ok {
				oldMap := oldValue.(protoreflect.Map)

				originMap := map[string]protoreflect.Value{}
				targetMap := map[string]protoreflect.Value{}

				oldMap.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
					originMap[key.String()] = value
					return true
				})
				newMap.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
					targetMap[key.String()] = value
					return true
				})

				if resultMap, mapChanged := mapDiff(descriptor, originMap, targetMap); mapChanged {
					changes++
					diffMessage[string(descriptor.Name())] = resultMap
				}
			}
			return true
		}
		val, diff := diffField(descriptor, newValue, oldValue)
		if diff {
			changes++
			diffMessage[string(descriptor.Name())] = val
		}
		return true
	})

	return diffMessage, changes > 0
}

func mapDiff(descriptor protoreflect.FieldDescriptor, originMap, targetMap map[string]protoreflect.Value) (interface{}, bool) {
	resultMap := map[string]interface{}{}
	lchanged := 0
	for key, value := range targetMap {
		oldVal, ok := originMap[key]
		if !ok {
			// No old value,
			resultMap["+"+key] = value.String()
			continue
		}
		val, diff := diffField(descriptor, oldVal.Interface(), value.Interface())
		if diff {
			if diff {
				lchanged++
				resultMap[key] = val
			}
		}
	}
	for key, value := range originMap {
		_, ok := targetMap[key]
		if !ok {
			// No new value, mark as deleted
			resultMap["-"+key] = value.String()
		}
	}
	return resultMap, lchanged > 0
}

func diffField(descriptor protoreflect.FieldDescriptor, oldValue, newValue interface{}) (interface{}, bool) {
	if descriptor.Kind() == protoreflect.MessageKind {
		// A pointer to message, we do not need to compare
		if newMsg, ok := newValue.(protoreflect.Message); ok {
			oldMsg := oldValue.(protoreflect.Message)
			fieldDiff, childFieldChanged := Diff(oldMsg, newMsg)
			if childFieldChanged {
				return fieldDiff, true
			}
			return "=", false
		}
	}
	if !reflect.DeepEqual(oldValue, newValue) {
		// Primitive value is not equal, set new value
		return newValue, true
	}
	return nil, false
}
