// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package clientinfo provides a set of utilities for adding client info to labels map
package clientinfo

import (
	"context"
	"os"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	nodeNameEnv      = "NODE_NAME"
	podNameEnv       = "POD_NAME"
	clusterNameEnv   = "CLUSTER_NAME"
	nodeNameLabel    = "nodeName"
	podNameLabel     = "podName"
	clusterNameLabel = "clusterName"
)

// AddClientInfo adds client info (node/pod/cluster names) to provided map, taking this info from corresponding
// environment variables
func AddClientInfo(ctx context.Context, labels map[string]string) {
	names := map[string]string{
		nodeNameEnv:    nodeNameLabel,
		podNameEnv:     podNameLabel,
		clusterNameEnv: clusterNameLabel,
	}
	for envName, labelName := range names {
		value, exists := os.LookupEnv(envName)
		if !exists {
			log.FromContext(ctx).Warnf("Environment variable %s is not set. Skipping.", envName)
			continue
		}
		oldValue, isPresent := labels[labelName]
		if isPresent {
			log.FromContext(ctx).Warnf("The label %s was already assigned to %s. Skipping.", labelName, oldValue)
			continue
		}
		labels[labelName] = value
	}
}
