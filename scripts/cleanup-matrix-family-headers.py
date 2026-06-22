#!/usr/bin/env python3
# Project: MXKeys (mxkeys.org)
# Company: Matrix Family Inc. (https://matrix.family)
# Owner: Matrix Family Inc.
# Contact: dev@matrix.family
# Support: support@matrix.family
# Matrix: @support:matrix.family
# Date: Sun 22 Jun 2026 00:52:00 UTC
# Status: Updated

"""Dedupe Matrix Family headers and strip legacy Maintainer blocks."""

from __future__ import annotations

import re
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
UTC = datetime.now(timezone.utc).strftime("%a %d %b %Y %H:%M:%S UTC")

STANDARD_MD = f"""Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: {UTC}
Status: Updated"""

STANDARD_BLOCK = f"""/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: {UTC}
 * Status: Updated
 */"""

HEADER_LINE = re.compile(
    r"^(Project:|Company:|Owner:|Maintainer:|Contact:|Support:|Matrix:|Date:|Status:)"
)
BLOCK_RE = re.compile(r"/\*[\s\S]*?\*/\s*\n?", re.MULTILINE)


def strip_md_headers(text: str) -> str:
    lines = text.splitlines(keepends=True)
    i = 0
    while i < len(lines):
        if lines[i].startswith("Project:"):
            while i < len(lines) and (
                HEADER_LINE.match(lines[i].rstrip("\n")) or lines[i].strip() == ""
            ):
                i += 1
            continue
        break
    return "".join(lines[i:])


def clean_markdown(path: Path) -> bool:
    original = path.read_text(encoding="utf-8")
    body = strip_md_headers(original).lstrip("\n")
    text = STANDARD_MD + "\n\n" + body
    if text != original:
        path.write_text(text, encoding="utf-8")
        return True
    return False


def clean_go_file(path: Path) -> bool:
    original = path.read_text(encoding="utf-8")
    blocks = list(BLOCK_RE.finditer(original))
    if not blocks:
        return False
    first = blocks[0]
    tail = original[first.end() :]
    tail = BLOCK_RE.sub("", tail)
    text = first.group(0)
    if not text.endswith("\n"):
        text += "\n"
    text += tail
    if text != original:
        path.write_text(text, encoding="utf-8")
        return True
    return False


def scrub_maintainer_footer(text: str) -> str:
    return re.sub(
        r"\n(?:Maintainer|Dev): [^\n]+\n",
        "\n",
        text,
    )


def main() -> None:
    changed = 0
    for path in list(ROOT.rglob("*.md")) + [ROOT / "LICENSE"]:
        if not path.is_file() or ".git" in path.parts or "node_modules" in path.parts:
            continue
        if clean_markdown(path):
            changed += 1
    for path in ROOT.rglob("*.go"):
        if ".git" in path.parts:
            continue
        if clean_go_file(path):
            changed += 1
    readme = ROOT / "README.md"
    if readme.exists():
        t = scrub_maintainer_footer(readme.read_text(encoding="utf-8"))
        readme.write_text(t, encoding="utf-8")
    print(f"cleaned {changed} files")


if __name__ == "__main__":
    main()
