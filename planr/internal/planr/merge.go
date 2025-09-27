package planr

import "github.com/dnswlt/hackz/planr/planpb"

func MergePlans(plans []*planpb.Plan) (*Plan, error) {
	plan := &planpb.Plan{}

	apps := map[string]*planpb.Application{}

	for _, p := range plans {
		// Merge L3 modules, not applications.
		for _, app := range p.GetApplications() {
			existing, found := apps[app.Name]
			if !found {
				apps[app.Name] = app
				continue
			}
			// Existing app: merge modules and description
			if app.Description != "" {
				existing.Description = app.Description
			}
			existing.Modules = append(existing.Modules, app.Modules...)
		}

		plan.Datastores = append(plan.Datastores, p.Datastores...)
		plan.Interfaces = append(plan.Interfaces, p.Interfaces...)
		plan.Releases = append(plan.Releases, p.Releases...)
	}

	// Copy over merged applications
	for _, app := range apps {
		plan.Applications = append(plan.Applications, app)
	}

	return NewPlan(plan), nil
}
