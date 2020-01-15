#!/bin/bash

# Original script by Andy Bursavich:
# https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-521493597

DIR=$( dirname "${BASH_SOURCE[0]}" )/../
cd "${DIR}"

set -euo pipefail

BRANCH=master

V=$(
        go mod download -json "github.com/networkservicemesh/networkservicemesh/controlplane/api@${BRANCH}" |
        sed -n 's|.*"Version": "\(.*\)".*|\1|p'
    )

MODS=()
while IFS='' read -r line
do
    MODS+=("$line")
done < <( grep "github.com/networkservicemesh/networkservicemesh" go.mod  | sed 's/^replace //' | awk '{print $1}' | sort -u)


for MOD in "${MODS[@]}"; do
  go mod edit "-replace=${MOD}=${MOD}@${V}"
done
go mod tidy
