package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/h2non/filetype"
)

type VideoInfo struct {
	Title     string `json:"title"`
	Duration  string `json:"duration"`
	Thumbnail string `json:"thumbnail"` // Base64 encoded thumbnail
}

func ListVideosHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	videos, err := getVideosList()
	if err != nil {
		log.Printf("Error getting videos list: %v", err)
		http.Error(w, "Failed to get videos list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(videos)
}

func getVideosList() ([]VideoInfo, error) {
	var videos []VideoInfo

	err := filepath.Walk("videos", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is a video
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Read first 261 bytes for filetype detection
		head := make([]byte, 261)
		if _, err := file.Read(head); err != nil {
			return err
		}

		// Reset file pointer
		if _, err := file.Seek(0, 0); err != nil {
			return err
		}

		kind, err := filetype.Match(head)
		if err != nil || !isVideo(kind.MIME.Value) {
			return nil
		}

		// Get video metadata
		videoInfo, err := extractVideoMetadata(path)
		if err != nil {
			log.Printf("Error extracting metadata for %s: %v", path, err)
			return nil
		}

		videos = append(videos, videoInfo)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return videos, nil
}

func isVideo(mimeType string) bool {
	videoMimeTypes := []string{
		"video/mp4",
		"video/webm",
		"video/x-matroska",
		"video/quicktime",
		"video/x-msvideo",
	}

	for _, vmt := range videoMimeTypes {
		if mimeType == vmt {
			return true
		}
	}
	return false
}

func extractVideoMetadata(videoPath string) (VideoInfo, error) {
	file, err := os.Open(videoPath)
	if err != nil {
		return VideoInfo{}, err
	}
	defer file.Close()

	// Get video title (filename without extension)
	base := filepath.Base(videoPath)
	ext := filepath.Ext(videoPath)
	title := base[:len(base)-len(ext)]

	// For MP4 files, try to get duration
	var duration time.Duration
	if ext == ".mp4" {
		duration, err = getMP4Duration(file)
		if err != nil {
			log.Printf("Error getting MP4 duration: %v", err)
		}
	}

	// For now, we'll use a placeholder thumbnail
	// In a real implementation, you might want to use a different approach
	// like generating thumbnails on upload and storing them separately
	thumbnail := "data:image/jpeg;base64,/9j/4AAQSkZJRg..."

	return VideoInfo{
		Title:     title,
		Duration:  formatDuration(duration),
		Thumbnail: thumbnail,
	}, nil
}

func getMP4Duration(file *os.File) (time.Duration, error) {
	// For now, return a default duration
	// In a real implementation, you would use a proper MP4 parser
	return 0, nil
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
