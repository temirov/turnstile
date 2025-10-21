package main

import (
	"strings"
	"testing"
)

func TestLoadConfigRequiresEtsSecret(t *testing.T) {
	t.Setenv("ORIGIN_ALLOWLIST", "https://app.example.com")
	t.Setenv("TVM_JWT_HS256_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("UPSTREAM_BASE_URL", "https://upstream.example.com")
	_, loadError := loadConfig()
	if loadError == nil {
		t.Fatalf("expected missing ETS secret to return an error")
	}
	if !strings.Contains(loadError.Error(), "ETS_SECRET_KEY") {
		t.Fatalf("expected error mentioning ETS_SECRET_KEY, got %v", loadError)
	}
}

func TestLoadConfigLoadsEtsSecret(t *testing.T) {
	t.Setenv("ORIGIN_ALLOWLIST", "https://app.example.com")
	t.Setenv("TVM_JWT_HS256_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("UPSTREAM_BASE_URL", "https://upstream.example.com")
	t.Setenv("ETS_SECRET_KEY", "  super-secret  ")
	config, loadError := loadConfig()
	if loadError != nil {
		t.Fatalf("loadConfig returned error: %v", loadError)
	}
	if config.EtsSecretKey != "super-secret" {
		t.Fatalf("expected trimmed secret, got %q", config.EtsSecretKey)
	}
}
