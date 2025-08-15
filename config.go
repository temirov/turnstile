package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	envKeyListenAddress          = "LISTEN_ADDR"
	envKeyOriginAllowlist        = "ORIGIN_ALLOWLIST"
	envKeyRequireTurnstile       = "REQUIRE_TURNSTILE"
	envKeyTurnstileSecret        = "TURNSTILE_SECRET_KEY"
	envKeyTokenLifetimeSeconds   = "TOKEN_LIFETIME_SECONDS"
	envKeyJwtHmacKey             = "TVM_JWT_HS256_KEY"
	envKeyUpstreamBaseURL        = "UPSTREAM_BASE_URL"
	envKeyRateLimitPerMinute     = "RATE_LIMIT_PER_MINUTE"
	envKeyUpstreamTimeoutSeconds = "UPSTREAM_TIMEOUT_SECONDS"

	defaultListenAddress          = ":8080"
	defaultTokenLifetimeSeconds   = 300
	defaultRateLimitPerMinute     = 60
	defaultUpstreamTimeoutSeconds = 40
)

type serverConfig struct {
	ListenAddress      string
	AllowedOrigins     map[string]struct{}
	RequireTurnstile   bool
	TurnstileSecretKey string
	TokenLifetime      time.Duration
	JwtHmacKey         []byte
	UpstreamBaseURL    *url.URL
	RateLimitPerMinute int
	UpstreamTimeout    time.Duration
}

func loadConfig() (serverConfig, error) {
	originAllowlistEnv := strings.TrimSpace(os.Getenv(envKeyOriginAllowlist))
	if originAllowlistEnv == "" {
		return serverConfig{}, fmt.Errorf("missing %s", envKeyOriginAllowlist)
	}
	allowedOrigins := make(map[string]struct{})
	for _, originItem := range strings.Split(originAllowlistEnv, ",") {
		trimmed := strings.TrimSpace(originItem)
		if trimmed != "" {
			allowedOrigins[trimmed] = struct{}{}
		}
	}

	listenAddress := os.Getenv(envKeyListenAddress)
	if listenAddress == "" {
		listenAddress = defaultListenAddress
	}

	tokenLifetimeSeconds := defaultTokenLifetimeSeconds
	if lifetimeEnv := strings.TrimSpace(os.Getenv(envKeyTokenLifetimeSeconds)); lifetimeEnv != "" {
		if parsedLifetime, parseLifetimeError := strconv.Atoi(lifetimeEnv); parseLifetimeError == nil && parsedLifetime > 0 {
			tokenLifetimeSeconds = parsedLifetime
		}
	}

	rateLimitPerMinute := defaultRateLimitPerMinute
	if rateEnv := strings.TrimSpace(os.Getenv(envKeyRateLimitPerMinute)); rateEnv != "" {
		if parsedRate, parseRateError := strconv.Atoi(rateEnv); parseRateError == nil && parsedRate > 0 {
			rateLimitPerMinute = parsedRate
		}
	}

	upstreamTimeoutSeconds := defaultUpstreamTimeoutSeconds
	if timeoutEnv := strings.TrimSpace(os.Getenv(envKeyUpstreamTimeoutSeconds)); timeoutEnv != "" {
		if parsedTimeout, parseTimeoutError := strconv.Atoi(timeoutEnv); parseTimeoutError == nil && parsedTimeout > 0 {
			upstreamTimeoutSeconds = parsedTimeout
		}
	}

	jwtHmacSecret := strings.TrimSpace(os.Getenv(envKeyJwtHmacKey))
	if len(jwtHmacSecret) < 16 {
		return serverConfig{}, fmt.Errorf("weak or missing %s", envKeyJwtHmacKey)
	}

	upstreamBaseURLString := strings.TrimSpace(os.Getenv(envKeyUpstreamBaseURL))
	if upstreamBaseURLString == "" {
		return serverConfig{}, fmt.Errorf("missing %s", envKeyUpstreamBaseURL)
	}
	upstreamBaseURL, parseURLError := url.Parse(upstreamBaseURLString)
	if parseURLError != nil {
		return serverConfig{}, fmt.Errorf("bad %s: %v", envKeyUpstreamBaseURL, parseURLError)
	}

	requireTurnstile := strings.EqualFold(strings.TrimSpace(os.Getenv(envKeyRequireTurnstile)), "true")
	turnstileSecretKey := strings.TrimSpace(os.Getenv(envKeyTurnstileSecret))
	if requireTurnstile && turnstileSecretKey == "" {
		return serverConfig{}, fmt.Errorf("REQUIRE_TURNSTILE=true but missing %s", envKeyTurnstileSecret)
	}

	return serverConfig{
		ListenAddress:      listenAddress,
		AllowedOrigins:     allowedOrigins,
		RequireTurnstile:   requireTurnstile,
		TurnstileSecretKey: turnstileSecretKey,
		TokenLifetime:      time.Duration(tokenLifetimeSeconds) * time.Second,
		JwtHmacKey:         []byte(jwtHmacSecret),
		UpstreamBaseURL:    upstreamBaseURL,
		RateLimitPerMinute: rateLimitPerMinute,
		UpstreamTimeout:    time.Duration(upstreamTimeoutSeconds) * time.Second,
	}, nil
}
