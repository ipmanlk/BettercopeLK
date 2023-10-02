package main

import (
	"log"
	"net/http"
	"os"

	"ipmanlk/bettercopelk/handlers"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handlers.HandlePublicDirServe)
	http.HandleFunc("/api/search", handlers.HandleSearch)
	http.HandleFunc("/api/download", handlers.HandleDownload)
	http.HandleFunc("/api/bulk-download", handlers.HandleBulkDownload)

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
