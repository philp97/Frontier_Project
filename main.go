package main

import (
	"log"
	"net/http"

	"frontier/internal/api"
)

func main() {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", api.HealthHandler)
	mux.HandleFunc("/api/analyze", api.AnalyzeHandler)

	// Static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	log.Println("🚀 Frontier server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
