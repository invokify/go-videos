package main

import (
	"go-video/handlers"
	"log"
	"net/http"
)

func main() {
	// Create a new router
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/stream/", handlers.StreamHandler)
	mux.HandleFunc("/videos", handlers.ListVideosHandler)
	mux.HandleFunc("/upload", handlers.UploadHandler)

	// Serve static files from the thumbnails directory
	mux.Handle("/thumbnails/", http.StripPrefix("/thumbnails/", http.FileServer(http.Dir("thumbnails"))))

	// Start the server
	log.Printf("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
