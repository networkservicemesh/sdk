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

default tokens_valid = false

tokens_valid {
	count(input.path_segments) == 0
}

tokens_valid {
	c := count({x | input.path_segments[x]; token_valid(input.path_segments[x].token)})
	c == count(input.path_segments)
}

token_valid(token) = r {
    [_, _, _] := io.jwt.decode(token)
	r := true
}