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

package signedjson

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"testing"
)

// Test vectors from the Matrix specification, Appendices, "Signing JSON".
const (
	specSeed   = "YJDBA9Xnr2sVqXD9Vj7XVUnmFZcZrlw8Md7kMW+3XA1"
	specSigner = "domain"
	specKeyID  = "ed25519:1"
)

func specKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	seed, err := base64.RawStdEncoding.DecodeString(specSeed)
	if err != nil {
		t.Fatal(err)
	}
	return ed25519.NewKeyFromSeed(seed)
}

func TestSignMatchesSpecVectors(t *testing.T) {
	priv := specKey(t)
	cases := []struct {
		input string
		want  string
	}{
		{`{}`, `{"signatures":{"domain":{"ed25519:1":"K8280/U9SSy9IVtjBuVeLr+HpOB4BQFWbg+UZaADMtTdGYI7Geitb76LTrr5QV/7Xg4ahLwYGYZzuHGZKM5ZAQ"}}}`},
		{`{"one":1,"two":"Two"}`, `{"one":1,"signatures":{"domain":{"ed25519:1":"KqmLSbO39/Bzb0QIYE82zqLwsA+PDzYIpIRA2sRQ4sL53+sN6/fpNSoqE7BP7vBZhG6kYdD13EIMJpvhJI+6Bw"}},"two":"Two"}`},
	}
	for _, tc := range cases {
		got, err := Sign([]byte(tc.input), specSigner, specKeyID, priv)
		if err != nil {
			t.Fatalf("Sign(%s): %v", tc.input, err)
		}
		if string(got) != tc.want {
			t.Fatalf("Sign(%s)\n got %s\nwant %s", tc.input, got, tc.want)
		}
	}
}

func TestVerifySpecVectorsAndRejections(t *testing.T) {
	pub := specKey(t).Public().(ed25519.PublicKey)
	const sig = "K8280/U9SSy9IVtjBuVeLr+HpOB4BQFWbg+UZaADMtTdGYI7Geitb76LTrr5QV/7Xg4ahLwYGYZzuHGZKM5ZAQ"

	if err := Verify([]byte(`{"signatures":{"domain":{"ed25519:1":"`+sig+`"}}}`), specSigner, specKeyID, pub); err != nil {
		t.Fatalf("valid spec vector rejected: %v", err)
	}
	if err := Verify([]byte(`{"unsigned":{"age":5},"signatures":{"domain":{"ed25519:1":"`+sig+`"}}}`), specSigner, specKeyID, pub); err != nil {
		t.Fatalf("unsigned must not be covered by the signature: %v", err)
	}

	rejected := map[string]string{
		"modified payload":  `{"a new key":"a new value","signatures":{"domain":{"ed25519:1":"` + sig + `"}}}`,
		"padded base64":     `{"signatures":{"domain":{"ed25519:1":"` + sig + `=="}}}`,
		"short signature":   `{"signatures":{"domain":{"ed25519:1":"not/a/valid/signature"}}}`,
		"wrong key id":      `{"signatures":{"domain":{"ed25519:2":"` + sig + `"}}}`,
		"no signatures":     `{}`,
		"signatures scalar": `{"signatures":"x"}`,
		"trailing data":     `{"signatures":{"domain":{"ed25519:1":"` + sig + `"}}} {}`,
	}
	for name, input := range rejected {
		if err := Verify([]byte(input), specSigner, specKeyID, pub); err == nil {
			t.Fatalf("%s: expected rejection", name)
		}
	}
}

func TestSignKeepsEarlierSignaturesValid(t *testing.T) {
	origin := specKey(t)
	_, notary, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	signed, err := Sign([]byte(`{"server_name":"origin.example","valid_until_ts":1795299303893,"old_verify_keys":{}}`), "origin.example", "ed25519:o", origin)
	if err != nil {
		t.Fatal(err)
	}
	countersigned, err := Sign(signed, "notary.example", "ed25519:n", notary)
	if err != nil {
		t.Fatal(err)
	}

	if err := Verify(countersigned, "origin.example", "ed25519:o", origin.Public().(ed25519.PublicKey)); err != nil {
		t.Fatalf("origin signature broken by countersigning: %v", err)
	}
	if err := Verify(countersigned, "notary.example", "ed25519:n", notary.Public().(ed25519.PublicKey)); err != nil {
		t.Fatalf("notary signature invalid: %v", err)
	}

	again, err := Sign(countersigned, "notary.example", "ed25519:n", notary)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(countersigned) {
		t.Fatalf("re-signing with the same key must be idempotent")
	}
}

func TestVerifyErrorKinds(t *testing.T) {
	pub := specKey(t).Public().(ed25519.PublicKey)
	if err := Verify([]byte(`{}`), specSigner, specKeyID, pub); !errors.Is(err, ErrMissingSignature) {
		t.Fatalf("want ErrMissingSignature, got %v", err)
	}
	if err := Verify([]byte(`[]`), specSigner, specKeyID, pub); !errors.Is(err, ErrNotObject) {
		t.Fatalf("want ErrNotObject, got %v", err)
	}
	bad := `{"x":1,"signatures":{"domain":{"ed25519:1":"K8280/U9SSy9IVtjBuVeLr+HpOB4BQFWbg+UZaADMtTdGYI7Geitb76LTrr5QV/7Xg4ahLwYGYZzuHGZKM5ZAQ"}}}`
	if err := Verify([]byte(bad), specSigner, specKeyID, pub); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("want ErrBadSignature, got %v", err)
	}
}
