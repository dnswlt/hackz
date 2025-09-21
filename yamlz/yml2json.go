package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("No YAML file specified")
	}
	filename := os.Args[1]

	// Open the file for streaming instead of reading it all at once.
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Error opening YAML file: %v", err)
	}
	defer file.Close()

	// Create a new YAML decoder from the file stream.
	decoder := yaml.NewDecoder(file)

	var allDocs []any

	// Loop through the YAML documents in the file.
	for {
		var doc any
		// Decode one document at a time.
		err := decoder.Decode(&doc)

		// The io.EOF error signals that we've reached the end of the file.
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatalf("Error decoding YAML document: %v", err)
		}

		allDocs = append(allDocs, doc)
	}

	// Marshal the slice of documents into a single JSON array.
	for i, doc := range allDocs {
		fmt.Printf("--- Document #%d ---\n", i+1)
		jsonOutput, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			log.Fatalf("Error converting to JSON: %v", err)
		}
		fmt.Println(string(jsonOutput))
	}
}
