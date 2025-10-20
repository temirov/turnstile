package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

func newReverseProxy(gatewayConfig serverConfig) *httputil.ReverseProxy {
	reverseProxy := httputil.NewSingleHostReverseProxy(gatewayConfig.UpstreamBaseURL)
	originalDirector := reverseProxy.Director
	reverseProxy.Director = func(incomingRequest *http.Request) {
		originalDirector(incomingRequest)
		if gatewayConfig.UpstreamSecretKey != "" {
			queryValues := incomingRequest.URL.Query()
			queryValues.Set("key", gatewayConfig.UpstreamSecretKey)
			incomingRequest.URL.RawQuery = queryValues.Encode()
		}
	}
	reverseProxy.ErrorHandler = func(httpResponseWriter http.ResponseWriter, httpRequest *http.Request, proxyError error) {
		log.Printf("reverse proxy error: %v", proxyError)
		httpErrorJSON(httpResponseWriter, http.StatusBadGateway, "upstream_error")
	}
	return reverseProxy
}

func newHTTPServer(gatewayConfig serverConfig) *http.Server {
	// reverse proxy (base origin only, no path)
	upstreamReverseProxy := newReverseProxy(gatewayConfig)

	replayCacheStore := &replayStore{seen: make(map[string]int64)}
	rateLimiterWindow := &windowLimiter{
		windowEnd:    timeNow().Unix() + 60,
		counts:       make(map[string]int),
		perMinuteCap: gatewayConfig.RateLimitPerMinute,
	}

	httpServerMux := http.NewServeMux()
	AttachGatewaySdk(httpServerMux)
	httpServerMux.HandleFunc("/tvm/issue", func(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
		handleTokenIssue(httpResponseWriter, httpRequest, gatewayConfig)
	})
	httpServerMux.HandleFunc("/api", func(httpResponseWriter http.ResponseWriter, httpRequest *http.Request) {
		handleProtectedProxy(httpResponseWriter, httpRequest, gatewayConfig, replayCacheStore, rateLimiterWindow, upstreamReverseProxy)
	})

	return &http.Server{
		Addr:              gatewayConfig.ListenAddress,
		Handler:           httpServerMux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// tiny indirection to ease testing (can be stubbed)
var timeNow = func() time.Time { return time.Now() }
