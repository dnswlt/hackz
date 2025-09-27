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
	printYaml := flag.Bool("print-yaml", true, "If set, prints the merged YAML to stdout.")
	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatal("No YAML files specified.")
	}

	var plans []*planpb.Plan

	for _, planFile := range flag.Args() {
		yamlFile, err := os.ReadFile(planFile)
		if err != nil {
			log.Fatalf("Error reading YAML file: %v", err)
		}

		var p planpb.Plan
		err = yaml.UnmarshalStrict(yamlFile, &p)
		if err != nil {
			log.Fatalf("Error unmarshaling YAML: %v", err)
		}
		plans = append(plans, &p)
	}

	plan, err := planr.MergePlans(plans)
	if err != nil {
		log.Fatalf("Could not merge plans: %v", err)
	}
	fmt.Printf("Read %d processes from %d files\n", len(plan.GetProcesses()), len(flag.Args()))

	if err := planr.ValidatePlan(plan); err != nil {
		log.Fatalf("Plan validation failed: %v", err)
	} else {
		log.Println("Plan validated successfully.")
	}

	if *printYaml {
		bs, err := yaml.Marshal(plan.Proto())
		if err != nil {
			log.Fatalf("Error marshaling YAML: %v", err)
		}
		fmt.Println(string(bs))
	}

	planr.PrintTimeline(plan, "halon-migration")
}
