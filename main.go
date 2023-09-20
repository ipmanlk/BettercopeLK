package main

import (
	"ipmanlk/bettercopelk/handlers"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handlers.HandlePublicDirServe)
	http.HandleFunc("/search", handlers.HandleSearch)
	http.HandleFunc("/download", handlers.HandleDownload)

	log.Println("listening on", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
