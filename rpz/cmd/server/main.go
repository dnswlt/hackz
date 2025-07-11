package main

import (
	"flag"
	"log"

	"github.com/dnswlt/hackz/rpz"
)

func main() {
	mode := flag.String("mode", "grpc", "Server mode: grpc or http")
	flag.Parse()

	switch *mode {
	case "grpc":
		log.Println("Starting gRPC server...")
		s := rpz.NewGRPCServer()
		s.Serve()
	case "http":
		log.Println("Starting HTTP server...")
		s := &rpz.HTTPServer{}
		s.Serve()
	default:
		log.Fatalf("Unknown mode: %s (use grpc or http)", *mode)
	}
}
