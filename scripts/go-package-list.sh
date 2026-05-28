#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Tue Apr 07 2026 UTC
# Status: Created

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
