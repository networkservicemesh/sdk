// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2025 OpenInfra Foundation Europe. All rights reserved.
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

package traceconcise

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/stringutils"
)

const (
	methodNameRequest = "Request"
	methodNameClose   = "Close"
	closePrefix       = "close"
	requestPrefix     = "request"
)

func logRequest(ctx context.Context, request any, prefixes ...string) {
	msg := strings.Join(prefixes, "-")
	logObject(ctx, msg, request)
}

func logResponse(ctx context.Context, response any, prefixes ...string) {
	msg := strings.Join(append(prefixes, "response"), "-")
	logObject(ctx, msg, response)
}

func logError(ctx context.Context, err error, operation string) error {
	log.FromContext(ctx).Errorf("%v", errors.Wrapf(err, "Error returned from %s", operation))
	return err
}

func logObject(ctx context.Context, k, v interface{}) {
	if !trace(ctx) {
		return
	}
	s := log.FromContext(ctx)
	msg := ""
	cc, err := json.Marshal(v)
	if err == nil {
		msg = stringutils.ConvertBytesToString(cc)
	} else {
		msg = fmt.Sprint(v)
	}
	s.Infof("%v=%s", k, msg)
}
