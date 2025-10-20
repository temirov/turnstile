package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAttachGatewaySdk_ServesEmbeddedModule(t *testing.T) {
	httpMux := http.NewServeMux()
	AttachGatewaySdk(httpMux)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://gateway.example/sdk/tvm.mjs", nil)

	httpMux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected /sdk/tvm.mjs to return 200 OK, got %d", recorder.Code)
	}
	responseBody := recorder.Body.String()
	if !strings.Contains(responseBody, "createGatewayClient") {
		t.Fatalf("expected response body to contain the sdk module contents")
	}
}
