// Copyright (c) 2021-2024 Doc.ai and/or its affiliates.
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

package mechanisms

import (
	"fmt"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
)

var (
	errCannotSupportMech = errors.New("cannot support any of the requested mechanism")
	errUnsupportedMech   = errors.New("unsupported mechanism")
)

const (
	clientMetricKey = "client_interface"
	serverMetricKey = "server_interface"
	nameKey         = "name"
	unresolvedName  = "unknown"
)

type interfacesInfo struct {
	interfaceName string
	interfaceType string
}

func (i *interfacesInfo) getInterfaceDetails() string {
	return fmt.Sprintf("%s/%s", i.interfaceType, i.interfaceName)
}

// Save interface details in Path.
func storeMetrics(conn *networkservice.Connection, mechanism *networkservice.Mechanism, isClient bool) {
	path := conn.GetPath()
	if path == nil {
		return
	}

	segments := path.GetPathSegments()
	if segments == nil {
		return
	}

	segment := segments[path.GetIndex()]
	params := mechanism.GetParameters()

	name := unresolvedName
	if params != nil {
		name = params[nameKey]
	}

	info := &interfacesInfo{
		interfaceName: name,
		interfaceType: mechanism.GetType(),
	}

	if segment.Metrics == nil {
		segment.Metrics = make(map[string]string)
	}

	metricKey := clientMetricKey
	if !isClient {
		metricKey = serverMetricKey
	}

	if _, ok := segment.GetMetrics()[metricKey]; !ok {
		segment.Metrics[metricKey] = info.getInterfaceDetails()
	}
}
