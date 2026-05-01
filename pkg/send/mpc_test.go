package send_test

import (
	"bytes"
	"testing"

	"github.com/afterdarksys/aftermail/pkg/send"
)

func TestShamirRoundTrip(t *testing.T) {
	secret := []byte("super-secret-group-key-32-bytes!")

	for _, tc := range []struct{ n, m int }{{3, 2}, {5, 3}, {5, 5}, {2, 2}} {
		shares, err := send.Split(secret, tc.n, tc.m)
		if err != nil {
			t.Fatalf("Split(%d,%d): %v", tc.n, tc.m, err)
		}
		if len(shares) != tc.n {
			t.Fatalf("expected %d shares, got %d", tc.n, len(shares))
		}

		// Reconstruct from exactly m shares.
		recovered, err := send.Combine(shares[:tc.m])
		if err != nil {
			t.Fatalf("Combine(%d,%d): %v", tc.n, tc.m, err)
		}
		if !bytes.Equal(secret, recovered) {
			t.Fatalf("(%d,%d): recovered %q, want %q", tc.n, tc.m, recovered, secret)
		}
	}
}

func TestShamirInsufficientShares(t *testing.T) {
	secret := []byte("test-secret")
	shares, _ := send.Split(secret, 5, 3)

	// Only 2 shares — should produce garbage (not an error, but wrong value).
	recovered, err := send.Combine(shares[:2])
	if err != nil {
		t.Fatal("Combine should not error with 2 shares")
	}
	if bytes.Equal(secret, recovered) {
		t.Fatal("should not recover secret from fewer than threshold shares")
	}
}

func TestGCMRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key { key[i] = byte(i) }

	plaintext := []byte("hello encrypted world")
	ct, err := send.EncryptGCM(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := send.DecryptGCM(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plaintext, recovered) {
		t.Fatalf("gcm mismatch: got %q", recovered)
	}
}

func TestGroupThreadRoundTrip(t *testing.T) {
	body := "This message requires 2 of 3 participants to decrypt."
	thread, err := send.NewGroupThread("test-thread-1", body, 3, 2)
	if err != nil {
		t.Fatal(err)
	}

	// Decrypt with shares 0 and 1 (any 2 of 3 should work).
	plaintext, err := thread.Decrypt(thread.Shares[:2])
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != body {
		t.Fatalf("decrypted %q, want %q", plaintext, body)
	}

	// Also try shares 1 and 2.
	plaintext2, err := thread.Decrypt(thread.Shares[1:3])
	if err != nil {
		t.Fatal(err)
	}
	if plaintext2 != body {
		t.Fatalf("decrypted %q, want %q", plaintext2, body)
	}
}

func TestGroupThreadBadDigest(t *testing.T) {
	thread, _ := send.NewGroupThread("test-thread-2", "secret", 2, 2)
	thread.KeyDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	_, err := thread.Decrypt(thread.Shares)
	if err == nil {
		t.Fatal("expected error on digest mismatch")
	}
}
