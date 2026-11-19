package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"distributed-file-index/internal/api"
	"distributed-file-index/internal/index"
)

func main() {
	port := flag.String("port", "8080", "HTTP server port")
	webDir := flag.String("web-dir", "web", "Directory containing static web dashboard files")
	flag.Parse()

	// Allow environment variable override for PORT if set
	if envPort := os.Getenv("PORT"); envPort != "" {
		*port = envPort
	}

	idx := index.New()
	router := api.NewRouter(idx, *webDir)

	addr := ":" + *port
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server run context for graceful shutdown
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 5 seconds
		shutdownCtx, cancel := context.WithTimeout(serverCtx, 5*time.Second)
		defer cancel()

		log.Println("Shutting down server gracefully...")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
		serverStopCtx()
	}()

	fmt.Println("==================================================")
	fmt.Println("  Distributed File Index - Central Index Server   ")
	fmt.Println("  (Napster-Inspired Architecture Demonstration)   ")
	fmt.Println("==================================================")
	fmt.Printf("Server listening on: http://localhost:%s\n", *port)
	fmt.Printf("Dashboard available: http://localhost:%s/\n", *port)
	fmt.Println("Registered endpoints:")
	fmt.Println("  - GET  /health")
	fmt.Println("  - POST /api/v1/peers/register")
	fmt.Println("  - GET  /api/v1/search?file=<filename>")
	fmt.Println("  - GET  /api/v1/peers")
	fmt.Println("  - GET  /api/v1/stats")
	fmt.Println("==================================================")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	<-serverCtx.Done()
	log.Println("Server stopped.")
}
