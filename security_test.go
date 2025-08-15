package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestEcdsaKeyFromJwk_RoundTrip(t *testing.T) {
	privateKey, keyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if keyErr != nil {
		t.Fatalf("ecdsa.GenerateKey: %v", keyErr)
	}
	publicKey := privateKey.PublicKey
	publicJwk := publicJwk{
		KeyType: "EC",
		Curve:   "P-256",
		X:       base64.RawURLEncoding.EncodeToString(publicKey.X.Bytes()),
		Y:       base64.RawURLEncoding.EncodeToString(publicKey.Y.Bytes()),
	}
	parsed, parseErr := ecdsaKeyFromJwk(publicJwk)
	if parseErr != nil {
		t.Fatalf("ecdsaKeyFromJwk: %v", parseErr)
	}
	if parsed.X.Cmp(publicKey.X) != 0 || parsed.Y.Cmp(publicKey.Y) != 0 {
		t.Fatalf("parsed key does not match original")
	}
}

func TestVerifyEs256_ValidAndInvalid(t *testing.T) {
	privateKey, keyErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if keyErr != nil {
		t.Fatalf("ecdsa.GenerateKey: %v", keyErr)
	}
	signingInput := []byte("sample input for signature")

	// create r||s (JOSE)
	rInt, sInt, signErr := ecdsa.Sign(rand.Reader, privateKey, sha256sum(signingInput))
	if signErr != nil {
		t.Fatalf("ecdsa.Sign: %v", signErr)
	}
	signatureJose := make([]byte, 64)
	copy(signatureJose[32-len(rInt.Bytes()):32], rInt.Bytes())
	copy(signatureJose[64-len(sInt.Bytes()):64], sInt.Bytes())

	if !verifyEs256(signingInput, signatureJose, &privateKey.PublicKey) {
		t.Fatalf("verifyEs256 should accept a valid signature")
	}

	// flip a bit -> must fail
	signatureJose[0] ^= 0x01
	if verifyEs256(signingInput, signatureJose, &privateKey.PublicKey) {
		t.Fatalf("verifyEs256 should reject a tampered signature")
	}
}

// helper: sha256 over bytes returning digest
func sha256sum(data []byte) []byte {
	sum := sha256.New()
	sum.Write(data)
	return sum.Sum(nil)
}
