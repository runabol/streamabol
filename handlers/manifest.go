package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

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

// Playlist handles requests for HLS playlist files
func Playlist(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/playlist"), "/v0.m3u8")
	playlist, error := stream.GetPlaylist(id)
	if error != nil {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	http.ServeFile(w, r, playlist)
}

// Segment handles requests for HLS segment files
func Segment(w http.ResponseWriter, r *http.Request) {
	re := regexp.MustCompile(`/segment/([0-9a-f]{32})/v0/(\d+)\.ts`)
	matches := re.FindStringSubmatch(r.URL.Path)
	if len(matches) < 3 {
		http.Error(w, "Invalid segment path", http.StatusBadRequest)
		return
	}
	playlistID := matches[1]
	segNum, err := strconv.Atoi(matches[2])
	if err != nil {
		http.Error(w, "Invalid segment number", http.StatusBadRequest)
		return
	}
	seg, err := stream.GetSegment(playlistID, segNum)
	if err != nil {
		http.Error(w, "Failed to get segment", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "video/mp2t")
	http.ServeFile(w, r, seg)
}
