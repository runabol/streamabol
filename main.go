package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

const baseDir = "/tmp/streams" // Adjust as needed

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/video", videoHandler)
	mux.HandleFunc("/streams/", serveHLSFiles)
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	src := r.URL.Query().Get("src")
	if src == "" {
		http.Error(w, "Missing src parameter", http.StatusBadRequest)
		return
	}

	hash := hashSrc(src)
	outputDir := baseDir + "/" + hash
	inputPath := outputDir + "/input.mp4"

	// Check if we've already processed this src
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		err = fetchVideo(src, inputPath) // Download directly to inputPath
		if err != nil {
			http.Error(w, "Failed to fetch video: %v", http.StatusInternalServerError)
			return
		}

		// Generate playlists
		err = generateHLS(inputPath, baseDir, src)
		if err != nil {
			http.Error(w, "Failed to process video: %v", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	http.ServeFile(w, r, outputDir+"/master.m3u8")
}

func fetchVideo(src, outputPath string) error {
	resp, err := http.Get(src)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Ensure the directory exists
	err = os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return err
	}

	// Create the output file
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Download directly to outputPath
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	log.Printf("Downloaded %s to %s", src, outputPath)
	return nil
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

func generateHLS(inputPath string, baseDir string, src string) error {
	duration, err := probeVideo(inputPath)
	if err != nil {
		return err
	}

	// Use MD5 hash of src as folder name
	hash := hashSrc(src)
	outputDir := baseDir + "/" + hash
	os.MkdirAll(outputDir+"/v0", 0755)

	// Write master.m3u8 with relative path
	masterContent := fmt.Sprintf(`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-STREAM-INF:BANDWIDTH=561911,AVERAGE-BANDWIDTH=497690,RESOLUTION=640x360,CODECS="avc1.64001e,mp4a.40.2"
/streams/%s/v0.m3u8
`, hash)
	err = os.WriteFile(outputDir+"/master.m3u8", []byte(masterContent), 0644)
	if err != nil {
		log.Printf("Error writing master.m3u8: %v", err)
		return err
	}

	// Generate v0.m3u8 with segment entries
	segmentDuration := 10.0
	numSegments := int(duration / segmentDuration)
	if duration-float64(numSegments)*segmentDuration > 0 {
		numSegments++
	}

	var v0Content strings.Builder
	v0Content.WriteString("#EXTM3U\n")
	v0Content.WriteString("#EXT-X-VERSION:3\n")
	v0Content.WriteString("#EXT-X-TARGETDURATION:10\n")
	v0Content.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	v0Content.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")

	for i := 0; i < numSegments; i++ {
		remaining := duration - float64(i)*segmentDuration
		segDur := segmentDuration
		if remaining < segmentDuration {
			segDur = remaining
		}
		v0Content.WriteString("#EXTINF:" + strconv.FormatFloat(segDur, 'f', 3, 64) + ",\n")
		v0Content.WriteString(fmt.Sprintf("/streams/%s/v0/segment%d.ts\n", hash, i))
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
	fullPath := "/tmp" + path // Base path matches /tmp/streams/

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
			re := regexp.MustCompile(`/streams/([0-9a-f]{32})/v0/segment(\d+)\.ts`)
			matches := re.FindStringSubmatch(path)
			if len(matches) < 3 {
				http.Error(w, "Invalid segment path", http.StatusBadRequest)
				return
			}
			hash := matches[1] // MD5 hash from URL
			segNum, _ := strconv.Atoi(matches[2])
			startTime := segNum * 10
			duration := 10

			// Use hash to locate input file
			inputPath := "/tmp/streams/" + hash + "/input.mp4"
			if _, err := os.Stat(inputPath); os.IsNotExist(err) {
				http.Error(w, "Source video not found", http.StatusInternalServerError)
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

func encodeChunk(inputPath, outputPath string, startTime int, duration int) error {
	log.Printf("Encoding %s: start=%d, duration=%d", outputPath, startTime, duration)
	cmd := ffmpeg_go.Input(inputPath, ffmpeg_go.KwArgs{
		"ss": startTime, // Seek to start time
	}).
		Output(outputPath, ffmpeg_go.KwArgs{
			"t":                duration,
			"c:v":              "libx264",
			"preset":           "ultrafast",
			"c:a":              "aac",
			"b:a":              "128k",
			"f":                "mpegts",
			"output_ts_offset": startTime,
		}).
		OverWriteOutput()
	// WithErrorOutput(os.Stderr)

	log.Printf("FFmpeg command: %s", cmd.String())
	err := cmd.Run()
	if err != nil {
		log.Printf("Encode chunk error: %v", err)
	} else {
		log.Printf("Successfully encoded %s", outputPath)
	}
	return err
}

func hashSrc(src string) string {
	hash := md5.Sum([]byte(src))
	return hex.EncodeToString(hash[:]) // 32-char hex string
}
