// cmd/planr/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dnswlt/hackz/planr/planpb"

	"gopkg.in/yaml.v3"
)

func main() {
	planFile := flag.String("plan", "plan.yml", "Path to the migration plan YAML file")
	flag.Parse()

	yamlFile, err := os.ReadFile(*planFile)
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	var plan planpb.Plan

	err = yaml.Unmarshal(yamlFile, &plan)
	if err != nil {
		log.Fatalf("Error unmarshaling YAML: %v", err)
	}

	fmt.Printf("Read %d applications from %s\n", len(plan.GetApplications()), *planFile)
	for i, app := range plan.GetApplications() {
		fmt.Printf("Application #%d: %s\n", i, app.Name)
	}
}
