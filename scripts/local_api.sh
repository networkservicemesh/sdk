#!/bin/bash

# Original script by Andy Bursavich:
# https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-521493597

DIR=$( dirname "${BASH_SOURCE[0]}" )/../
cd "${DIR}"

set -euo pipefail

MODS=()
while IFS='' read -r line
do
    MODS+=("$line")
done < <( grep "github.com/networkservicemesh/api" go.mod  | sed 's/^replace //' | awk '{print $1}' | sort -u)


for MOD in "${MODS[@]}"; do
  go mod edit -replace="${MOD}"="${MOD/github.com\/networkservicemesh\/api/../api}"
done
go mod tidy
