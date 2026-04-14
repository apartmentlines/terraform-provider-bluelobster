#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${REPO_ROOT}/dist"
BIN_NAME="terraform-provider-bluelobster"

TARGETS=(
  "linux/amd64"
  # "linux/arm64"
  # "darwin/amd64"
  # "darwin/arm64"
  # "windows/amd64"
  # "windows/arm64"
)

mkdir -p "${DIST_DIR}"

for target in "${TARGETS[@]}"; do
  IFS=/ read -r goos goarch <<< "${target}"

  ext=""
  if [ "${goos}" = "windows" ]; then
    ext=".exe"
  fi

  output="${DIST_DIR}/${BIN_NAME}_${goos}_${goarch}${ext}"

  echo "Building ${output}"
  CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
    go build -o "${output}" "${REPO_ROOT}"
done
