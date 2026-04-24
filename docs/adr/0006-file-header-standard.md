Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Fri Apr 24 2026 UTC
Status: Updated

# ADR-0006: File Header Standard

## Status

Accepted.

## Visibility

Public.

## Ecosystem Scope

This ADR is the MXKeys rendering of
`../../../ecosystem-docs/adr/ECO-0005-file-header-standard.md`. The ecosystem
ADR owns the cross-project header policy; this file owns MXKeys-specific field
values and comment rendering rules.

## Context

Source, configuration, documentation, and infrastructure files
carried a multi-field banner (`Project`, `Company`, `Owner`,
`Maintainer`, `Role`, `Contact`, `Support`, `Matrix`, `Date`,
`Status`). Fields drifted across files and it was unclear which
identity was authoritative when two fields disagreed.

The organization user-rule specifies a minimal header shape
(`Project`, `Company`, `Maintainer`, `Date`, `Status`, `Contact`).

## Decision

Every tracked source, config, doc, workflow, and shell script
carries the following header at the top of the file (immediately
after the shebang line in executables):

```text
Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: <Day Mon DD YYYY UTC>
Status: <Created|Updated>
```

Render rules per file type:

- Go / TypeScript / CSS: `/* ... */` block with ` * ` per line.
- Markdown: raw text block as the first content before the H1.
- YAML / shell / Dockerfile: `# ` per line.
- SQL: `-- ` per line. Optional for migrations, which use a
  prose-comment block instead.

Fields dropped vs. the previous banner:

- `Owner`: redundant with `Company`.
- `Role`: duplicated the commit-author signature already available
  through `git log`.
- `Support`: covered by `Contact`.
- `Matrix`: belongs in `docs/federation-behavior.md`, not on every
  file.

## Consequences

- A single source of truth for provenance per file.
- Narrower diffs: editing a file flips only `Date` and `Status`,
  not four redundant fields.
- Operator forks rebrand by replacing `Company`, `Maintainer`, and
  `Contact`. The header shape is stable enough for `sed`.

## Alternatives Considered

- Keep the multi-field banner. Rejected: churn and conflict
  between fields.
- Drop headers, rely on SPDX plus `git blame`. Rejected:
  operators want provenance in the file, not in a separate query.

## References

- ECO-0005 File Header Standard - canonical ecosystem policy for provenance
  headers.
- `LICENSE` - repository legal context.
- `docs/release-process.md` - release process that updates provenance metadata.

## Alternatives

None recorded at authoring time. Any future revision that modifies this decision must list the rejected options explicitly.
