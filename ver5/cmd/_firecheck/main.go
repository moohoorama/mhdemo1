// Temporary: plays B02 with the game's AI and prints what 황보숭 does each round, with his MP.
package main

import (
	"fmt"
	"os"
	"maps"
	"slices"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

func main() {
	d, err := content.Load("assets/data")
	check(err)
	difficulty = "normal"
	i := slices.IndexFunc(d.Stages, func(s content.Stage) bool { return s.ID == "B02" })
	if os.Getenv("BASE") != "" {
		st := d.Stages[i]
		st.Party = slices.Clone(st.Party)
		for k := range st.Party {
			if st.Party[k].Character == "huangfusong" {
				st.Party[k].Traits, st.Party[k].Equipment = nil, nil
			}
		}
		d.Stages[i] = st
	}
	wins := 0
	for seed := uint64(1); seed <= 30; seed++ {
		dd := *d
		dd.Characters = maps.Clone(d.Characters)
		for _, n := range d.Nodes {
			if n.Kind == "battle" && n.Stage == "B02" {
				dd.Campaign.Start = n.ID
			}
		}
		dd.Campaign.Roster, dd.Campaign.Warehouse = sortie(&dd, d.Stages[i])
		dd.Campaign.Deployment = dd.Campaign.Roster
		e, err := core.NewGame(&dd, seed, difficulty)
		check(err)
		for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() {
			_, err = e.Apply(c)
			check(err)
		}
		_, err = e.Apply(core.Command{Kind: "start"})
		check(err)
		fmt.Println("seed", seed)
		defer func() {}()
		shown := false
		for step := 0; step < 20000; step++ {
			o := e.Observe()
			if o.Phase != "battle" && o.Phase != "deployment" {
				fmt.Println("  result", o.Result, "round", o.Round)
				if o.Result == "victory" {
					wins++
				}
				break
			}
			c, err := ai.Next(e)
			check(err)
			if c.Actor == "huangfusong" && c.Kind != "move" {
				for _, u := range o.UnitViews {
					if u.ID == "huangfusong" {
						if !shown {
							shown = true
							fmt.Printf("  traits %v skills %v rec %d\n", u.Traits, u.Skills, u.Stats.Recovery)
						}
						fmt.Printf("  r%d MP %2d/%d  %s %s %s\n", o.Round, u.MP, u.Stats.MaxMP, c.Kind, c.Skill, c.Target)
					}
				}
			}
			r, err := e.Apply(c)
			check(err)
			for _, ev := range r.Events {
				if ev.Kind == "retreat" || ev.Kind == "death" {
					fmt.Printf("      HIT %s by %s %d (r%d)\n", ev.Kind, ev.Actor, ev.Amount, e.Observe().Round)
				}
			}
			if c.Actor == "huangfusong" {
				for _, ev := range r.Events {
					if ev.Actor == "huangfusong" && (ev.Kind == "damage" || ev.Kind == "critical") {
						fmt.Printf("      %s -> %s %d\n", ev.Kind, ev.Target, ev.Amount)
					}
				}
			}
		}
	}
	fmt.Println("wins", wins, "/ 30")
}
