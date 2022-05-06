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
	spireServerConfFileName = "server/server.conf"
	spireServerConfContents = `server {
    bind_address = "127.0.0.1"
    bind_port = "8081"
    trust_domain = "example.org"
    data_dir = "%[1]s/data"
    log_level = "WARN"
    ca_key_type = "rsa-2048"
    default_svid_ttl = "1h"
    ca_subject = {
        country = ["US"],
        organization = ["SPIFFE"],
        common_name = "",
    }
}

plugins {
    DataStore "sql" {
        plugin_data {
            database_type = "sqlite3"
            connection_string = "%[1]s/data/datastore.sqlite3"
        }
    }

    NodeAttestor "join_token" {
        plugin_data {
        }
    }

    KeyManager "memory" {
        plugin_data = {}
    }
}
`
)
