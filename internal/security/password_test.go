package security

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	const pw = "correct horse battery staple"
	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == pw {
		t.Fatal("hash must not equal the plaintext")
	}

	ok, err := VerifyPassword(pw, hash)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}

	ok, err = VerifyPassword("wrong password", hash)
	if err != nil {
		t.Fatalf("VerifyPassword(wrong): %v", err)
	}
	if ok {
		t.Fatal("expected wrong password to fail")
	}
}

func TestHashPasswordEmpty(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestVerifyPasswordBadHash(t *testing.T) {
	if _, err := VerifyPassword("x", "not-a-valid-phc-string"); err == nil {
		t.Fatal("expected error for malformed hash")
	}
}

func TestUniquenessOfHashes(t *testing.T) {
	// Same password must produce different hashes (random salt).
	a, _ := HashPassword("samepw")
	b, _ := HashPassword("samepw")
	if a == b {
		t.Fatal("hashes of the same password should differ due to salt")
	}
}
