#!/usr/bin/env bash
# Project: Matrix Family Ecosystem
# Company: Matrix Family Inc.
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: 2026-04-24
# Status: Created
#
# Activate repository-local Git hooks by pointing core.hooksPath at .githooks.
#
# This is idempotent and safe to rerun. Run once per clone on a new machine.
#
# Usage:
#   ecosystem-docs/scripts/install-hooks.sh
#   or from any repo: ./scripts/install-hooks.sh

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel 2>/dev/null || true)
if [[ -z "$repo_root" ]]; then
    echo "not inside a git working tree" >&2
    exit 1
fi

hooks_dir="$repo_root/.githooks"
if [[ ! -d "$hooks_dir" ]]; then
    echo "no .githooks/ directory at $repo_root" >&2
    exit 1
fi

chmod +x "$hooks_dir"/* 2>/dev/null || true
git -C "$repo_root" config core.hooksPath .githooks
echo "core.hooksPath set to .githooks in $repo_root"
