Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

# ADR-0006: File Header Standard

## Status

Accepted (supersedes prior Maintainer-based banner).

## Visibility

Public.

## Ecosystem Scope

This ADR is the MXKeys rendering of the Matrix Family file-header
standard documented in `docs/matrix-family-standardization.md`. The
ecosystem policy owns the cross-project header shape; this file owns
MXKeys-specific field values and comment rendering rules.

## Context

Source, configuration, documentation, and infrastructure files
carried drifting provenance banners, including personal `Maintainer`
and `Dev` fields. Matrix Family standardized on a corporate header
with no personal names.

## Decision

Every tracked source, config, doc, workflow, and shell script
carries the Matrix Family header at the top of the file (immediately
after the shebang line in executables).

MXKeys field values are:

- `Project`: `MXKeys (mxkeys.org)`
- `Company`: `Matrix Family Inc. (https://matrix.family)`
- `Owner`: `Matrix Family Inc.`
- `Contact`: `dev@matrix.family`
- `Support`: `support@matrix.family`
- `Matrix`: `@support:matrix.family`

Personal fields such as `Maintainer`, `Dev`, and `Role` are not
allowed. Comment rendering follows `docs/matrix-family-standardization.md`;
SQL migrations may use a prose-comment header instead of the generic
SQL rendering. Bulk normalization lives in
`scripts/apply-matrix-family-headers.sh`.

## Consequences

- A single source of truth for provenance per file across Matrix Family.
- Operator forks rebrand by replacing `Company`, `Contact`, `Support`.
- Compliance checks: `rg 'Maintainer:|Dev: |Role:'` must be empty.

## Alternatives Considered

None recorded at authoring time.

## References

- `docs/matrix-family-standardization.md` — checklist and git SSH port 42224.
