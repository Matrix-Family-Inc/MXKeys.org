// Project: MXKeys (mxkeys.org)
// Company: Matrix Family Inc. (https://matrix.family)
// Owner: Matrix Family Inc.
// Contact: dev@matrix.family
// Support: support@matrix.family
// Matrix: @support:matrix.family
// Date: Thu 23 Jul 2026 02:34:00 UTC
// Status: Created
//
// Module boundary stub. The landing tree is a Bun/Vite frontend, not
// Go code, but some npm packages ship .go files (for example
// flatted/golang). Without this stub `go test ./...` from the repo
// root descends into landing/node_modules and picks those packages
// up. A nested go.mod excludes the whole subtree from the root
// module's ./... patterns; the CI helper scripts/go-package-list.sh
// keeps its own exclusion for older checkouts.

module mxkeys/landing

go 1.26
