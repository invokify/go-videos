package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/h2non/filetype"
)

type VideoInfo struct {
	Title     string `json:"title"`
	Duration  string `json:"duration"`
	Thumbnail string `json:"thumbnail"` // Base64 encoded thumbnail
}

type PageData struct {
	Videos []VideoInfo
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

	// Parse templates
	tmpl, err := template.ParseFiles("templates/base.html", "templates/videos.html")
	if err != nil {
		log.Printf("Error parsing templates: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Prepare data for template
	data := PageData{
		Videos: videos,
	}

	// Execute template
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
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

	// Get duration for all video types
	duration, err := getMP4Duration(file)
	if err != nil {
		log.Printf("Error getting video duration: %v", err)
	}

	// Generate thumbnail using FFmpeg
	thumbnail, err := generateThumbnail(videoPath)
	if err != nil {
		log.Printf("Error generating thumbnail: %v", err)
		// Use a default thumbnail if generation fails
		thumbnail = "/thumbnails/default.jpg"
	}

	return VideoInfo{
		Title:     title,
		Duration:  formatDuration(duration),
		Thumbnail: thumbnail,
	}, nil
}

func getMP4Duration(file *os.File) (time.Duration, error) {
	// Get the file path
	filePath := file.Name()

	// Run FFprobe to get duration
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get duration: %v", err)
	}

	// Parse the duration (FFprobe outputs seconds as a float)
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}

	// Convert to time.Duration
	return time.Duration(duration * float64(time.Second)), nil
}

func generateThumbnail(videoPath string) (string, error) {
	// Get video filename without extension for the thumbnail name
	base := filepath.Base(videoPath)
	ext := filepath.Ext(videoPath)
	videoName := base[:len(base)-len(ext)]

	// Ensure thumbnails directory exists
	thumbnailDir := "thumbnails"
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnails directory: %v", err)
	}

	// Create path for the thumbnail
	thumbnailPath := filepath.Join(thumbnailDir, videoName+".jpg")

	// Check if thumbnail already exists
	if _, err := os.Stat(thumbnailPath); os.IsNotExist(err) {
		// Run FFmpeg to extract a frame at 1 second
		cmd := exec.Command("ffmpeg", "-i", videoPath, "-ss", "00:00:01", "-vframes", "1", "-q:v", "2", thumbnailPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to generate thumbnail: %v", err)
		}
	}

	// Return the URL path to the thumbnail
	return "/thumbnails/" + videoName + ".jpg", nil
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
