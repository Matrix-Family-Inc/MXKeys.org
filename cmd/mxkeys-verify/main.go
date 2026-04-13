/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type signedTreeHead struct {
	TreeSize    int    `json:"tree_size"`
	RootHash    string `json:"root_hash"`
	Timestamp   string `json:"timestamp"`
	TimestampMS int64  `json:"timestamp_ms"`
	Signer      string `json:"signer"`
	KeyID       string `json:"key_id"`
	Signature   string `json:"signature"`
	SignPayload string `json:"sign_payload"`
}

type notaryKey struct {
	ServerName  string `json:"server_name"`
	KeyID       string `json:"key_id"`
	Algorithm   string `json:"algorithm"`
	PublicKey   string `json:"public_key"`
	Fingerprint string `json:"fingerprint"`
}

func main() {
	baseURL := flag.String("url", "", "MXKeys server base URL (e.g. https://mxkeys.org)")
	prevFile := flag.String("prev", "", "Path to previous STH JSON for consistency check")
	outFile := flag.String("out", "", "Save current STH to file for future consistency checks")
	flag.Parse()

	if *baseURL == "" {
		fmt.Fprintln(os.Stderr, "Usage: mxkeys-verify -url <base-url> [-prev <prev-sth.json>] [-out <sth.json>]")
		os.Exit(1)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	fmt.Println("Fetching notary public key...")
	key, err := fetchJSON[notaryKey](client, *baseURL+"/_mxkeys/notary/key")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch public key: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Server:      %s\n", key.ServerName)
	fmt.Printf("  Key ID:      %s\n", key.KeyID)
	fmt.Printf("  Fingerprint: %s\n", key.Fingerprint)

	pubKeyBytes, err := base64.RawStdEncoding.DecodeString(key.PublicKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode public key: %v\n", err)
		os.Exit(1)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		fmt.Fprintf(os.Stderr, "Invalid public key size: %d\n", len(pubKeyBytes))
		os.Exit(1)
	}

	fmt.Println("\nFetching signed tree head...")
	sth, err := fetchJSON[signedTreeHead](client, *baseURL+"/_mxkeys/transparency/signed-head")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fetch STH: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  Tree size:  %d\n", sth.TreeSize)
	fmt.Printf("  Root hash:  %s\n", sth.RootHash)
	fmt.Printf("  Timestamp:  %s\n", sth.Timestamp)
	fmt.Printf("  Signer:     %s\n", sth.Signer)

	fmt.Println("\nVerifying signature...")
	sigBytes, err := base64.RawStdEncoding.DecodeString(sth.Signature)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode signature: %v\n", err)
		os.Exit(1)
	}

	if !ed25519.Verify(ed25519.PublicKey(pubKeyBytes), []byte(sth.SignPayload), sigBytes) {
		fmt.Fprintln(os.Stderr, "FAIL: Signature verification failed")
		os.Exit(1)
	}
	fmt.Println("  OK: Signature is valid")

	if sth.Signer != key.ServerName {
		fmt.Fprintf(os.Stderr, "WARN: Signer mismatch: STH says %s, key says %s\n", sth.Signer, key.ServerName)
	}

	if *prevFile != "" {
		fmt.Println("\nChecking consistency with previous STH...")
		prevData, err := os.ReadFile(*prevFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read previous STH: %v\n", err)
			os.Exit(1)
		}
		var prev signedTreeHead
		if err := json.Unmarshal(prevData, &prev); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse previous STH: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("  Previous: size=%d root=%s\n", prev.TreeSize, prev.RootHash[:16]+"...")
		fmt.Printf("  Current:  size=%d root=%s\n", sth.TreeSize, sth.RootHash[:16]+"...")

		if sth.TreeSize < prev.TreeSize {
			fmt.Fprintln(os.Stderr, "FAIL: Tree size decreased (possible rollback)")
			os.Exit(1)
		}
		if sth.TreeSize == prev.TreeSize && sth.RootHash != prev.RootHash {
			fmt.Fprintln(os.Stderr, "FAIL: Same size but different root (tree was modified)")
			os.Exit(1)
		}
		if sth.TimestampMS < prev.TimestampMS {
			fmt.Fprintln(os.Stderr, "FAIL: Timestamp went backwards")
			os.Exit(1)
		}
		if sth.TreeSize > prev.TreeSize {
			fmt.Printf("  OK: Tree grew from %d to %d entries (append-only)\n", prev.TreeSize, sth.TreeSize)
		} else {
			fmt.Println("  OK: Tree unchanged since last check")
		}
	}

	if *outFile != "" {
		data, _ := json.MarshalIndent(sth, "", "  ")
		if err := os.WriteFile(*outFile, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save STH: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nSTH saved to %s\n", *outFile)
	}

	fmt.Println("\nAll checks passed.")
}

func fetchJSON[T any](client *http.Client, url string) (*T, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
