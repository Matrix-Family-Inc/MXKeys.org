Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Fri Apr 24 2026 UTC
Status: Updated

# ADR-0010: File Size Policy

## Status

Accepted.

## Visibility

Public.

## Ecosystem Scope

This ADR applies
`../../../ecosystem-docs/adr/ECO-0006-file-size-policy.md` to MXKeys. The
ecosystem ADR owns the cross-project cohesion policy; this file owns the
MXKeys-specific warning and failure thresholds.

## Context

The organization-wide rule calls for files of "250 - 300 lines". An
earlier pass across this repository treated the bound as strict and
split cohesive files to fit.

Two costs of the strict reading were observed in review:

1. Single algorithms (Raft election, canonical JSON, WAL read path)
   were spread across six or seven files. Following a flow cost
   extra file jumps; a single-change refactor touched many files.
2. Test files that benefit from long case tables or golden vectors
   were cut at arbitrary line counts, which hurt readability.

The intent of the line budget is one responsibility per file with a
reading cost that fits in a review window. It is a target, not a
hard budget.

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
- Files already split below 300 lines in earlier passes stay as
  they are. They are recombined only if a reader finds the split
  confusing.
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

- ECO-0006 File Size and Cohesion Policy - canonical ecosystem cohesion policy.
- Organization user-rule - source guideline for the 250 to 300 line target.
- `scripts/file-size-lint.sh` - MXKeys local line-count enforcement.
- `.github/workflows/pr-gate.yml` - CI job that runs the file-size check.

## Alternatives

None recorded at authoring time. Any future revision that modifies this decision must list the rejected options explicitly.
