package planr

import (
	"time"

	"github.com/dnswlt/hackz/planr/planpb"
)

type Plan struct {
	plan     *planpb.Plan
	releases map[string]*planpb.Release
}

func NewPlan(plan *planpb.Plan) *Plan {
	p := &Plan{
		plan:     plan,
		releases: make(map[string]*planpb.Release),
	}
	for _, rel := range p.plan.GetReleases() {
		p.releases[rel.GetName()] = rel
	}

	return p
}

func (p *Plan) Proto() *planpb.Plan {
	return p.plan
}

func (p *Plan) GetProcesses() []*planpb.Process {

	var processes []*planpb.Process

	for _, app := range p.plan.GetApplications() {
		for _, mod := range app.GetModules() {
			processes = append(processes, mod.GetProcesses()...)
		}
	}

	return processes
}

func (p *Plan) FreezeDate(releaseID string) (time.Time, bool) {
	rel, ok := p.releases[releaseID]
	if !ok {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", rel.FreezeDate)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
