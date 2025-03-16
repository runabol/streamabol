package handlers

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

const baseDir = "/tmp/streams" // Adjust as needed

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

func Video(w http.ResponseWriter, r *http.Request) {
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
	sourceFile := outputDir + "/source.txt"

	// Check if we've already processed this src
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		// Store the URL in the output directory
		err = os.MkdirAll(outputDir, 0755)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
			return
		}
		err = os.WriteFile(sourceFile, []byte(src), 0644)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to write source file: %v", err), http.StatusInternalServerError)
			return
		}

		// Generate playlists
		err = generateHLS(src, outputDir, hash)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to process video: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	http.ServeFile(w, r, outputDir+"/master.m3u8")
}

func generateHLS(src, outputDir, hash string) error {
	duration, err := probeVideo(src)
	if err != nil {
		return err
	}

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

func probeVideo(src string) (float64, error) {
	output, err := ffmpeg_go.Probe(src)
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

func hashSrc(src string) string {
	hash := md5.Sum([]byte(src))
	return hex.EncodeToString(hash[:]) // 32-char hex string
}
