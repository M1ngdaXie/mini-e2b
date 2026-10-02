package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/M1ngdaXie/mini-e2b/internal/fc"
)

func main() {
	sockPath := "/tmp/fc.sock"
	process, err := fc.NewVM(sockPath)
	if err != nil {
		log.Fatalf("Error creating VM : %v", err)
	}
	err = process.Boot()
	if err != nil {
		log.Fatalf("Error booting VM : %v", err)
	}

	signCh := make(chan os.Signal, 1)
	signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)
out:
	for {
		select {
		case <-signCh:
			log.Println("Received interrupt signal, stopping...")
			err := process.Stop()
			if err != nil {
				log.Printf("Error stopping VM : %v", err)
			}
			break out
		case err := <-process.Done():
			err = process.Stop()
			if err != nil {
				log.Printf("Error stopping VM : %v", err)
			}
			break out
		}
	}
}
