package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	gameHtmlFile  = flag.String("html", "index.html", "Path to game HTML file")
	serverAddress = flag.String("address", "", "Address on which to listen")
	serverPort    = flag.Int("port", 8084, "Port on which to listen")
	gameHtml      string
)

func gameHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, gameHtml)
}

func main() {
	flag.Parse()
	html, err := os.ReadFile(*gameHtmlFile)
	if err != nil {
		log.Fatal("Could not read html: ", err)
	}
	gameHtml = string(html)
	http.HandleFunc("/hexz", gameHandler)
	addr := fmt.Sprintf("%s:%d", *serverAddress, *serverPort)
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
