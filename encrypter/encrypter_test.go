package encrypter

import (
	"bytes"
	"testing"
)

const testMAC = "CF:30:16:00:DE:1D"

func TestRoundTrip(t *testing.T) {
	enc, err := NewCubeEncrypter(testMAC, 1)
	if err != nil {
		t.Fatalf("NewCubeEncrypter: %v", err)
	}
	for _, n := range []int{16, 20} {
		data := make([]byte, n)
		for i := range data {
			data[i] = byte(i*7 + 3)
		}
		got := enc.Decrypt(enc.Encrypt(data))
		if !bytes.Equal(got, data) {
			t.Errorf("round trip failed for len %d: got %v want %v", n, got, data)
		}
	}
}

func TestKeyDerivation(t *testing.T) {
	key, iv, err := getKeyAndIV(testMAC)
	if err != nil {
		t.Fatalf("getKeyAndIV: %v", err)
	}
	if len(key) != 16 || len(iv) != 16 {
		t.Fatalf("expected 16-byte key/iv, got %d/%d", len(key), len(iv))
	}
	if key[0] != 50 {
		t.Errorf("key[0] = %d, want 50", key[0])
	}
	if iv[0] != 46 {
		t.Errorf("iv[0] = %d, want 46", iv[0])
	}
}

func TestInvalidMAC(t *testing.T) {
	for _, mac := range []string{"zz", "CF:30", ""} {
		if _, err := NewCubeEncrypter(mac, 1); err == nil {
			t.Errorf("expected error for MAC %q", mac)
		}
	}
}

func TestUnknownCubeType(t *testing.T) {
	if _, err := NewCubeEncrypter(testMAC, 99); err == nil {
		t.Error("expected error for unknown cube type")
	}
}
