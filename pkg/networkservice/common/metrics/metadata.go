// Copyright (c) 2022 Cisco and/or its affiliates.
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

package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric/instrument/syncint64"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type keyType struct{}
type metricsMap = map[string]syncint64.Histogram

func loadOrStore(ctx context.Context, metrics metricsMap) (value metricsMap, ok bool) {
	rawValue, ok := metadata.Map(ctx, false).LoadOrStore(keyType{}, metrics)
	return rawValue.(metricsMap), ok
}
