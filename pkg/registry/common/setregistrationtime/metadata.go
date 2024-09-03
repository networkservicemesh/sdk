// Copyright (c) 2023 Cisco and/or its affiliates.
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

package setregistrationtime

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/networkservicemesh/sdk/pkg/registry/utils/metadata"
)

type key struct{}

// store sets the initialRegistrationTime stored in per NSE metadata.
func store(ctx context.Context, initialRegistrationTime protoreflect.ProtoMessage) {
	metadata.Map(ctx, false).Store(key{}, proto.Clone(initialRegistrationTime))
}

// load returns the initialRegistrationTime stored in per NSE metadata,.
func load(ctx context.Context) (value *timestamppb.Timestamp, ok bool) {
	rawValue, ok := metadata.Map(ctx, false).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(*timestamppb.Timestamp)
	return value, ok
}

// deleteTime deletes the initialRegistrationTime stored in per NSE metadata,.
func deleteTime(ctx context.Context) {
	metadata.Map(ctx, false).Delete(key{})
}
