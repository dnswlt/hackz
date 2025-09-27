package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dnswlt/hackz/planr/internal/backstage"
)

func main() {

	svgFlag := flag.String("svg", "", "Component to render an SVG for")
	flag.Parse()

	repo := backstage.NewRepository()

	for _, arg := range flag.Args() {
		es, err := backstage.ReadEntities(arg)
		if err != nil {
			log.Fatalf("Failed to read %s: %v", arg, err)
		}
		for _, e := range es {
			err = repo.AddEntity(e)
			if err != nil {
				log.Fatalf("Failed to add entity %s to repository: %v", e.GetQName(), err)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "Read %d entities\n", repo.Size())

	if err := repo.Validate(); err != nil {
		log.Fatalf("Repository validation failed: %v", err)
	}

	if *svgFlag != "" {
		svg, err := backstage.GenerateComponentSVG(repo, *svgFlag)
		if err != nil {
			log.Fatalf("SVG generation failed: %v", err)
		}
		fmt.Println(string(svg))
	}
}
