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
	envKeyRequireETS             = "REQUIRE_ETS"
	envKeyETSSecret              = "ETS_SECRET_KEY"
	envKeyTokenLifetimeSeconds   = "TOKEN_LIFETIME_SECONDS"
	envKeyJwtHmacKey             = "TVM_JWT_HS256_KEY"
	envKeyUpstreamBaseURL        = "UPSTREAM_BASE_URL"
	envKeyUpstreamServiceSecret  = "UPSTREAM_SERVICE_SECRET"
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
	RequireETS         bool
	EtsSecretKey       string
	TokenLifetime      time.Duration
	JwtHmacKey         []byte
	UpstreamBaseURL    *url.URL
	UpstreamSecretKey  string
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

	requireETS := strings.EqualFold(strings.TrimSpace(os.Getenv(envKeyRequireETS)), "true")
	etsSecretKey := strings.TrimSpace(os.Getenv(envKeyETSSecret))
	if requireETS && etsSecretKey == "" {
		return serverConfig{}, fmt.Errorf("REQUIRE_ETS=true but missing %s", envKeyETSSecret)
	}

	upstreamServiceSecret := strings.TrimSpace(os.Getenv(envKeyUpstreamServiceSecret))

	return serverConfig{
		ListenAddress:      listenAddress,
		AllowedOrigins:     allowedOrigins,
		RequireETS:         requireETS,
		EtsSecretKey:       etsSecretKey,
		TokenLifetime:      time.Duration(tokenLifetimeSeconds) * time.Second,
		JwtHmacKey:         []byte(jwtHmacSecret),
		UpstreamBaseURL:    upstreamBaseURL,
		UpstreamSecretKey:  upstreamServiceSecret,
		RateLimitPerMinute: rateLimitPerMinute,
		UpstreamTimeout:    time.Duration(upstreamTimeoutSeconds) * time.Second,
	}, nil
}
