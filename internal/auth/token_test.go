package auth

import "testing"

func TestNewTokenUniqueAndHashed(t *testing.T) {
	raw1, hash1, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	raw2, hash2, _ := NewToken()
	if raw1 == raw2 {
		t.Fatal("Tokens sollten zufällig/eindeutig sein")
	}
	if hash1 == raw1 {
		t.Fatal("Hash darf nicht dem Rohtoken entsprechen")
	}
	if hash1 == hash2 {
		t.Fatal("Hashes verschiedener Tokens sollten sich unterscheiden")
	}
	if HashToken(raw1) != hash1 {
		t.Fatal("HashToken ist nicht deterministisch")
	}
}
