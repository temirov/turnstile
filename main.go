package main

import (
	"log"
	"net/http"
)

func main() {
	gatewayConfig, loadConfigError := loadConfig()
	if loadConfigError != nil {
		log.Fatalf("config error: %v", loadConfigError)
	}
	httpServer := newHTTPServer(gatewayConfig)
	log.Printf("turnstile listening on %s", gatewayConfig.ListenAddress)
	if serveError := httpServer.ListenAndServe(); serveError != nil && serveError != http.ErrServerClosed {
		log.Fatalf("server error: %v", serveError)
	}
}
