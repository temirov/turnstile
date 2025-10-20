package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHandleProtectedProxy_InvalidDpopDoesNotMarkReplayCache(t *testing.T) {
	upstreamURL, parseErr := url.Parse("http://upstream.example")
	if parseErr != nil {
		t.Fatalf("url.Parse: %v", parseErr)
	}

	tokenSigningKey := []byte("0123456789abcdef0123456789abcdef")
	tokenID := "test-token-id"

	gatewayConfig := serverConfig{
		AllowedOrigins:     map[string]struct{}{"https://app.example.com": {}},
		RequireTurnstile:   false,
		TokenLifetime:      5 * time.Minute,
		JwtHmacKey:         tokenSigningKey,
		UpstreamBaseURL:    upstreamURL,
		RateLimitPerMinute: 100,
		UpstreamTimeout:    10 * time.Second,
	}

	replayCache := &replayStore{seen: make(map[string]int64)}
	rateLimiter := &windowLimiter{
		windowEnd:    time.Now().Unix() + 60,
		counts:       make(map[string]int),
		perMinuteCap: 100,
	}

	accessToken := issueTestAccessToken(t, tokenSigningKey, tokenID)

	request := httptest.NewRequest(http.MethodPost, "http://gateway.example/api", strings.NewReader(`{"hello":"world"}`))
	request.Header.Set("Origin", "https://app.example.com")
	request.Header.Set("Authorization", "Bearer "+accessToken)

	recorder := httptest.NewRecorder()

	handleProtectedProxy(recorder, request, gatewayConfig, replayCache, rateLimiter, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("expected upstream proxy to be skipped for invalid DPoP")
	}))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized for missing DPoP, got %d", recorder.Code)
	}
	if _, exists := replayCache.seen[tokenID]; exists {
		t.Fatalf("replay cache should not be marked when DPoP validation fails")
	}
}

func issueTestAccessToken(t *testing.T, signingKey []byte, tokenID string) string {
	t.Helper()
	currentTime := time.Now()
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{audienceApi},
			IssuedAt:  jwt.NewNumericDate(currentTime),
			NotBefore: jwt.NewNumericDate(currentTime.Add(-1 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(currentTime.Add(5 * time.Minute)),
			ID:        tokenID,
		},
		Confirmation: confirmation{JwkThumbprint: "test-thumb"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, signErr := token.SignedString(signingKey)
	if signErr != nil {
		t.Fatalf("token.SignedString: %v", signErr)
	}
	return signedToken
}
