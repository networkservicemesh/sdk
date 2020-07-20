// Copyright (c) 2019-2020 VMware, Inc.
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

package roundrobin

import (
	"reflect"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"go.uber.org/goleak"
)

type args struct {
	ns                      *registry.NetworkService
	networkServiceEndpoints []*registry.NetworkServiceEndpoint
}

type rrtests struct {
	name string
	args args
	want *registry.NetworkServiceEndpoint
}

var tests = []rrtests{
	{
		name: "network-service-1 first pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-1",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-1",
				},
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-3",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-1",
		},
	},
	{
		name: "network-service-1 second pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-1",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-1",
				},
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-3",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-2",
		},
	},
	{
		name: "network-service-1 remove NSE-1 first pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-1",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-3",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-2",
		},
	},
	{
		name: "network-service-1 remove NSE-1 second pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-1",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-3",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-3",
		},
	},
	{
		name: "network-service-2 first pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-2",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-3",
				},
				{
					Name: "NSE-1",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-2",
		},
	},
	{
		name: "network-service-2 second pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-2",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-3",
				},
				{
					Name: "NSE-1",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-3",
		},
	},
	{
		name: "network-service-2 remove NSE-3 first pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-2",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-1",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-2",
		},
	},
	{
		name: "network-service-2 remove NSE-3 second pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-2",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-1",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-1",
		},
	},
	{
		name: "network-service-1 third pass",
		args: args{
			ns: &registry.NetworkService{
				Name: "network-service-1",
			},
			networkServiceEndpoints: []*registry.NetworkServiceEndpoint{
				{
					Name: "NSE-1",
				},
				{
					Name: "NSE-2",
				},
				{
					Name: "NSE-3",
				},
			},
		},
		want: &registry.NetworkServiceEndpoint{
			Name: "NSE-2",
		},
	},
}

func Test_roundRobinSelector_SelectEndpoint(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	rr := newRoundRobinSelector()
	for _, tt := range tests {
		if got := rr.selectEndpoint(tt.args.ns, tt.args.networkServiceEndpoints); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: roundRobinSelector.selectEndpoint() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
