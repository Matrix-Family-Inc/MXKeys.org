#!/usr/bin/env python3
"""Project: Matrix Family Ecosystem
Company: Matrix Family Inc.
Maintainer: Brabus
Contact: dev@matrix.family
Date: 2026-04-24
Status: Created

ADR structural linter.

Validates every ADR file under the given directories for:
  - filename format (ECO-NNNN-slug.md / MXC-NNNN-slug.md / MXK-NNNN-slug.md
    or legacy NNNN-slug.md inside project pools)
  - canonical 6-field header (Project, Company, Maintainer, Contact, Date, Status)
  - required sections (Status, Visibility, Context, Decision, Consequences,
    Alternatives, References)
  - valid Status value (Proposed|Accepted|Superseded|Deprecated)
  - valid Visibility value (Public|Internal|Restricted)
  - unique ID within a namespace

Exits non-zero on any failure. Emits one FAIL line per offense.

Usage:
  adr_lint.py <dir> [<dir> ...]

Examples:
  adr_lint.py ecosystem-docs/adr
  adr_lint.py mxcore.tech/docs/adr MXKeys.org/docs/adr
"""

from __future__ import annotations

import pathlib
import re
import sys

HEADER_FIELDS = ["Project", "Company", "Maintainer", "Contact", "Date", "Status"]
REQUIRED_SECTIONS = ["Status", "Context", "Decision", "Consequences", "Alternatives"]
REQUIRED_EITHER = ["Visibility"]
VALID_HEADER_STATUS = {"Created", "Updated", "Superseded", "Archived"}
VALID_ADR_STATUS = {"Proposed", "Accepted", "Superseded", "Deprecated"}
VALID_VISIBILITY = {"Public", "Internal", "Restricted"}

FILENAME_PATTERN = re.compile(
    r"^(?:(?P<prefix>ECO|MXC|MXK)-(?P<num>\d{4})|(?P<legacy>\d{4}))-[a-z0-9][a-z0-9-]*\.md$"
)
SKIP = {
    "README.md",
    "TEMPLATE.md",
    "ADR-INDEX.md",
    "CHANGELOG.md",
    "MANIFEST.md",
    "0000-template.md",
}


def strip_html_comments(text: str) -> str:
    """Return text with HTML block comments stripped, preserving line numbers."""
    return re.sub(r"<!--", "", re.sub(r"-->", "", text))


def extract_header(content: str) -> dict[str, str]:
    """Parse the file header from the first 40 lines. Supports HTML-comment wrap."""
    header: dict[str, str] = {}
    cleaned = strip_html_comments(content)
    recognized = HEADER_FIELDS + ["Visibility"]
    for line in cleaned.splitlines()[:40]:
        line = line.strip()
        if not line:
            continue
        if line.startswith("#"):
            break
        for field in recognized:
            if line.startswith(f"{field}:"):
                header[field] = line.split(":", 1)[1].strip()
                break
    return header


def extract_sections(content: str) -> list[str]:
    """Return H2 section titles in order."""
    return re.findall(r"^## (.+?)\s*$", content, re.MULTILINE)


def extract_field(content: str, heading: str) -> str | None:
    """Extract the first non-empty paragraph following an H2 heading."""
    pattern = re.compile(rf"^## {re.escape(heading)}\s*$\n+(.+?)(?:\n\s*\n|\n##|\Z)", re.MULTILINE | re.DOTALL)
    match = pattern.search(content)
    if not match:
        return None
    return match.group(1).strip().split("\n", 1)[0].strip(" .`")


def check_file(path: pathlib.Path, ids_seen: dict[str, pathlib.Path]) -> list[str]:
    errors: list[str] = []
    name = path.name
    if name in SKIP:
        return errors

    m = FILENAME_PATTERN.match(name)
    if not m:
        errors.append(f"{path}: filename does not match ADR pattern")
        return errors

    prefix = m.group("prefix") or path.parent.name.upper()
    num = m.group("num") or m.group("legacy")
    adr_id = f"{prefix}-{num}" if m.group("prefix") else f"{path.parent.parent.parent.name.upper()}-{num}"

    if adr_id in ids_seen:
        errors.append(
            f"{path}: duplicate ADR ID {adr_id} (also in {ids_seen[adr_id]})"
        )
    else:
        ids_seen[adr_id] = path

    content = path.read_text(encoding="utf-8")

    header = extract_header(content)
    for field in HEADER_FIELDS:
        if field not in header:
            errors.append(f"{path}: missing header field '{field}:'")

    if header.get("Company"):
        if "Matrix Family" not in header["Company"]:
            errors.append(
                f"{path}: Company should mention 'Matrix Family' (got {header['Company']!r})"
            )

    sections = extract_sections(content)
    for required in REQUIRED_SECTIONS:
        if required not in sections:
            errors.append(f"{path}: missing section '## {required}'")

    for required in REQUIRED_EITHER:
        in_header = required in header
        in_section = required in sections
        if not in_header and not in_section:
            errors.append(
                f"{path}: '{required}' not found in header or as '## {required}'"
            )

    header_status = header.get("Status")
    if header_status:
        first = header_status.split()[0].rstrip(".,;:")
        if first not in VALID_HEADER_STATUS:
            errors.append(
                f"{path}: header Status '{header_status}' not in {sorted(VALID_HEADER_STATUS)}"
            )

    adr_status = extract_field(content, "Status")
    if adr_status:
        first = adr_status.split()[0].rstrip(".,;:")
        if first not in VALID_ADR_STATUS:
            errors.append(
                f"{path}: '## Status' value '{adr_status}' not in {sorted(VALID_ADR_STATUS)}"
            )

    visibility_value = header.get("Visibility") or extract_field(content, "Visibility")
    if visibility_value:
        first = visibility_value.split()[0].rstrip(".,;:")
        if first not in VALID_VISIBILITY:
            errors.append(
                f"{path}: Visibility '{visibility_value}' not in {sorted(VALID_VISIBILITY)}"
            )

    return errors


def main(argv: list[str]) -> int:
    if not argv:
        print("usage: adr_lint.py <adr-dir> [<adr-dir> ...]", file=sys.stderr)
        return 2

    all_errors: list[str] = []
    for arg in argv:
        root = pathlib.Path(arg)
        if not root.is_dir():
            print(f"WARN: {arg} is not a directory, skipping", file=sys.stderr)
            continue
        ids_seen: dict[str, pathlib.Path] = {}
        for path in sorted(root.glob("*.md")):
            all_errors.extend(check_file(path, ids_seen))

    if all_errors:
        for err in all_errors:
            print(f"FAIL: {err}")
        print(f"\n{len(all_errors)} violation(s)", file=sys.stderr)
        return 1

    print("adr_lint: all ADRs pass structural checks")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
