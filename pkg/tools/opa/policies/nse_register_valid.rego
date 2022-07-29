# Copyright (c) 2022 Cisco and/or its affiliates.
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

default nse_register_allowed = false
	
# new NSE case
nse_register_allowed {
        nses := { nse | nse := input.SpiffieIDNSEsMap[_][_]; nse == input.NSEName }
        count(nses) == 0
}
	
# refresh NSE case
nse_register_allowed {
    input.SpiffieIDNSEsMap[input.SpiffieID][_] == input.NSEName
}