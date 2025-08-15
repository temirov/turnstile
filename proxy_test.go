package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExpectedHtu_UsesXForwardedProto(t *testing.T) {
	request := httptest.NewRequest("POST", "http://api.example.com/api?x=1", nil)
	request.Header.Set(forwardedProtoHeader, "https")
	got := expectedHtu(request)
	want := "https://api.example.com/api?x=1"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestCheckOrigin_AllowsExactOriginAndSetsCors(t *testing.T) {
	allowed := map[string]struct{}{"https://app.example.com": {}}
	rec := httptest.NewRecorder()
	request := httptest.NewRequest("OPTIONS", "http://api.example.com/tvm/issue", nil)
	request.Header.Set("Origin", "https://app.example.com")

	if !checkOrigin(rec, request, allowed) {
		t.Fatalf("expected origin to be allowed")
	}
	if rec.Header().Get(headerAccessControlAllowOrigin) != "https://app.example.com" {
		t.Fatalf("missing CORS allow-origin header")
	}
}

func TestCheckOrigin_RejectsUnknownOrigin(t *testing.T) {
	allowed := map[string]struct{}{"https://app.example.com": {}}
	rec := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "http://api.example.com/api", nil)
	request.Header.Set("Origin", "https://evil.example.com")

	if checkOrigin(rec, request, allowed) {
		t.Fatalf("expected origin to be rejected")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
