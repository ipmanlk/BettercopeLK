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
	http.HandleFunc("/search", handlers.HandleSearch)
	http.HandleFunc("/download", handlers.HandleDownload)
	http.HandleFunc("/bulk-download", handlers.HandleBulkDownload)

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
