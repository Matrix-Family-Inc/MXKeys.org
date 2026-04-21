/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package keyprovider

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// At-rest encryption format for the file-backed signing key.
//
// The file layout is:
//
//	offset  size  field
//	0       8     magic       "MXKENC01"
//	8       4     iterations  PBKDF2 iteration count, uint32 big-endian
//	12      16    salt        random, PBKDF2 salt
//	28      12    nonce       random, AES-256-GCM nonce
//	40      N+16  ciphertext  AES-256-GCM(plaintext || auth_tag)
//
// The 40-byte header (magic || iterations || salt || nonce) is fed into
// AES-256-GCM as Additional Authenticated Data so that a tampered header
// causes decryption to fail.
//
// PBKDF2-HMAC-SHA256 is used as the KDF. 600 000 iterations is the OWASP
// 2023 recommendation for SHA-256 and is safely above what is needed to
// slow a hardware adversary on a consumer GPU. The iteration count is
// stored in the header so the parameter can be rotated upward on next
// write without breaking reads of older files.
//
// The magic string doubles as format detection: a file that does not
// begin with MXKENC01 is treated as legacy plaintext (backward-compat
// for operators migrating from the pre-encryption build).

const (
	encMagic            = "MXKENC01"
	encMagicLen         = 8
	encIterationsOffset = 8
	encSaltOffset       = 12
	encSaltLen          = 16
	encNonceOffset      = 28
	encNonceLen         = 12
	encHeaderLen        = 40

	// productionPBKDF2Iterations is the OWASP-2023 recommendation for
	// PBKDF2-HMAC-SHA256 and is the default burned into every
	// newly-written encrypted file in production builds.
	productionPBKDF2Iterations = 600_000

	// minPBKDF2Iterations is the absolute floor the decoder will
	// accept at read time and the encoder at write time. Anything
	// below this is treated as corruption or an attacker trying
	// to weaken the KDF.
	minPBKDF2Iterations = 10_000

	// maxPBKDF2Iterations is an upper bound that lets the encoder
	// store the count in a 4-byte header field without concern
	// for a wrap-around on unusual platforms. 10^8 is well above
	// the tolerable wall time for human-interactive decryption
	// and orders of magnitude above the current default.
	maxPBKDF2Iterations = 100_000_000

	// aesKeyLen is the KEK length for AES-256-GCM.
	aesKeyLen = 32
)

// pbkdf2Iterations is the iteration count used by newly-written files.
// It is a var (not a const) so that tests can lower it via a package
// level override to keep the test suite fast without sacrificing real
// KDF/AEAD coverage. Production code never writes to this; the
// TestProductionEncryptKeyUsesDefaultIterations test pins the initial
// value to productionPBKDF2Iterations.
var pbkdf2Iterations = productionPBKDF2Iterations

// ErrBadMagic indicates the payload is not MXKENC01-formatted. Callers treat
// this as "file is plaintext" for legacy backwards compatibility.
var ErrBadMagic = errors.New("keyprovider: payload is not MXKENC01-encrypted")

// ErrDecryptFailed wraps AES-GCM authentication failures. Common causes:
// wrong passphrase, tampered ciphertext, or a file whose header was
// damaged.
var ErrDecryptFailed = errors.New("keyprovider: decryption failed (wrong passphrase or tampered file)")

// encryptKey seals plaintext with a KEK derived from the passphrase and a
// fresh random salt + nonce using the default PBKDF2 iteration count.
func encryptKey(plaintext []byte, passphrase []byte) ([]byte, error) {
	return encryptKeyWithIterations(plaintext, passphrase, pbkdf2Iterations)
}

