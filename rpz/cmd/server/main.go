package main

import (
	"flag"
	"log"

	"github.com/dnswlt/hackz/rpz"
)

func main() {
	mode := flag.String("mode", "grpc", "Server mode: grpc or http")
	insecure := flag.Bool("insecure", false, "If false, TLS is disabled")
	certFile := flag.String("tls-cert", "certs/dev_cert.pem", "TLS certificate")
	keyFile := flag.String("tls-key", "certs/dev_key.pem", "TLS key")
	flag.Parse()

	switch *mode {
	case "grpc":
		log.Println("Starting gRPC server...")
		s := rpz.NewGRPCServer()
		if !*insecure {
			s.ServeTLS(*certFile, *keyFile)
		} else {
			s.Serve()
		}
	case "http":
		log.Println("Starting HTTP server...")
		s := &rpz.HTTPServer{}
		s.Serve()
	default:
		log.Fatalf("Unknown mode: %s (use grpc or http)", *mode)
	}
}
