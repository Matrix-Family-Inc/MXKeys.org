/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testOrigin is a TLS origin serving its own signed key document after
// delay. Requests are counted.
type testOrigin struct {
	server *httptest.Server
	name   string
	hits   atomic.Int32

	mu  sync.Mutex
	doc []byte
}

func newTestOrigin(t *testing.T, delay time.Duration, release <-chan struct{}) *testOrigin {
	t.Helper()
	o := &testOrigin{}
	o.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.hits.Add(1)
		select {
		case <-time.After(delay):
		case <-release:
			return
		}
		o.mu.Lock()
		doc := o.doc
		o.mu.Unlock()
		_, _ = w.Write(doc)
	}))
	o.name = strings.TrimPrefix(o.server.URL, "https://")
	doc, _ := createSignedKeysResponse(t, o.name)
	o.mu.Lock()
	o.doc = doc
	o.mu.Unlock()
	return o
}

func notaryFetchingFrom(t *testing.T, origin *testOrigin) *Notary {
	t.Helper()
	n, _ := testNotary(t)
	n.fetcher = NewFetcherWithConfig(FetcherConfig{Timeout: 5 * time.Second, RetryAttempts: 1})
	n.fetcher.SetBlockPrivateIPs(false)
	n.fetcher.client = origin.server.Client()
	return n
}

func TestQueryKeysAnswersWithinDeadline(t *testing.T) {
	release := make(chan struct{})
	slow := newTestOrigin(t, time.Hour, release)
	defer slow.server.Close()
	defer close(release)

	n := notaryFetchingFrom(t, slow)
	n.queryDeadline = 300 * time.Millisecond

	cachedRaw, _ := createSignedKeysResponse(t, "cached.example")
	cached, _, err := parseServerKeys(cachedRaw)
	if err != nil {
		t.Fatal(err)
	}
	n.cache["cached.example"] = &cachedResponse{response: cached, validUntil: time.Now().Add(time.Hour), fetchedAt: time.Now()}

	start := time.Now()
	resp := n.QueryKeys(context.Background(), &KeyQueryRequest{
		ServerKeys: map[string]map[string]KeyCriteria{"cached.example": {}, slow.name: {}},
	})
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("query took %v, deadline is 300ms", elapsed)
	}
	if len(resp.ServerKeys) != 1 || resp.ServerKeys[0].ServerName != "cached.example" {
		t.Fatalf("answered servers must be returned, got %+v", resp.ServerKeys)
	}
	if _, ok := resp.Failures[slow.name]; !ok {
		t.Fatalf("unanswered server must be listed under failures, got %v", resp.Failures)
	}
}

func TestFetchSharedSurvivesCallerLeaving(t *testing.T) {
	origin := newTestOrigin(t, 300*time.Millisecond, nil)
	defer origin.server.Close()
	n := notaryFetchingFrom(t, origin)

	leaving, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	firstErr := make(chan error, 1)
	go func() {
		_, err := n.fetchShared(leaving, origin.name, FetchSourceDirect)
		firstErr <- err
	}()
	time.Sleep(20 * time.Millisecond)

	waiting, cancelWaiting := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelWaiting()
	keys, err := n.fetchShared(waiting, origin.name, FetchSourceDirect)
	if err != nil {
		t.Fatalf("a caller leaving must not fail the others: %v", err)
	}
	if keys.ServerName != origin.name {
		t.Fatalf("unexpected server %q", keys.ServerName)
	}
	if err := <-firstErr; err == nil {
		t.Fatal("the caller that left must see its own deadline")
	}
	if got := origin.hits.Load(); got != 1 {
		t.Fatalf("concurrent callers must share one fetch, origin saw %d requests", got)
	}
	if _, ok := n.memoryLookup(origin.name, time.Now()); !ok {
		t.Fatal("the shared fetch must populate the memory cache")
	}
}

func TestMinimumValidUntilRefetchIsRateLimited(t *testing.T) {
	origin := newTestOrigin(t, 0, nil)
	defer origin.server.Close()
	n := notaryFetchingFrom(t, origin)

	raw, _ := createSignedKeysResponse(t, origin.name)
	known, _, err := parseServerKeys(raw)
	if err != nil {
		t.Fatal(err)
	}
	farFuture := time.Now().Add(365 * 24 * time.Hour).UnixMilli()

	n.cache[origin.name] = &cachedResponse{response: known, validUntil: time.Now().Add(time.Hour), fetchedAt: time.Now()}
	if _, err := n.GetServerKeysWithCriteria(context.Background(), origin.name, farFuture); err != nil {
		t.Fatal(err)
	}
	if got := origin.hits.Load(); got != 0 {
		t.Fatalf("a document fetched just now must not be refetched, origin saw %d requests", got)
	}

	n.cache[origin.name] = &cachedResponse{response: known, validUntil: time.Now().Add(time.Hour), fetchedAt: time.Now().Add(-2 * minRefetchInterval)}
	if _, err := n.GetServerKeysWithCriteria(context.Background(), origin.name, farFuture); err != nil {
		t.Fatal(err)
	}
	if got := origin.hits.Load(); got != 1 {
		t.Fatalf("an older document must be refetched once, origin saw %d requests", got)
	}
}
