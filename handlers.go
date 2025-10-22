package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	dpopReplayWindow     = 5 * time.Minute
	dpopAllowedClockSkew = 5 * time.Second
)

type tokenIssueRequest struct {
	DpopPublicJwk publicJwk `json:"dpopPublicJwk"`
}

type tokenIssueResponse struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int    `json:"expiresIn"`
}

func handleTokenIssue(httpResponseWriter http.ResponseWriter, httpRequest *http.Request, gatewayConfig serverConfig) {
	if !checkOrigin(httpResponseWriter, httpRequest, gatewayConfig.AllowedOrigins) {
		return
	}
	if httpRequest.Method == http.MethodOptions {
		httpResponseWriter.WriteHeader(http.StatusNoContent)
		return
	}
	if httpRequest.Method != http.MethodPost && httpRequest.Method != http.MethodGet {
		httpErrorJSON(httpResponseWriter, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	requestBodyBytes, readBodyError := io.ReadAll(httpRequest.Body)
	if readBodyError != nil {
		httpErrorJSON(httpResponseWriter, http.StatusBadRequest, "bad_request_body")
		return
	}
	defer httpRequest.Body.Close()

	var tokenRequest tokenIssueRequest
	if unmarshalError := json.Unmarshal(requestBodyBytes, &tokenRequest); unmarshalError != nil {
		httpErrorJSON(httpResponseWriter, http.StatusBadRequest, "invalid_json")
		return
	}

	if tokenRequest.DpopPublicJwk.KeyType != "EC" || tokenRequest.DpopPublicJwk.Curve != "P-256" {
		httpErrorJSON(httpResponseWriter, http.StatusBadRequest, "unsupported_jwk")
		return
	}
	jwkThumbprintValue, thumbprintError := jwkThumbprint(tokenRequest.DpopPublicJwk)
	if thumbprintError != nil {
		httpErrorJSON(httpResponseWriter, http.StatusBadRequest, "bad_jwk_thumbprint")
		return
	}

	currentTime := time.Now()
	tokenExpiration := currentTime.Add(gatewayConfig.TokenLifetime)
	tokenID := fmt.Sprintf("%d-%d", currentTime.UnixNano(), os.Getpid())

	accessTokenClaims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audienceApi},
			IssuedAt:  jwt.NewNumericDate(currentTime),
			NotBefore: jwt.NewNumericDate(currentTime.Add(-1 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(tokenExpiration),
			ID:        tokenID,
		},
		Confirmation: confirmation{JwkThumbprint: jwkThumbprintValue},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	signedToken, signError := jwtToken.SignedString(gatewayConfig.JwtHmacKey)
	if signError != nil {
		httpErrorJSON(httpResponseWriter, http.StatusInternalServerError, "sign_error")
		return
	}

	tokenResponse := tokenIssueResponse{AccessToken: signedToken, ExpiresIn: int(gatewayConfig.TokenLifetime.Seconds())}
	httpResponseWriter.Header().Set(headerContentType, contentTypeJSON)
	_ = json.NewEncoder(httpResponseWriter).Encode(tokenResponse)
}

func handleProtectedProxy(httpResponseWriter http.ResponseWriter, httpRequest *http.Request, gatewayConfig serverConfig, replayCache *replayStore, rateLimiter *windowLimiter, upstreamProxy http.Handler) {
	if !checkOrigin(httpResponseWriter, httpRequest, gatewayConfig.AllowedOrigins) {
		return
	}
	if httpRequest.Method == http.MethodOptions {
		httpResponseWriter.WriteHeader(http.StatusNoContent)
		return
	}
	if httpRequest.Method != http.MethodPost && httpRequest.Method != http.MethodGet {
		httpErrorJSON(httpResponseWriter, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if !rateLimiter.allow(rateKey(httpRequest.RemoteAddr, httpRequest.Header.Get("Origin"))) {
		httpErrorJSON(httpResponseWriter, http.StatusTooManyRequests, "rate_limited")
		return
	}

	bearerAccessToken := parseBearer(httpRequest.Header.Get(headerAuthorization))
	if bearerAccessToken == "" {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "missing_bearer")
		return
	}

	var parsedClaims accessClaims
	parsedJWT, parseTokenError := jwt.ParseWithClaims(bearerAccessToken, &parsedClaims, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected_jwt_alg")
		}
		return gatewayConfig.JwtHmacKey, nil
	})
	if parseTokenError != nil || !parsedJWT.Valid {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "invalid_token")
		return
	}

	currentTime := time.Now()
	if !audienceHas(parsedClaims.Audience, audienceApi) ||
		parsedClaims.ExpiresAt == nil || currentTime.After(parsedClaims.ExpiresAt.Time) ||
		(parsedClaims.NotBefore != nil && currentTime.Before(parsedClaims.NotBefore.Time)) {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "bad_claims")
		return
	}

	if parsedClaims.ID == "" {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "replay")
		return
	}

	tokenExpirationTime := parsedClaims.ExpiresAt.Time

	rawDpopHeader := stringsTrimSpace(httpRequest.Header.Get(headerDpop))
	if rawDpopHeader == "" {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "missing_dpop")
		return
	}

	dpopHeaderObject, dpopPayloadObject, dpopSigningInput, dpopSignatureBytes, parseDpopError := parseCompactJws(rawDpopHeader)
	if parseDpopError != nil {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "bad_dpop")
		return
	}
	if !stringsEqualFold(dpopHeaderObject.Type, "dpop+jwt") || !stringsEqualFold(dpopHeaderObject.Alg, "ES256") {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "bad_dpop_header")
		return
	}

	publicKeyFromJwk, ecdsaBuildError := ecdsaKeyFromJwk(dpopHeaderObject.Jwk)
	if ecdsaBuildError != nil {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "bad_dpop_key")
		return
	}
	if !verifyEs256(dpopSigningInput, dpopSignatureBytes, publicKeyFromJwk) {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "bad_dpop_sig")
		return
	}

	jwkThumbprintComputed, thumbError := jwkThumbprint(dpopHeaderObject.Jwk)
	if thumbError != nil || jwkThumbprintComputed != parsedClaims.Confirmation.JwkThumbprint {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "cnf_mismatch")
		return
	}

	if dpopPayloadObject.HttpMethod != httpRequest.Method {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "htm_mismatch")
		return
	}
	if dpopPayloadObject.HttpUri != expectedHtu(httpRequest) {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "htu_mismatch")
		return
	}

	if dpopPayloadObject.JwtID == "" {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "missing_dpop_jti")
		return
	}

	if dpopPayloadObject.IssuedAt == 0 {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "missing_dpop_iat")
		return
	}

	now := time.Now()
	issuedAtTime := time.Unix(dpopPayloadObject.IssuedAt, 0)
	if issuedAtTime.After(now.Add(dpopAllowedClockSkew)) {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "dpop_iat_in_future")
		return
	}
	if issuedAtTime.Before(now.Add(-1 * dpopReplayWindow)) {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "dpop_iat_too_old")
		return
	}

	replayExpiresAt := issuedAtTime.Add(dpopReplayWindow)
	if replayExpiresAt.After(tokenExpirationTime) {
		replayExpiresAt = tokenExpirationTime
	}

	if !replayCache.mark(dpopPayloadObject.JwtID, replayExpiresAt) {
		httpErrorJSON(httpResponseWriter, http.StatusUnauthorized, "replay")
		return
	}

	upstreamContext, cancelUpstream := context.WithTimeout(httpRequest.Context(), gatewayConfig.UpstreamTimeout)
	defer cancelUpstream()

	httpRequest = httpRequest.WithContext(upstreamContext)
	upstreamProxy.ServeHTTP(httpResponseWriter, httpRequest)
}

func handleHealth(httpResponseWriter http.ResponseWriter, _ *http.Request) {
	httpResponseWriter.Header().Set(headerContentType, contentTypeJSON)
	httpResponseWriter.WriteHeader(http.StatusOK)
	_, _ = httpResponseWriter.Write([]byte("{\"status\":\"ok\"}"))
}
