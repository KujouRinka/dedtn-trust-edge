package utils

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
)

func SignBytes(b []byte, key *ecdsa.PrivateKey) ([]byte, error) {
	hash := sha256.Sum256(b)
	signature, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func VerifySigBytes(message, b []byte, key *ecdsa.PublicKey) bool {
	hash := sha256.Sum256(message)
	return ecdsa.VerifyASN1(key, hash[:], b)
}
