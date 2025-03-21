package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type Server struct {
	Address   string
	SecretKey string
	BaseDir   string
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/manifest.m3u8", s.Manifest)
	mux.HandleFunc("/playlist/", s.Playlist)
	mux.HandleFunc("/segment/", s.Segment)
	handler := CORSMiddleware(
		NewHMACMiddleware(s.SecretKey).Handle(LoggerMiddleware(mux)),
	)
	return http.ListenAndServe(s.Address, handler)
}

// Manifest handles requests for HLS manifest files
// It generates a manifest file for the given source
// and serves it to the client
func (s *Server) Manifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	src := r.URL.Query().Get("src")
	if src == "" {
		http.Error(w, "Missing src parameter", http.StatusBadRequest)
		return
	}
	manifest, err := s.getManifest(src)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate manifest: %v", err), http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, manifest)
}

// Playlist handles requests for HLS playlist files
func (s *Server) Playlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/playlist"), "/v0.m3u8")
	playlist, error := s.getPlaylist(id)
	if error != nil {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	http.ServeFile(w, r, playlist)
}

// Segment handles requests for HLS segment files
func (s *Server) Segment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
	seg, err := s.getSegment(playlistID, segNum)
	if err != nil {
		http.Error(w, "Failed to get segment", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "video/mp2t")
	http.ServeFile(w, r, seg)
}
