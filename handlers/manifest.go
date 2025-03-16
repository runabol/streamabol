package handlers

import (
	"fmt"
	"net/http"

	"github.com/runabol/streamabol/stream"
)

// Manifest generates HLS playlists for the given video source
// and returns the master playlist
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
