// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package forwarder provides common chain elements for the interpose NSEs (a.k.a forwarders)
package forwarder

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect/translation"
)

// NewTranslationClient returns a new translation client for the interpose NSE
func NewTranslationClient() networkservice.NetworkServiceClient {
	return new(translation.Builder).
		WithRequestOptions(
			translation.ReplaceMechanism(),
			translation.ReplaceMechanismPreferences(),
		).
		WithConnectionOptions(
			translation.WithContext(),
			translation.WithPathSegments(),
		).
		Build()
}