// encryptKeyWithIterations is the same as encryptKey but lets callers
// pick the PBKDF2 iteration count. Production code uses encryptKey
// (which hard-codes the OWASP-recommended default); tests use this
// entry point with a low value to keep the suite fast without
// sacrificing coverage of the format layer.
func encryptKeyWithIterations(plaintext, passphrase []byte, iterations int) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, errors.New("keyprovider: empty passphrase")
	}
	if iterations < minPBKDF2Iterations {
		return nil, fmt.Errorf("keyprovider: iterations %d below minimum %d", iterations, minPBKDF2Iterations)
	}

	salt := make([]byte, encSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("keyprovider: salt: %w", err)
	}
	nonce := make([]byte, encNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("keyprovider: nonce: %w", err)
	}

	kek := pbkdf2SHA256(passphrase, salt, iterations, aesKeyLen)
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: gcm: %w", err)
	}

	if iterations < 0 || iterations > int(maxPBKDF2Iterations) {
		return nil, fmt.Errorf("keyprovider: iterations %d out of range [%d, %d]",
			iterations, minPBKDF2Iterations, maxPBKDF2Iterations)
	}
	header := make([]byte, encHeaderLen)
	copy(header[0:encMagicLen], encMagic)
	binary.BigEndian.PutUint32(header[encIterationsOffset:encSaltOffset], uint32(iterations))
	copy(header[encSaltOffset:encNonceOffset], salt)
	copy(header[encNonceOffset:encHeaderLen], nonce)

	// AAD binds the header to the ciphertext; any mutation of the
	// stored parameters causes Open to fail closed.
	ciphertext := aead.Seal(nil, nonce, plaintext, header)

	out := make([]byte, 0, encHeaderLen+len(ciphertext))
	out = append(out, header...)
	out = append(out, ciphertext...)
	return out, nil
}

// decryptKey opens a MXKENC01 envelope. Returns ErrBadMagic for unknown
// prefixes (the caller treats that as "legacy plaintext") and
// ErrDecryptFailed for authentication failures.
func decryptKey(blob []byte, passphrase []byte) ([]byte, error) {
	if len(blob) < encHeaderLen {
		return nil, ErrBadMagic
	}
	if string(blob[0:encMagicLen]) != encMagic {
		return nil, ErrBadMagic
	}
	if len(passphrase) == 0 {
		return nil, errors.New("keyprovider: file is encrypted but no passphrase provided (set MXKEYS_KEY_PASSPHRASE)")
	}

	iterations := binary.BigEndian.Uint32(blob[encIterationsOffset:encSaltOffset])
	if iterations < minPBKDF2Iterations {
		return nil, fmt.Errorf("keyprovider: implausibly low iteration count %d", iterations)
	}
	salt := blob[encSaltOffset:encNonceOffset]
	nonce := blob[encNonceOffset:encHeaderLen]
	ciphertext := blob[encHeaderLen:]

	kek := pbkdf2SHA256(passphrase, salt, int(iterations), aesKeyLen)
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: gcm: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, blob[0:encHeaderLen])
	if err != nil {
		return nil, ErrDecryptFailed
	}
	return plaintext, nil
}

// hasEncryptedMagic is a tiny helper for readers that want to dispatch
// between plaintext and encrypted without attempting decryption.
func hasEncryptedMagic(blob []byte) bool {
	return len(blob) >= encMagicLen && string(blob[0:encMagicLen]) == encMagic
}

// pbkdf2SHA256 is a minimal RFC 2898 implementation on top of crypto/hmac
// and crypto/sha256. Deliberately dependency-free: the Go standard
// library ships every primitive PBKDF2 needs, and external x/crypto
// would introduce a dependency just for this ~20 LOC function.
//
// pbkdf2SHA256 returns exactly keyLen bytes; iter must be >= 1.
func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	if iter < 1 {
		iter = 1
	}
	hashLen := sha256.Size
	blocks := (keyLen + hashLen - 1) / hashLen

	out := make([]byte, 0, blocks*hashLen)
	u := make([]byte, hashLen)
	prev := make([]byte, hashLen)

	for i := 1; i <= blocks; i++ {
		// U1 = HMAC(password, salt || INT(i))
		h := hmac.New(sha256.New, password)
		h.Write(salt)
		var ib [4]byte
		binary.BigEndian.PutUint32(ib[:], uint32(i))
		h.Write(ib[:])
		sum := h.Sum(nil)
		copy(u, sum)
		copy(prev, sum)

		// U_j = HMAC(password, U_{j-1}); block_i = U1 XOR U2 XOR ... XOR U_iter
		for j := 2; j <= iter; j++ {
			h := hmac.New(sha256.New, password)
			h.Write(prev)
			sum := h.Sum(nil)
			copy(prev, sum)
			for k := 0; k < hashLen; k++ {
				u[k] ^= sum[k]
			}
		}
		out = append(out, u...)
	}
	return out[:keyLen]
}
