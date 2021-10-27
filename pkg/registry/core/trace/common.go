// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

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
