package main

import (
	"log"
	"net/http"

	"github.com/M1ngdaXie/mini-e2b/internal/proxy"
)

func main() {
	p, err := proxy.NewProxy("http://localhost:3000")
	if err != nil {
		log.Fatalf("error starting server:%w", err)
	}
	log.Fatal(http.ListenAndServe(":9000", p))
}
