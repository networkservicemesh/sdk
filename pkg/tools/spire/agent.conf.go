// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

package spire

const (
	/* See default socket_path value: https://github.com/spiffe/spire/blob/v1.2.3/doc/spire_agent.md */
	spireDefaultEndpointSocket = "/tmp/spire-agent/public/api.sock"

	spireAgentConfFilename = "agent/agent.conf"
	spireAgentConfContents = `agent {
    data_dir = "%[1]s/data"
    log_level = "WARN"
    server_address = "127.0.0.1"
    server_port = "8081"
    insecure_bootstrap = true
    trust_domain = "example.org"
}

plugins {
    NodeAttestor "join_token" {
        plugin_data {
        }
    }
    KeyManager "disk" {
        plugin_data {
            directory = "%[1]s/data"
        }
    }
    WorkloadAttestor "unix" {
        plugin_data {
            discover_workload_path = true
        }
    }
}
`
)
