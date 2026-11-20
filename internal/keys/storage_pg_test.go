/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"mxkeys/internal/storage/migrations"
)

// testDB opens the PostgreSQL database named by MXKEYS_TEST_DATABASE_URL,
// applies migrations and empties the tables. Tests skip without it.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("MXKEYS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MXKEYS_TEST_DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := migrations.Apply(db); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	if _, err := db.Exec(`TRUNCATE server_keys, server_key_responses, key_transparency_log, server_info_cache`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func signedDocument(t *testing.T, serverName string, validUntil time.Time) (*ServerKeysResponse, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raw := buildSignedOriginPayload(t, serverName, "ed25519:o", priv, map[string]interface{}{
		"old_verify_keys": map[string]interface{}{},
		"valid_until_ts":  validUntil.UnixMilli(),
	})
	doc, _, err := parseServerKeys(raw)
	if err != nil {
		t.Fatal(err)
	}
	return doc, pub
}

func TestStorageRoundTripsDocumentBytes(t *testing.T) {
	db := testDB(t)
	storage, _ := NewStorage(db)
	doc, _ := signedDocument(t, "origin.example", time.Now().Add(time.Hour))

	if err := storage.StoreServerResponse(doc.ServerName, doc, time.UnixMilli(doc.ValidUntilTS)); err != nil {
		t.Fatal(err)
	}
	got, fetchedAt, err := storage.GetServerResponse(context.Background(), doc.ServerName)
	if err != nil || got == nil {
		t.Fatalf("GetServerResponse: %v %v", got, err)
	}
	if string(got.Raw) != string(doc.Raw) {
		t.Fatal("stored document bytes must round-trip unchanged")
	}
	if time.Since(fetchedAt) > time.Minute {
		t.Fatalf("fetched_at = %v", fetchedAt)
	}
}

func TestStorageIgnoresRowsWithoutDocumentBytes(t *testing.T) {
	db := testDB(t)
	storage, _ := NewStorage(db)
	if _, err := db.Exec(`INSERT INTO server_key_responses (server_name, response, valid_until) VALUES ('legacy.example', '{}', NOW() + INTERVAL '1 hour')`); err != nil {
		t.Fatal(err)
	}
	got, _, err := storage.GetServerResponse(context.Background(), "legacy.example")
	if err != nil || got != nil {
		t.Fatalf("a row without raw bytes must read as absent, got %v %v", got, err)
	}
}

func TestDeleteExpiredKeysKeepsDocumentsWithinRetention(t *testing.T) {
	db := testDB(t)
	storage, _ := NewStorage(db)
	recent, _ := signedDocument(t, "recent.example", time.Now().Add(-24*time.Hour))
	old, _ := signedDocument(t, "old.example", time.Now().Add(-400*24*time.Hour))
	for _, doc := range []*ServerKeysResponse{recent, old} {
		if err := storage.StoreServerResponse(doc.ServerName, doc, time.UnixMilli(doc.ValidUntilTS)); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := storage.DeleteExpiredKeys(context.Background(), 365*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := storage.GetServerResponse(context.Background(), "recent.example"); got == nil {
		t.Fatal("a document expired within the retention window must be kept")
	}
	if got, _, _ := storage.GetServerResponse(context.Background(), "old.example"); got != nil {
		t.Fatal("a document expired beyond the retention window must be deleted")
	}
}

func TestQueryServesLastKnownDocumentWhenOriginIsDown(t *testing.T) {
	db := testDB(t)
	storage, _ := NewStorage(db)
	const serverName = "127.0.0.1:1" // nothing listens on port 1
	doc, originPub := signedDocument(t, serverName, time.Now().Add(-time.Hour))
	if err := storage.StoreServerResponse(serverName, doc, time.UnixMilli(doc.ValidUntilTS)); err != nil {
		t.Fatal(err)
	}

	n, notaryPub := testNotary(t)
	n.storage = storage
	n.fetcher = NewFetcherWithConfig(FetcherConfig{Timeout: time.Second, RetryAttempts: 1})
	n.fetcher.SetBlockPrivateIPs(false)

	resp := n.QueryKeys(context.Background(), &KeyQueryRequest{ServerKeys: map[string]map[string]KeyCriteria{serverName: {}}})
	if len(resp.ServerKeys) != 1 {
		t.Fatalf("the last known document must be served when the origin is down, failures: %v", resp.Failures)
	}
	served := encodeLikeHandler(t, resp.ServerKeys[0])
	if !verifyOriginSignature(t, served, serverName, "ed25519:o", originPub) {
		t.Fatal("origin signature must verify on the served document")
	}
	if !verifyOriginSignature(t, served, "notary.example", "ed25519:n", notaryPub) {
		t.Fatal("notary signature must verify on the served document")
	}
}
