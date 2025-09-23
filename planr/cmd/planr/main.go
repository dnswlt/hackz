// cmd/planr/main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dnswlt/hackz/planr/internal/planr"
	"github.com/dnswlt/hackz/planr/planpb"

	"sigs.k8s.io/yaml"
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

	if err := planr.ValidatePlan(&plan); err != nil {
		log.Fatalf("Plan validation failed: %v", err)
	} else {
		log.Println("Plan validated successfully.")
	}

	bs, err := yaml.Marshal(plan)
	if err != nil {
		log.Fatalf("Error marshaling YAML: %v", err)
	}
	fmt.Println(string(bs))
}
