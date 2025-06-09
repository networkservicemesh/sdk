# Copyright (c) 2020-2023 Cisco and/or its affiliates.
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

default valid := false
default index := 0

index = input.index

valid {
	right_side_segments := array.slice(input.path_segments, index, count(input.path_segments))
	count({x | right_side_segments[x]; token_alive(right_side_segments[x].token)}) == count(right_side_segments)
}

# alive means not expired
token_alive(token) {
	[_, payload, _] := io.jwt.decode(token)
	now < payload.exp
}

now := t {
	ns := time.now_ns()
	t := ns / 1e9
}
