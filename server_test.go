package main

import (
	"net/http"
	"net/url"
	"testing"
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
	request, requestErr := http.NewRequest(http.MethodGet, "http://gateway.example/api?prompt=hi", nil)
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
	request, requestErr := http.NewRequest(http.MethodGet, "http://gateway.example/api?prompt=hi&key=user", nil)
	if requestErr != nil {
		t.Fatalf("http.NewRequest: %v", requestErr)
	}

	reverseProxy.Director(request)
	if request.URL.Query().Get("key") != "super-secret" {
		t.Fatalf("expected injected key to override existing value")
	}
}
