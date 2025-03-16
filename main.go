package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

var streamCache = make(map[string]string)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/video", videoHandler)
	mux.HandleFunc("/", serveHLSFiles) // Catch-all for .m3u8 and .ts
	handler := corsMiddleware(mux)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func videoHandler(w http.ResponseWriter, r *http.Request) {
	src := r.URL.Query().Get("src")
	if src == "" {
		http.Error(w, "Missing src parameter", http.StatusBadRequest)
		return
	}

	outputDir, exists := streamCache[src]
	if !exists {
		inputPath, err := fetchVideo(src)
		if err != nil {
			http.Error(w, "Failed to fetch video: %v", http.StatusInternalServerError)
			return
		}
		outputDir = "/tmp/stream-" + randomString(8)
		os.MkdirAll(outputDir, 0755)
		err = generateHLS(inputPath, outputDir, src) // Pass src here
		if err != nil {
			http.Error(w, "Failed to process video: %v", http.StatusInternalServerError)
			return
		}
		streamCache[src] = outputDir
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	http.ServeFile(w, r, outputDir+"/master.m3u8")
}

func fetchVideo(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Save to temporary file
	tmpFile := "/tmp/video-" + randomString(8) + ".mp4"
	out, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return tmpFile, err
}

type StreamInfo struct {
	Duration string `json:"duration"`
	Codec    string `json:"codec_name"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type FormatInfo struct {
	Duration string `json:"duration"`
}

type ProbeResult struct {
	Streams []StreamInfo `json:"streams"`
	Format  FormatInfo   `json:"format"`
}

func probeVideo(inputPath string) (float64, error) {
	output, err := ffmpeg_go.Probe(inputPath)
	if err != nil {
		log.Printf("Probe error: %v", err)
		return 0, err
	}

	var result ProbeResult
	err = json.Unmarshal([]byte(output), &result)
	if err != nil {
		log.Printf("Unmarshal error: %v", err)
		return 0, err
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		log.Printf("Duration parse error: %v", err)
		return 0, err
	}
	log.Printf("Video duration: %.2f seconds", duration)
	return duration, nil
}

func generateHLS(inputPath string, outputDir string, src string) error {
	duration, err := probeVideo(inputPath)
	if err != nil {
		return err
	}

	// Create directories
	os.MkdirAll(outputDir+"/v0", 0755)

	// Write master.m3u8 with ?src query param
	masterContent := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=561911,AVERAGE-BANDWIDTH=497690,RESOLUTION=640x360,CODECS="avc1.64001e,mp4a.40.2"
v0.m3u8?src=%s
`, url.QueryEscape(src)) // Escape src to handle special chars
	err = os.WriteFile(outputDir+"/master.m3u8", []byte(masterContent), 0644)
	if err != nil {
		log.Printf("Error writing master.m3u8: %v", err)
		return err
	}

	// Generate v0.m3u8 with segment entries
	segmentDuration := 10.0 // Fixed 10-second segments
	numSegments := int(duration / segmentDuration)
	if duration-float64(numSegments)*segmentDuration > 0 {
		numSegments++ // Account for partial last segment
	}

	var v0Content strings.Builder
	v0Content.WriteString("#EXTM3U\n")
	v0Content.WriteString("#EXT-X-VERSION:3\n")
	v0Content.WriteString("#EXT-X-TARGETDURATION:10\n")
	v0Content.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")

	for i := 0; i < numSegments; i++ {
		remaining := duration - float64(i)*segmentDuration
		segDur := segmentDuration
		if remaining < segmentDuration {
			segDur = remaining // Adjust last segment duration
		}
		v0Content.WriteString("#EXTINF:" + strconv.FormatFloat(segDur, 'f', 3, 64) + ",\n")
		v0Content.WriteString(fmt.Sprintf("v0/segment%d.ts?src=%s\n", i, url.QueryEscape(src)))
	}
	v0Content.WriteString("#EXT-X-ENDLIST\n")

	err = os.WriteFile(outputDir+"/v0.m3u8", []byte(v0Content.String()), 0644)
	if err != nil {
		log.Printf("Error writing v0.m3u8: %v", err)
		return err
	}

	log.Printf("Generated playlists in %s for src=%s", outputDir, src)
	return nil
}

func serveHLSFiles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	src := r.URL.Query().Get("src")
	if src == "" {
		http.Error(w, "Missing src parameter", http.StatusBadRequest)
		return
	}

	outputDir, exists := streamCache[src]
	if !exists {
		http.Error(w, "Stream not found", http.StatusNotFound)
		return
	}

	fullPath := outputDir + path
	log.Printf("Requested: %s", fullPath)

	if strings.HasSuffix(path, ".m3u8") {
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			http.Error(w, "Playlist not found", http.StatusNotFound)
			return
		}
		http.ServeFile(w, r, fullPath)
		return
	}

	if strings.HasSuffix(path, ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			re := regexp.MustCompile(`segment(\d+)\.ts`)
			matches := re.FindStringSubmatch(path)
			if len(matches) < 2 {
				http.Error(w, "Invalid segment name", http.StatusBadRequest)
				return
			}
			segNum, _ := strconv.Atoi(matches[1])
			startTime := segNum * 10
			duration := 10

			inputPath, err := fetchVideo(src) // Optimize this later
			if err != nil {
				http.Error(w, "Failed to fetch source video", http.StatusInternalServerError)
				return
			}

			segDir := filepath.Dir(fullPath)
			os.MkdirAll(segDir, 0755)

			log.Printf("Encoding chunk: %s (start: %d, duration: %d)", fullPath, startTime, duration)
			err = encodeChunk(inputPath, fullPath, startTime, duration)
			if err != nil {
				http.Error(w, "Failed to encode chunk", http.StatusInternalServerError)
				return
			}
		}
		http.ServeFile(w, r, fullPath)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

func encodeChunk(inputPath, outputPath string, startTime, duration int) error {
	return ffmpeg_go.Input(inputPath, ffmpeg_go.KwArgs{"ss": startTime}).
		Output(outputPath, ffmpeg_go.KwArgs{
			"t":   duration,  // Segment duration
			"c:v": "libx264", // H.264 codec
			"c:a": "aac",     // AAC audio
			"f":   "mpegts",  // TS format for HLS
		}).
		OverWriteOutput().Run()
}

// randomString generates a random string of the specified length using letters and numbers.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
