#!/usr/bin/env bash
# Project: MXKeys (mxkeys.org)
# Company: Matrix Family Inc. (https://matrix.family)
# Owner: Matrix Family Inc.
# Contact: dev@matrix.family
# Support: support@matrix.family
# Matrix: @support:matrix.family
# Date: Mon 22 Jun 2026 00:50:40 UTC
# Status: Updated
set -euo pipefail

mode="${1:-imports}"

if [[ "${mode}" == "dirs" ]]; then
  while IFS= read -r line; do
    import_path="${line%% *}"
    dir_path="${line#* }"
    case "${import_path}" in
      mxkeys/landing/node_modules/*) ;;
      *) echo "${dir_path}" ;;
    esac
  done < <(go list -f '{{.ImportPath}} {{.Dir}}' ./...)
  exit 0
fi

while IFS= read -r pkg; do
  case "${pkg}" in
    mxkeys/landing/node_modules/*) ;;
    *) echo "${pkg}" ;;
  esac
done < <(go list ./...)
