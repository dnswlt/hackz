package main

import (
	"flag"
	"log"

	"github.com/dnswlt/hackz/rpz"
)

func main() {
	var config rpz.Config
	mode := flag.String("mode", "grpc", "Server mode: {grpc, http}")
	flag.BoolVar(&config.Insecure, "insecure", false, "If true, TLS is disabled")
	flag.StringVar(&config.CertFile, "tls-cert", "certs/dev_cert.pem", "TLS certificate")
	flag.StringVar(&config.KeyFile, "tls-key", "certs/dev_key.pem", "TLS key")
	flag.IntVar(&config.PayloadBytes, "payload-bytes", 0,
		"Number of random bytes to return in the .payload field (for http these are base64 encoded)")
	flag.Parse()

	switch *mode {
	case "grpc":
		log.Println("Starting gRPC server...")
		s := rpz.NewGRPCServer(config)
		s.Serve()
	case "http":
		proto := "https"
		if config.Insecure {
			proto = "http"
		}
		log.Printf("Starting %s server...\n", proto)
		s := rpz.NewHTTPServer(config)
		s.Serve()
	default:
		log.Fatalf("Unknown mode: %s (use grpc or http)", *mode)
	}
}
