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

// Exit codes for machine-readable integration
const (
	ExitOK                = 0
	ExitUsageError        = 1
	ExitFetchError        = 2
	ExitSignatureInvalid  = 3
	ExitConsistencyFailed = 4
	ExitIOError           = 5
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
	ServerName    string `json:"server_name"`
	KeyID         string `json:"key_id"`
	Algorithm     string `json:"algorithm"`
	PublicKey     string `json:"public_key"`
	Fingerprint   string `json:"fingerprint"`
	SelfSignature string `json:"self_signature"`
	SignPayload   string `json:"sign_payload"`
}

type verifyResult struct {
	OK               bool   `json:"ok"`
	Server           string `json:"server"`
	KeyFingerprint   string `json:"key_fingerprint"`
	TreeSize         int    `json:"tree_size"`
	RootHash         string `json:"root_hash"`
	Timestamp        string `json:"timestamp"`
	SignatureValid   bool   `json:"signature_valid"`
	ConsistencyValid *bool  `json:"consistency_valid,omitempty"`
	PrevTreeSize     *int   `json:"prev_tree_size,omitempty"`
	Error            string `json:"error,omitempty"`
}

func main() {
	baseURL := flag.String("url", "", "MXKeys server base URL (e.g. https://mxkeys.org)")
	prevFile := flag.String("prev", "", "Path to previous STH JSON for consistency check")
	outFile := flag.String("out", "", "Save current STH to file for future consistency checks")
	jsonOutput := flag.Bool("json", false, "Machine-readable JSON output")
	flag.Parse()

	if *baseURL == "" {
		if *jsonOutput {
			outputJSON(verifyResult{Error: "url parameter required"})
		} else {
			fmt.Fprintln(os.Stderr, "Usage: mxkeys-verify -url <base-url> [-prev <prev-sth.json>] [-out <sth.json>] [-json]")
		}
		os.Exit(ExitUsageError)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	result := verifyResult{}

	logf := func(format string, args ...interface{}) {
		if !*jsonOutput {
			fmt.Printf(format+"\n", args...)
		}
	}

	logf("Fetching notary public key...")
	key, err := fetchJSON[notaryKey](client, *baseURL+"/_mxkeys/notary/key")
	if err != nil {
		result.Error = fmt.Sprintf("fetch public key: %v", err)
		fail(*jsonOutput, result, ExitFetchError)
	}
	result.Server = key.ServerName
	result.KeyFingerprint = key.Fingerprint
	logf("  Server:      %s", key.ServerName)
	logf("  Key ID:      %s", key.KeyID)
	logf("  Fingerprint: %s", key.Fingerprint)

	pubKeyBytes, err := base64.RawStdEncoding.DecodeString(key.PublicKey)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		result.Error = "invalid public key encoding or size"
		fail(*jsonOutput, result, ExitFetchError)
	}

	logf("\nFetching signed tree head...")
	sth, err := fetchJSON[signedTreeHead](client, *baseURL+"/_mxkeys/transparency/signed-head")
	if err != nil {
		result.Error = fmt.Sprintf("fetch STH: %v", err)
		fail(*jsonOutput, result, ExitFetchError)
	}
	result.TreeSize = sth.TreeSize
	result.RootHash = sth.RootHash
	result.Timestamp = sth.Timestamp
	logf("  Tree size:  %d", sth.TreeSize)
	logf("  Root hash:  %s", sth.RootHash)
	logf("  Timestamp:  %s", sth.Timestamp)

	logf("\nVerifying signature...")
	sigBytes, err := base64.RawStdEncoding.DecodeString(sth.Signature)
	if err != nil {
		result.Error = "invalid signature encoding"
		fail(*jsonOutput, result, ExitSignatureInvalid)
	}

	if !ed25519.Verify(ed25519.PublicKey(pubKeyBytes), []byte(sth.SignPayload), sigBytes) {
		result.SignatureValid = false
		result.Error = "STH signature verification failed"
		fail(*jsonOutput, result, ExitSignatureInvalid)
	}
	result.SignatureValid = true
	logf("  OK: Signature is valid")

	if sth.Signer != key.ServerName {
		logf("  WARN: Signer mismatch: STH says %s, key says %s", sth.Signer, key.ServerName)
	}

	if *prevFile != "" {
		logf("\nChecking consistency with previous STH...")
		prevData, err := os.ReadFile(*prevFile)
		if err != nil {
			result.Error = fmt.Sprintf("read previous STH: %v", err)
			fail(*jsonOutput, result, ExitIOError)
		}
		var prev signedTreeHead
		if err := json.Unmarshal(prevData, &prev); err != nil {
			result.Error = fmt.Sprintf("parse previous STH: %v", err)
			fail(*jsonOutput, result, ExitIOError)
		}

		prevSize := prev.TreeSize
		result.PrevTreeSize = &prevSize
		logf("  Previous: size=%d root=%s", prev.TreeSize, truncHash(prev.RootHash))
		logf("  Current:  size=%d root=%s", sth.TreeSize, truncHash(sth.RootHash))

		consistent := true
		if sth.TreeSize < prev.TreeSize {
			consistent = false
			result.Error = "tree size decreased (possible rollback)"
		} else if sth.TreeSize == prev.TreeSize && sth.RootHash != prev.RootHash {
			consistent = false
			result.Error = "same size but different root (tree was modified)"
		} else if sth.TimestampMS < prev.TimestampMS {
			consistent = false
			result.Error = "timestamp went backwards"
		}
		result.ConsistencyValid = &consistent

		if !consistent {
			fail(*jsonOutput, result, ExitConsistencyFailed)
		}
		if sth.TreeSize > prev.TreeSize {
			logf("  OK: Tree grew from %d to %d entries (append-only)", prev.TreeSize, sth.TreeSize)
		} else {
			logf("  OK: Tree unchanged since last check")
		}
	}

	if *outFile != "" {
		data, _ := json.MarshalIndent(sth, "", "  ")
		if err := os.WriteFile(*outFile, data, 0644); err != nil {
			result.Error = fmt.Sprintf("save STH: %v", err)
			fail(*jsonOutput, result, ExitIOError)
		}
		logf("\nSTH saved to %s", *outFile)
	}

	result.OK = true
	if *jsonOutput {
		outputJSON(result)
	} else {
		fmt.Println("\nAll checks passed.")
	}
	os.Exit(ExitOK)
}

func fail(jsonMode bool, result verifyResult, code int) {
	if jsonMode {
		outputJSON(result)
	} else {
		fmt.Fprintf(os.Stderr, "FAIL: %s\n", result.Error)
	}
	os.Exit(code)
}

func outputJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func truncHash(h string) string {
	if len(h) > 16 {
		return h[:16] + "..."
	}
	return h
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
