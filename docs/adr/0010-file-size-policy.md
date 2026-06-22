Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

# ADR-0010: File Size Policy

## Status

Accepted.

## Visibility

Public.

## Context

The organization-wide rule calls for files of "250 - 300 lines".
Treating that bound as a hard cap over-split cohesive algorithms and
test case tables. The line budget is a reviewability target: one
responsibility per file, with a hard stop only when a file accumulates
unrelated concerns.

## Decision

- **Target**: 250 - 300 lines. This is the first question when a
  file grows: is it still one responsibility.
- **Ceiling**: 400 lines. Files above this limit require a top-of-
  file comment that states the reason (cohesion cost of splitting
  exceeds the navigation benefit).
- **Scope**: tracked source, test, and doc files edited by humans.
  Generated code, fixtures, and vendored data are exempt.
- **Enforcement**: `scripts/file-size-lint.sh` warns at 300 lines
  and fails at 400 lines. It runs in the `file-size` CI job and is
  in the required status checks for `main`.

## Consequences

- Source files keep local cohesion. A 320-line file holding one
  algorithm is preferred to two 170-line files that a reader has
  to alternate between.
- The 400-line ceiling rejects files that accumulate unrelated
  responsibilities while leaving room for coherent test tables.
- Tests are subject to the same policy. A coherent 400-line case
  table is acceptable.

## Alternatives Considered

- **Strict 250 - 300 cap**. Rejected. Over-split code was harder
  to read and refactor.
- **No cap**. Rejected. Files above ~400 lines accumulate
  unrelated concerns.
- **Lint-only enforcement without review judgement**. Kept as
  lint, not sole gate; code review still applies the cohesion
  question.

## References

- `scripts/file-size-lint.sh` - MXKeys local line-count enforcement.
- `.github/workflows/pr-gate.yml` - CI job that runs the file-size check.
