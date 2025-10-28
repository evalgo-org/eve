// +build ignore

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eve.evalgo.org/network"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "proxy-config.json", "Path to proxy configuration file")
	flag.Parse()

	// Create proxy instance
	proxy, err := network.NewZitiProxy(*configPath)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start proxy in goroutine
	go func() {
		if err := proxy.Start(); err != nil {
			log.Fatalf("Proxy error: %v", err)
		}
	}()

	log.Println("Ziti Proxy is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-stop

	log.Println("Shutting down gracefully...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop proxy
	if err := proxy.Stop(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Proxy stopped successfully")
}
