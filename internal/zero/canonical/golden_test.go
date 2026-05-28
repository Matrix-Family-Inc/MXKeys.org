package canonical

import (
	stdjson "encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCanonicalJSONGoldenVectors is the byte-for-byte conformance suite.
// Every vector in testdata/golden_vectors.json must transform from input
// to canonical exactly. Any regression here implies that MXKeys and any
// peer using the same canonical JSON contract will disagree on
// signature-covered bytes, which breaks Matrix federation.
//
// Vectors cover: empty container shapes, key ordering, whitespace
// stripping, integer boundary values, string escapes, mixed arrays,
// deeply nested objects, plus two realistic federation shapes
// (signed_tree_head, server_keys).
func TestCanonicalJSONGoldenVectors(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "golden_vectors.json"))
	if err != nil {
		t.Fatalf("read golden vectors: %v", err)
	}

	var vectors []struct {
		Name      string `json:"name"`
		Input     string `json:"input"`
		Canonical string `json:"canonical"`
	}
	if err := stdjson.Unmarshal(raw, &vectors); err != nil {
		t.Fatalf("parse golden vectors: %v", err)
	}
	if len(vectors) == 0 {
		t.Fatal("golden vectors file is empty")
	}

	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			got, err := JSON([]byte(v.Input))
			if err != nil {
				t.Fatalf("JSON(%q) failed: %v", v.Input, err)
			}
			if string(got) != v.Canonical {
				t.Fatalf("canonical output mismatch:\n got: %q\nwant: %q", got, v.Canonical)
			}

			// Idempotence re-check per vector: canonicalizing the canonical
			// output yields the same bytes. Guards against drift between
			// the fuzz invariant and the concrete fixture.
			again, err := JSON(got)
			if err != nil {
				t.Fatalf("JSON(canonical) failed: %v", err)
			}
			if string(again) != v.Canonical {
				t.Fatalf("canonical is not a fixed point:\n first: %q\nsecond: %q", got, again)
			}
		})
	}
}
