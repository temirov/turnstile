package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
)

const (
	headerAuthorization             = "Authorization"
	headerDpop                      = "DPoP"
	headerContentType               = "Content-Type"
	headerAccessControlAllowOrigin  = "Access-Control-Allow-Origin"
	headerAccessControlAllowHeaders = "Access-Control-Allow-Headers"
	headerAccessControlAllowMethods = "Access-Control-Allow-Methods"
	headerVary                      = "Vary"

	headerAllowHeadersValue = "Authorization, Content-Type, DPoP"
	headerAllowMethodsValue = "GET, POST, OPTIONS"
	contentTypeJSON         = "application/json"

	audienceApi          = "ets"
	forwardedProtoHeader = "X-Forwarded-Proto"
)

type publicJwk struct {
	KeyType string `json:"kty"`
	Curve   string `json:"crv"`
	X       string `json:"x"`
	Y       string `json:"y"`
}

type dpopHeader struct {
	Type string    `json:"typ"`
	Alg  string    `json:"alg"`
	Jwk  publicJwk `json:"jwk"`
}

type dpopPayload struct {
	HttpMethod string `json:"htm"`
	HttpUri    string `json:"htu"`
	JwtID      string `json:"jti"`
	IssuedAt   int64  `json:"iat"`
}

type confirmation struct {
	JwkThumbprint string `json:"jkt"`
}

type accessClaims struct {
	jwt.RegisteredClaims
	Confirmation confirmation `json:"cnf"`
}

func jwkThumbprint(jwkObject publicJwk) (string, error) {
	canonical := fmt.Sprintf("{\"crv\":\"%s\",\"kty\":\"%s\",\"x\":\"%s\",\"y\":\"%s\"}", jwkObject.Curve, jwkObject.KeyType, jwkObject.X, jwkObject.Y)
	sha256Digest := sha256.Sum256([]byte(canonical))
	return base64.RawURLEncoding.EncodeToString(sha256Digest[:]), nil
}

func parseCompactJws(compactJwsString string) (dpopHeader, dpopPayload, []byte, []byte, error) {
	parts := stringsSplit(compactJwsString, ".")
	if len(parts) != 3 {
		return dpopHeader{}, dpopPayload{}, nil, nil, fmt.Errorf("parts")
	}
	headerBytes, decodeHeaderError := base64.RawURLEncoding.DecodeString(parts[0])
	if decodeHeaderError != nil {
		return dpopHeader{}, dpopPayload{}, nil, nil, decodeHeaderError
	}
	payloadBytes, decodePayloadError := base64.RawURLEncoding.DecodeString(parts[1])
	if decodePayloadError != nil {
		return dpopHeader{}, dpopPayload{}, nil, nil, decodePayloadError
	}
	signatureBytes, decodeSignatureError := base64.RawURLEncoding.DecodeString(parts[2])
	if decodeSignatureError != nil {
		return dpopHeader{}, dpopPayload{}, nil, nil, decodeSignatureError
	}
	var headerObject dpopHeader
	var payloadObject dpopPayload
	if json.Unmarshal(headerBytes, &headerObject) != nil {
		return dpopHeader{}, dpopPayload{}, nil, nil, fmt.Errorf("hdr")
	}
	if json.Unmarshal(payloadBytes, &payloadObject) != nil {
		return dpopHeader{}, dpopPayload{}, nil, nil, fmt.Errorf("pl")
	}
	return headerObject, payloadObject, []byte(parts[0] + "." + parts[1]), signatureBytes, nil
}

func ecdsaKeyFromJwk(jwkObject publicJwk) (*ecdsa.PublicKey, error) {
	if jwkObject.KeyType != "EC" || jwkObject.Curve != "P-256" {
		return nil, fmt.Errorf("unsupported")
	}
	xCoordinateBytes, decodeXError := base64.RawURLEncoding.DecodeString(jwkObject.X)
	if decodeXError != nil {
		return nil, decodeXError
	}
	yCoordinateBytes, decodeYError := base64.RawURLEncoding.DecodeString(jwkObject.Y)
	if decodeYError != nil {
		return nil, decodeYError
	}
	xCoordinate := new(big.Int).SetBytes(xCoordinateBytes)
	yCoordinate := new(big.Int).SetBytes(yCoordinateBytes)
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: xCoordinate, Y: yCoordinate}, nil
}

func verifyEs256(signingInput []byte, joseSignature []byte, publicKey *ecdsa.PublicKey) bool {
	if len(joseSignature) != 64 {
		return false
	}
	rComponent := new(big.Int).SetBytes(joseSignature[:32])
	sComponent := new(big.Int).SetBytes(joseSignature[32:])
	digest := sha256.Sum256(signingInput)
	return ecdsa.Verify(publicKey, digest[:], rComponent, sComponent)
}

func audienceHas(audience jwt.ClaimStrings, expected string) bool {
	for _, value := range audience {
		if value == expected {
			return true
		}
	}
	return false
}
