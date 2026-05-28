/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

// mxkeys-walctl is the operator-facing tool for offline Raft WAL
// maintenance. Its single supported subcommand today is `upgrade`,
// which converts a legacy v2 WAL (CRC32C only) into the current v3
// format (CRC32C + HMAC-SHA256) without losing entries.
//
// Usage:
//
//	mxkeys-walctl upgrade \
//	  --dir /var/lib/mxkeys/raft \
//	  --secret-env MXKEYS_CLUSTER_SECRET
//
// The tool MUST be run while the notary service is stopped.
package main

import (
	"flag"
	"fmt"
	"os"

	"mxkeys/internal/zero/raft/walupgrade"
)

const (
	exitOK          = 0
	exitUsageError  = 1
	exitUpgradeFail = 2
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(exitUsageError)
	}
	switch os.Args[1] {
	case "upgrade":
		os.Exit(runUpgrade(os.Args[2:]))
	case "help", "-h", "--help":
		usage(os.Stdout)
		os.Exit(exitOK)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %q\n\n", os.Args[1])
		usage(os.Stderr)
		os.Exit(exitUsageError)
	}
}

func runUpgrade(args []string) int {
	fs := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	dir := fs.String("dir", "", "Raft state directory (required)")
	secretEnv := fs.String("secret-env", "MXKEYS_CLUSTER_SECRET",
		"name of the environment variable that holds the cluster shared secret")
	backup := fs.Bool("backup", true, "save the original WAL as raft.wal.v2-backup")
	if err := fs.Parse(args); err != nil {
		return exitUsageError
	}
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "mxkeys-walctl upgrade: --dir is required")
		return exitUsageError
	}
	secret := os.Getenv(*secretEnv)
	if secret == "" {
		fmt.Fprintf(os.Stderr,
			"mxkeys-walctl upgrade: %s is empty; export the shared secret to that env var\n",
			*secretEnv)
		return exitUsageError
	}

	report, err := walupgrade.Upgrade(walupgrade.Options{
		Dir:      *dir,
		HMACKey:  []byte(secret),
		KeepV2:   *backup,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "mxkeys-walctl upgrade failed: %v\n", err)
		return exitUpgradeFail
	}
	fmt.Printf("upgraded %d records from v2 to v3\n", report.Records)
	fmt.Printf("v3 file: %s\n", report.V3Path)
	if report.V2BackupPath != "" {
		fmt.Printf("v2 backup: %s\n", report.V2BackupPath)
	}
	return exitOK
}

func usage(w *os.File) {
	fmt.Fprintf(w, `mxkeys-walctl: offline Raft WAL maintenance

Usage:
  mxkeys-walctl <subcommand> [flags]

Subcommands:
  upgrade   Convert a legacy v2 WAL to the v3 format (CRC + HMAC).
            Run while the notary service is stopped.
  help      Print this help text.

Run "mxkeys-walctl <subcommand> -h" for subcommand-specific flags.
`)
}
