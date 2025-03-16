package handlers

import (
	"fmt"
	"net/http"

	"github.com/runabol/streamabol/stream"
)

// Manifest handles requests for HLS manifest files
// It generates a manifest file for the given source
// and serves it to the client
func Manifest(w http.ResponseWriter, r *http.Request) {
	src := r.URL.Query().Get("src")
	if src == "" {
		http.Error(w, "Missing src parameter", http.StatusBadRequest)
		return
	}

	manifest, err := stream.GetManifest(src)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, manifest)
}
