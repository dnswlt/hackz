package main

import (
	"flag"
	"log"

	"github.com/dnswlt/hackz/rpz"
)

func main() {
	mode := flag.String("mode", "grpc", "Server mode: {grpc, http}")
	insecure := flag.Bool("insecure", false, "If true, TLS is disabled")
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
		proto := "https"
		if *insecure {
			proto = "http"
		}
		log.Printf("Starting %s server...\n", proto)
		s := &rpz.HTTPServer{}
		if !*insecure {
			s.ServeTLS(*certFile, *keyFile)
		} else {
			s.Serve()
		}
	default:
		log.Fatalf("Unknown mode: %s (use grpc or http)", *mode)
	}
}
