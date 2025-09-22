package planr

import (
	"fmt"
	"time"

	"github.com/dnswlt/hackz/planr/planpb"
)

func ValidateProcesses(plan *planpb.Plan) error {
	processes := map[string]bool{}

	interfaces := map[string]bool{}
	for _, iface := range plan.GetInterfaces() {
		interfaces[iface.Name] = true
	}
	databases := map[string]bool{}
	for _, db := range plan.GetDatabases() {
		databases[db.Name] = true
	}
	releases := map[string]bool{}
	for _, rel := range plan.GetReleases() {
		releases[rel.Name] = true
	}

	for _, app := range plan.GetApplications() {
		for _, mod := range app.GetModules() {
			for _, p := range mod.GetProcesses() {
				// Duplicate names.
				if processes[p.Name] {
					return fmt.Errorf("duplicate process name: %s", p.Name)
				}
				processes[p.Name] = true

				// Undefined interfaces
				for iface := range p.GetInterfaces().GetConsumed() {
					if !interfaces[iface] {
						return fmt.Errorf("process %s uses undefined interface %s", p.Name, iface)
					}
				}

				// Undefined databases
				for db := range p.GetDatabases() {
					if !databases[db] {
						return fmt.Errorf("process %s uses undefined database %s", p.Name, db)
					}
				}

				// Undefined releases
				rel := p.GetPlannedRelease()
				if !releases[rel] {
					return fmt.Errorf("process %s uses undefined release %s", p.Name, rel)
				}

			}
		}
	}
	return nil
}

func ValidateInterfaces(plan *planpb.Plan) error {
	interfaces := map[string]bool{}
	for _, iface := range plan.GetInterfaces() {
		if interfaces[iface.Name] {
			return fmt.Errorf("duplicate interface definition: %s", iface.Name)
		}
		interfaces[iface.Name] = true
	}
	return nil
}

func ValidateDatabases(plan *planpb.Plan) error {
	dbs := map[string]bool{}
	for _, db := range plan.GetDatabases() {
		if dbs[db.Name] {
			return fmt.Errorf("duplicate database definition: %s", db.Name)
		}
		dbs[db.Name] = true
	}
	return nil
}

func ValidateReleases(plan *planpb.Plan) error {
	releases := map[string]bool{}

	for _, rel := range plan.GetReleases() {
		if releases[rel.Name] {
			return fmt.Errorf("duplicate release: %s", rel.Name)
		}
		releases[rel.Name] = true
		if _, err := time.Parse("2006-01-02", rel.FreezeDate); err != nil {
			return fmt.Errorf("invalid freeze date for release %s (expecting YYYY-MM-DD): %w", rel.Name, err)
		}
		if _, err := time.Parse("2006-01-02", rel.GoliveDate); err != nil {
			return fmt.Errorf("invalid golive date for release %s (expecting YYYY-MM-DD): %w", rel.Name, err)
		}
	}
	return nil
}

func ValidatePlan(plan *planpb.Plan) error {
	if err := ValidateInterfaces(plan); err != nil {
		return err
	}
	if err := ValidateDatabases(plan); err != nil {
		return err
	}
	if err := ValidateReleases(plan); err != nil {
		return err
	}
	if err := ValidateProcesses(plan); err != nil {
		return err
	}
	return nil
}
