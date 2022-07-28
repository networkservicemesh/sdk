# Copyright (c) 2020 Cisco and/or its affiliates.
#
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

package nsm

default service_connection = false

service_connection {
	conn_ids := {y | y = input.spiffe_id_connection_map[input.service_spiffe_id][_]}
   path_conn_ids := {x | x = input.selector_connection_ids[_]}
   count(path_conn_ids) > 0
   count(conn_ids) > 0
   inter := conn_ids & path_conn_ids
   count(inter) > 0
}
