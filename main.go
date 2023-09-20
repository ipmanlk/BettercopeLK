package main

import (
	"ipmanlk/bettercopelk/handlers"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handlers.HandlePublicDirServe)
	http.HandleFunc("/search", handlers.HandleSearch)
	http.HandleFunc("/download", handlers.HandleDownload)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
