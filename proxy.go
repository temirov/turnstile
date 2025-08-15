package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func expectedHtu(httpRequest *http.Request) string {
	protoHeaderValue := strings.ToLower(strings.TrimSpace(httpRequest.Header.Get(forwardedProtoHeader)))
	scheme := "http"
	if protoHeaderValue == "https" || httpRequest.TLS != nil {
		scheme = "https"
	}
	hostHeader := httpRequest.Host
	pathAndQuery := httpRequest.URL.Path
	if httpRequest.URL.RawQuery != "" {
		pathAndQuery += "?" + httpRequest.URL.RawQuery
	}
	return scheme + "://" + hostHeader + pathAndQuery
}

func parseBearer(authorizationHeaderValue string) string {
	if !strings.HasPrefix(authorizationHeaderValue, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authorizationHeaderValue, "Bearer "))
}

func httpErrorJSON(httpResponseWriter http.ResponseWriter, statusCode int, errorCode string) {
	httpResponseWriter.Header().Set(headerContentType, contentTypeJSON)
	httpResponseWriter.WriteHeader(statusCode)
	_, _ = httpResponseWriter.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", errorCode)))
}

func checkOrigin(httpResponseWriter http.ResponseWriter, httpRequest *http.Request, allowedOrigins map[string]struct{}) bool {
	originHeader := httpRequest.Header.Get("Origin")
	if _, isAllowed := allowedOrigins[originHeader]; !isAllowed {
		httpErrorJSON(httpResponseWriter, http.StatusForbidden, "origin_not_allowed")
		return false
	}
	httpResponseWriter.Header().Set(headerAccessControlAllowOrigin, originHeader)
	httpResponseWriter.Header().Set(headerVary, "Origin")
	httpResponseWriter.Header().Set(headerAccessControlAllowHeaders, headerAllowHeadersValue)
	httpResponseWriter.Header().Set(headerAccessControlAllowMethods, headerAllowMethodsValue)
	return true
}

func rateKey(remoteAddress string, originHeader string) string {
	hostPart, _, splitError := net.SplitHostPort(remoteAddress)
	if splitError != nil {
		hostPart = remoteAddress
	}
	return originHeader + "|" + hostPart
}
