#!/usr/bin/env bash
# Project: MXKeys (mxkeys.org)
# Company: Matrix Family Inc. (https://matrix.family)
# Owner: Matrix Family Inc.
# Contact: dev@matrix.family
# Support: support@matrix.family
# Matrix: @support:matrix.family
# Date: Mon 22 Jun 2026 00:50:40 UTC
# Status: Updated
# One-shot header normalizer for Matrix Family standardization.
# Safe to re-run: idempotent on already-normalized files.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UTC_DATE="$(date -u '+%a %d %b %Y %H:%M:%S UTC')"

BLOCK_HEADER="/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: ${UTC_DATE}
 * Status: Updated
 */"

MD_HEADER="Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: ${UTC_DATE}
Status: Updated"

HASH_HEADER="# Project: MXKeys (mxkeys.org)
# Company: Matrix Family Inc. (https://matrix.family)
# Owner: Matrix Family Inc.
# Contact: dev@matrix.family
# Support: support@matrix.family
# Matrix: @support:matrix.family
# Date: ${UTC_DATE}
# Status: Updated"

HTML_HEADER="<!--
Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: ${UTC_DATE}
Status: Updated
-->"

apply_block() {
  local file="$1"
  python3 - "$file" "$BLOCK_HEADER" <<'PY'
import re, sys
path, header = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as f:
    body = f.read()
old = re.compile(
    r"/\*[\s\S]*?\*/\s*\n",
    re.MULTILINE,
)
if old.match(body):
    body = old.sub(header + "\n\n", body, count=1)
else:
    body = header + "\n\n" + body
with open(path, "w", encoding="utf-8") as f:
    f.write(body)
PY
}

apply_md() {
  local file="$1"
  python3 - "$file" "$MD_HEADER" <<'PY'
import re, sys
path, header = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as f:
    body = f.read()
pat = re.compile(
    r"^(?:Project:.*\n)+?(?=\n#|\n##|\Z)",
    re.MULTILINE,
)
if pat.match(body):
    body = pat.sub(header + "\n\n", body, count=1)
else:
    body = header + "\n\n" + body
with open(path, "w", encoding="utf-8") as f:
    f.write(body)
PY
}

apply_hash() {
  local file="$1"
  python3 - "$file" "$HASH_HEADER" <<'PY'
import re, sys
path, header = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as f:
    lines = f.read().splitlines(keepends=True)
out = []
i = 0
if lines and lines[0].startswith("#!"):
    out.append(lines[0])
    i = 1
while i < len(lines) and lines[i].startswith("# ") and any(
    k in lines[i] for k in ("Project:", "Company:", "Owner:", "Maintainer:", "Contact:", "Support:", "Matrix:", "Date:", "Status:")
):
    i += 1
while i < len(lines) and lines[i].strip() == "#":
    i += 1
out.append(header + "\n")
if i < len(lines) and lines[i].strip() == "":
    i += 1
out.extend(lines[i:])
with open(path, "w", encoding="utf-8") as f:
    f.writelines(out)
PY
}

apply_html() {
  local file="$1"
  python3 - "$file" "$HTML_HEADER" <<'PY'
import re, sys
path, header = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as f:
    body = f.read()
pat = re.compile(r"<!--[\s\S]*?-->\s*\n", re.MULTILINE)
if pat.match(body):
    body = pat.sub(header + "\n\n", body, count=1)
else:
    body = header + "\n\n" + body
with open(path, "w", encoding="utf-8") as f:
    f.write(body)
PY
}

cd "$ROOT"

while IFS= read -r -d '' f; do
  apply_block "$f"
done < <(find . -type f \( -name '*.go' -o -name '*.ts' -o -name '*.tsx' -o -name '*.css' -o -name '*.mjs' \) \
  ! -path './landing/node_modules/*' ! -path './landing/dist/*' ! -path './.git/*' -print0)

while IFS= read -r -d '' f; do
  apply_md "$f"
done < <(find . -type f -name '*.md' ! -path './.git/*' -print0)

while IFS= read -r -d '' f; do
  apply_hash "$f"
done < <(find . -type f \( -name '*.sh' -o -name '*.yaml' -o -name '*.yml' -o -name 'Dockerfile' -o -name 'robots.txt' \) \
  ! -path './.git/*' -print0)

apply_html "landing/index.html"

echo "headers applied (${UTC_DATE})"
