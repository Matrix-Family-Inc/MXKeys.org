Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Maintainer: Brabus
Role: Lead Architect
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon Mar 16 2026 UTC
Status: Created

# MXKeys Release Notes (Current Candidate)

## Changes

- Strengthened cryptographic validation and canonical JSON strictness.
- Improved deterministic query-path processing and trust policy enforcement.
- Added negative matrix tests for `errcode` and request validation.
- Added live interoperability checks for federation query strictness/compatibility/failure-path.
- Hardened secure-by-default notary signing key permissions (`0700` directory, `0600` key file).
- Added key backup/restore drill tests and rotation/compromise/DR runbook procedures.

## Known Constraints

- Branch protection enforcement (`A3.1`, `A3.2`) requires VCS-hosting verification with repository settings access.
- DB backup regularity (`E3.1`) depends on an external scheduler/operations layer and must be validated with separate operational evidence.
