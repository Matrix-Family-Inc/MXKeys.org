Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Created

# ADR-0006: File Header Standard

## Status

Accepted

## Context

Every source, configuration, documentation, and infrastructure file in
the repository began with a multi-field banner (`Project`, `Company`,
`Owner`, `Maintainer`, `Role`, `Contact`, `Support`, `Matrix`, `Date`,
`Status`). Fields drifted across files, making provenance checks noisy
and reviewers uncertain which identity to trust when multiple conflicted.

The project-level user-rule for the maintaining organization specifies
a minimal header shape (`Project`, `Company`, `Maintainer`, `Date`,
`Status`, `Contact`). Bringing the tree into alignment removes
duplicated metadata and lets automated lint catch stale or missing
headers.

## Decision

Every tracked source, config, doc, workflow, and shell script must
carry the following header at the top of the file (or immediately after
a shebang line in executables):

```text
Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: <Day Mon DD YYYY UTC>
Status: <Created|Updated>
```

Render rules per file type:

- Go / TypeScript / CSS: wrap in `/* ... */` with ` * ` per line.
- Markdown: raw text block as the first content before the H1.
- YAML / shell / Dockerfile: `# ` per line.
- SQL: `-- ` per line, optional (not mandatory; migrations use a
  prose-comment block instead).

Fields dropped vs. the previous banner:

- `Owner`: redundant with `Company`.
- `Role`: duplicated the commit-author signature the repository already
  has via `git log`.
- `Support`: covered by `Contact`.
- `Matrix`: links belonged in `docs/federation-behavior.md`, not on
  every file.

## Consequences

- Every file has a single source of truth for provenance.
- Diffs are narrower: touching a file flips only `Date` and `Status`,
  not four redundant fields.
- Operator forks rebrand by editing `Company`, `Maintainer`, and
  `Contact` once per file (scripted; header shape is stable enough to
  sed-replace safely).
- `docs/release-process.md` can enforce this via a lint step; that is
  a follow-up.

## Alternatives Considered

- Leave the existing multi-field banner: rejected for churn and
  conflict potential.
- Drop headers entirely and rely on SPDX + `git blame`: rejected
  because operators cloning the repo want provenance in the file,
  not in a separate database query.

## References

- `LICENSE` header of every tracked file.
- `docs/release-process.md` (future header-lint gate).
