#!/bin/bash

# Copyright (c) 2020 Doc.ai and/or its affiliates.
#
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PRG_NAME="$0"

function usage() {
  echo "${PRG_NAME} -in=package/file.template.go -out=file.gen.go gen \"KeyType=string,int ValueType=string,int\""
}

######################
# Parse args
######################

template=
out=
gen=

while [[ $# -ne 0 ]]; do
  if [[ "$1" == -in=* ]]; then
    template="${1#"-in="}"
    shift
  elif [[ "$1" == -out=* ]]; then
    out="${1#"-out="}"
    shift
  elif [[ "$1" == "gen" ]]; then
    gen="$2"
    shift 2
    if [[ $# -ne 0 ]]; then
      usage && exit 1
    fi
  else
    usage && exit 1
  fi
done

######################
# Find template file
######################

package="$(echo "${template}" | sed -E "s/(.*)\/.*/\1/g")"
root="$(go list -f "{{.Root}}" "${package}")"

module="$(go list -f "{{.Module}}" "${package}" | sed -E "s/([.]*) .*/\1/g")"
path="${template#${module}}"

######################
# Call genny
######################

genny "-in=${root}/${path}" "-out=${out}" "-pkg=${GOPACKAGE}" gen "${gen}"
