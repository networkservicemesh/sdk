// Copyright (c) 2021-2023 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2024 Cisco Systems, Inc.
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

package traceverbose

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/OneOfOne/xxhash"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/stringutils"
)

const (
	methodNameRequest = "Request"
	methodNameClose   = "Close"
)

func logRequest(ctx context.Context, request proto.Message, prefixes ...string) {
	connInfo, _ := trace(ctx)

	if v := hashCode(request); connInfo.Request != v {
		if connInfo.Request != 0 {
			logObjectTrace(ctx, strings.Join(append(prefixes, "diff"), "-"), request)
		} else {
			logObjectTrace(ctx, strings.Join(prefixes, "-"), request)
		}
		connInfo.Request = v
	}
}

func logResponse(ctx context.Context, response proto.Message, prefixes ...string) {
	connInfo, _ := trace(ctx)
	if v := hashCode(response); connInfo.Response != v {
		if connInfo.Response != 0 {
			logObjectTrace(ctx, strings.Join(append(prefixes, "response", "diff"), "-"), response)
		} else {
			logObjectTrace(ctx, strings.Join(append(prefixes, "response"), "-"), response)
		}
		connInfo.Response = v
	}
}

// Diff - calculate a protobuf message diff
func Diff(oldMessage, newMessage protoreflect.Message) (map[string]interface{}, bool) {
	diffMessage := map[string]interface{}{}
	oldReflectMessage := oldMessage

	// Marker we had any changes
	changes := 0
	fields := newMessage.Descriptor().Fields()
	for fi := 0; fi < fields.Len(); fi++ {
		descriptor := fields.Get(fi)
		newRefValue := newMessage.Get(descriptor)
		rawOldValue := oldReflectMessage.Get(descriptor)
		oldValue := rawOldValue.Interface()
		newValue := newRefValue.Interface()

		if descriptor.Cardinality() == protoreflect.Repeated {
			originMap := map[string]protoreflect.Value{}
			targetMap := map[string]protoreflect.Value{}

			switch val := newValue.(type) {
			case protoreflect.List:
				// Convert list to MAP with indexes
				oldList := oldValue.(protoreflect.List)

				for i := 0; i < oldList.Len(); i++ {
					originMap[strconv.Itoa(i)] = oldList.Get(i)
				}
				for i := 0; i < val.Len(); i++ {
					targetMap[strconv.Itoa(i)] = val.Get(i)
				}
			case protoreflect.Map:
				oldMap := oldValue.(protoreflect.Map)

				oldMap.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
					originMap[key.String()] = value
					return true
				})
				val.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
					targetMap[key.String()] = value
					return true
				})
			}
			if resultMap, mapChanged := mapDiff(descriptor, originMap, targetMap); mapChanged {
				changes++
				diffMessage[string(descriptor.Name())] = resultMap
			}
			continue
		}
		val, diff := diffField(descriptor, oldValue, newValue)
		if diff {
			changes++
			diffMessage[string(descriptor.Name())] = val
		}
	}

	return diffMessage, changes > 0
}

func mapDiff(descriptor protoreflect.FieldDescriptor, originMap, targetMap map[string]protoreflect.Value) (interface{}, bool) {
	resultMap := map[string]interface{}{}
	lchanged := 0
	for key, value := range targetMap {
		oldVal, ok := originMap[key]
		if !ok {
			// No old value,
			putToMapDiff(value, resultMap, "+"+key)
			lchanged++
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
			putToMapDiff(value, resultMap, "-"+key)
			lchanged++
		}
	}
	return resultMap, lchanged > 0
}

func putToMapDiff(value protoreflect.Value, resultMap map[string]interface{}, key string) {
	if msg, ok := value.Interface().(protoreflect.Message); ok {
		smsg := msg.Interface()
		resultMap[key] = smsg
	} else {
		resultMap[key] = value.String()
	}
}

func diffField(descriptor protoreflect.FieldDescriptor, oldValue, newValue interface{}) (interface{}, bool) {
	if descriptor.Kind() == protoreflect.MessageKind {
		// A pointer to message, we do not need to compare
		if newMsg, ok := newValue.(protoreflect.Message); ok {
			oldMsg, oldOk := oldValue.(protoreflect.Message)
			if !oldOk {
				// No old message defined
				return newMsg.Interface(), true
			}
			fieldDiff, childFieldChanged := Diff(oldMsg, newMsg)
			if childFieldChanged {
				return fieldDiff, true
			}
			return "=", false
		} else if oldMsg, ok := oldValue.(protoreflect.Message); ok {
			// No new message defined
			// No old message defined
			return oldMsg.Interface(), true
		}
	}
	if !reflect.DeepEqual(oldValue, newValue) {
		// Primitive value is not equal, set new value
		return newValue, true
	}
	return nil, false
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func logError(ctx context.Context, err error, operation string) error {
	if _, ok := err.(stackTracer); !ok {
		if err == error(nil) {
			return nil
		}
		err = errors.Wrapf(err, "Error returned from %s", operation)
		log.FromContext(ctx).Errorf("%+v", err)
		return err
	}
	log.FromContext(ctx).Errorf("%v", err)
	return err
}

func hashCode(msg proto.Message) uint64 {
	bytes, _ := proto.MarshalOptions{Deterministic: true, AllowPartial: true, UseCachedSize: true}.Marshal(msg)
	return xxhash.Checksum64(bytes)
}

func logObjectTrace(ctx context.Context, k, v interface{}) {
	s := log.FromContext(ctx)
	msg := ""
	cc, err := json.Marshal(v)
	if err == nil {
		msg = stringutils.ConvertBytesToString(cc)
	} else {
		msg = fmt.Sprint(v)
	}
	s.Tracef("%v=%s", k, msg)
}
