Project: MXKeys (mxkeys.org)
Company: Matrix.Family Inc. (https://matrix.family)
Owner: Matrix.Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Sat 21 Nov 2026 02:00:00 UTC
Status: Updated

# Architecture Decision Records

These records are about this notary. Numbers that are absent were
not decisions about the notary and are not stored here. Later numbers
were not renumbered. New records copy `TEMPLATE.md`.

| Record | Status | Decision |
|--------|--------|----------|
| [0001](0001-cluster-consensus-modes.md) | Accepted | Cluster is off by default. When on, `crdt` or `raft`. |
| [0002](0002-zero-dependency-core-packages.md) | Accepted | Canonical JSON, metrics, and config stay in-tree. |
| [0003](0003-postgresql-as-persistent-cache.md) | Accepted | Verified key responses are stored in PostgreSQL. |
| [0004](0004-strict-json-and-matrix-error-contract.md) | Accepted | Requests are strict JSON. Errors use Matrix `errcode`. |
| [0005](0005-core-api-stability-scope.md) | Accepted | Only the Matrix key API is the stable contract. |
| [0007](0007-signing-key-provider.md) | Accepted | The notary signing key is loaded through a provider. |
| [0008](0008-schema-migrations.md) | Accepted | Schema changes are versioned SQL migrations. |
| [0009](0009-landing-fsd-stack.md) | Accepted | The public page of this notary uses Feature-Sliced Design. |
