// Temporary tool for staging: prints where each hero stage puts its people and which way they face when
// it opens (after a forced stage's opening cutscene), with the cells and end facing of every cutscene move
// (docs/presentation-facing.md).
//
//	go run ./cmd/_facecheck        every stage
//	go run ./cmd/_facecheck B01    one stage
package main

import (
	"fmt"
	"os"

	"srpg/internal/content"
	"srpg/internal/core"
)

func main() {
	d, err := content.LoadScenario("assets/data", "hero")
	if err != nil {
		panic(err)
	}
	for _, st := range d.Stages {
		if len(os.Args) > 1 && st.ID != os.Args[1] {
			continue
		}
		start := ""
		for _, n := range d.Nodes {
			if n.Kind == "battle" && n.Stage == st.ID {
				start = n.ID
			}
		}
		if start == "" {
			continue
		}
		dd := *d
		dd.Campaign.Start = start
		e := core.New(&dd, 1)
		r, err := e.Apply(core.Command{Kind: "start"})
		if err != nil {
			fmt.Println(st.ID, err)
			continue
		}
		before := map[string]string{}
		for _, u := range e.Observe().Units {
			before[u.ID] = fmt.Sprintf("(%d,%d) %s", u.X, u.Y, u.Facing)
		}
		plan, _ := e.DeploymentPlan()
		for _, c := range plan {
			r2, _ := e.Apply(c)
			r.Events = append(r.Events, r2.Events...)
		}
		fmt.Println("==", st.ID, st.Name)
		for _, u := range e.Observe().Units {
			fmt.Printf("  %-6s %-16s start %-14s now (%d,%d) %s\n", u.Faction, u.ID, before[u.ID], u.X, u.Y, u.Facing)
		}
		for _, ev := range r.Events {
			for _, g := range ev.Cutscene {
				for _, s := range g {
					if s.Kind == "move" {
						fmt.Println("  move", s.Unit, s.Path, s.Facing)
					}
				}
			}
		}
	}
}
