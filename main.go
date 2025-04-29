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

	// Start the server
	log.Printf("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
