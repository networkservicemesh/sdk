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

import input.attributes.request.http as http_request

default service_connection = false

service_connection {
	input.ID == svc_spiffe_id
}

svc_spiffe_id = spiffe_id {
   [_, _, uri_type_san] := split(http_request.headers["x-forwarded-client-cert"], ";")
   [_, spiffe_id] := split(uri_type_san, "=")
}