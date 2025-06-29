package main

import (
	"context"
	"ipmanlk/bettercopelk/internal/handlers"
	"ipmanlk/bettercopelk/internal/services"
	"ipmanlk/bettercopelk/internal/sources"
	"ipmanlk/bettercopelk/internal/sources/baiscopelk"
	"ipmanlk/bettercopelk/internal/sources/cineru"
	"ipmanlk/bettercopelk/internal/sources/piratelk"
	"ipmanlk/bettercopelk/internal/sources/zoomlk"
	"ipmanlk/bettercopelk/internal/static"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	sourceManager := sources.NewManager()

	sourceManager.RegisterSource(baiscopelk.New())
	sourceManager.RegisterSource(cineru.New())
	sourceManager.RegisterSource(piratelk.New())
	sourceManager.RegisterSource(zoomlk.New())

	subtitleService := services.NewSubtitleService(sourceManager)

	subtitleHandler := handlers.NewSubtitleHandler(subtitleService)
	mux := http.NewServeMux()
	subtitleHandler.RegisterRoutes(mux)

	staticHandler := static.GetStaticFileServer()
	mux.Handle("GET /", http.RedirectHandler("/index.html", http.StatusMovedPermanently))
	mux.Handle("GET /index.html", staticHandler)
	mux.Handle("GET /assets/", staticHandler)

	handler := loggingMiddleware(corsMiddleware(mux))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	go func() {
		log.Println("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type, Cache-Control, Connection")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
