package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cfg "github.com/M1ngdaXie/mini-e2b/internal/config"
	"github.com/M1ngdaXie/mini-e2b/internal/server"
)

func main() {
	config := flag.String("config", "config.json", "load config")
	flag.Parse()
	curCfg, err := cfg.Load(*config)
	if err != nil {
		log.Fatalf("Error : %v ", err)
	}
	s := server.NewServer("127.0.0.1:9999", nil, *curCfg)
	log.Println("Server started")
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)
	<-exitChan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = s.HandleShutdown(ctx)
	if err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
