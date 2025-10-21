package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewReverseProxy_AppendsSecretWhenMissing(t *testing.T) {
	upstreamURL, parseErr := url.Parse("http://upstream.example:8080")
	if parseErr != nil {
		t.Fatalf("url.Parse: %v", parseErr)
	}
	config := serverConfig{
		UpstreamBaseURL:   upstreamURL,
		UpstreamSecretKey: "super-secret",
	}

	reverseProxy := newReverseProxy(config)
	request, requestErr := http.NewRequest(http.MethodGet, "http://ets.example/api?prompt=hi", nil)
	if requestErr != nil {
		t.Fatalf("http.NewRequest: %v", requestErr)
	}

	reverseProxy.Director(request)

	query := request.URL.Query()
	if query.Get("key") != "super-secret" {
		t.Fatalf("expected injected key query parameter")
	}
	if query.Get("prompt") != "hi" {
		t.Fatalf("expected existing query parameters to remain")
	}
}

func TestNewReverseProxy_OverridesExistingSecret(t *testing.T) {
	upstreamURL, parseErr := url.Parse("http://upstream.example:8080")
	if parseErr != nil {
		t.Fatalf("url.Parse: %v", parseErr)
	}
	config := serverConfig{
		UpstreamBaseURL:   upstreamURL,
		UpstreamSecretKey: "super-secret",
	}

	reverseProxy := newReverseProxy(config)
	request, requestErr := http.NewRequest(http.MethodGet, "http://ets.example/api?prompt=hi&key=user", nil)
	if requestErr != nil {
		t.Fatalf("http.NewRequest: %v", requestErr)
	}

	reverseProxy.Director(request)
	if request.URL.Query().Get("key") != "super-secret" {
		t.Fatalf("expected injected key to override existing value")
	}
}

func TestNewHTTPServer_RoutesApiSubpaths(t *testing.T) {
	upstreamURL, parseErr := url.Parse("http://upstream.example:8080")
	if parseErr != nil {
		t.Fatalf("url.Parse: %v", parseErr)
	}

	config := serverConfig{
		ListenAddress:      ":8080",
		AllowedOrigins:     map[string]struct{}{"https://app.example.com": {}},
		TokenLifetime:      5 * time.Minute,
		JwtHmacKey:         []byte("0123456789abcdef0123456789abcdef"),
		UpstreamBaseURL:    upstreamURL,
		UpstreamSecretKey:  "",
		RateLimitPerMinute: 60,
		UpstreamTimeout:    10 * time.Second,
	}

	httpServer := newHTTPServer(config)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://ets.example/api/search", strings.NewReader(`{"prompt":"hi"}`))
	request.Header.Set("Origin", "https://app.example.com")

	httpServer.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected /api/subpath to reach proxy handler and return 401 for missing bearer, got %d", recorder.Code)
	}
}
