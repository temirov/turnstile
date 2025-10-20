package main

import (
	"embed"
	"net/http"
)

//go:embed sdk/tvm.mjs
var embeddedSdkFiles embed.FS

func AttachGatewaySdk(httpMux *http.ServeMux) {
	httpMux.Handle("/sdk/", http.FileServer(http.FS(embeddedSdkFiles)))
}
