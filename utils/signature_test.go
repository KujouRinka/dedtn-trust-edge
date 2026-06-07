package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestSign(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal("cannot create key pair:", err)
	}
	publicKey := &privateKey.PublicKey

	message := []byte("To do or not to do, that is NOT the question.")
	signature, err := SignBytes(message, privateKey)
	if err != nil {
		t.Fatal("cannot sign message:", err)
	}

	if r := VerifySigBytes(message, signature, publicKey); !r {
		t.Fatal("verify signature failed: false value")
	}

	badMsg := []byte("To do or not to do, that is N0T the question.")
	if r := VerifySigBytes(badMsg, signature, publicKey); r {
		t.Fatal("verify signature failed: should return false")
	}
}
