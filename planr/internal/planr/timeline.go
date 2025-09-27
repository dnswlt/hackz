package planr

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/dnswlt/hackz/planr/planpb"
)

func PrintTimeline(plan *Plan, changeID string) {
	var processes []*planpb.Process
	for _, p := range plan.GetProcesses() {
		if _, found := p.Changes[changeID]; !found {
			continue
		}
		processes = append(processes, p)
	}

	slices.SortFunc(processes, func(p1, p2 *planpb.Process) int {
		t1, _ := plan.FreezeDate(p1.Changes[changeID].PlannedRelease)
		t2, _ := plan.FreezeDate(p2.Changes[changeID].PlannedRelease)

		if c := t1.Compare(t2); c != 0 {
			return c
		}
		return cmp.Compare(p1.GetName(), p2.GetName())
	})

	for _, p := range processes {
		fmt.Printf("%s: %s\n", p.GetName(), p.Changes[changeID].PlannedRelease)
	}
}
