package main

import (
	"fmt"
	"net/http"

	"github.com/timskillet/distributed-filestore/internal/api"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/init-upload", api.HandleInitUpload)
	mux.HandleFunc("/finalize-upload", api.HandleFinalizeUpload)
	mux.HandleFunc("/download-plan", api.HandleDownloadPlan)

	fmt.Println("Starting mock API server on port 8080")
	http.ListenAndServe(":8080", mux)
}
