// Temporary tool for writing scenario content: loads a scenario (all validation runs), lists its
// stages and plays each one with the AI on both sides to see how its events and goals behave.
//
//	go run ./cmd/_m8check -scenario hero                  every stage
//	go run ./cmd/_m8check -scenario hero -stage B02 -v   one stage, every event fired
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

func main() {
	dir := flag.String("content", "assets/data", "content directory")
	scenario := flag.String("scenario", "", "scenario id (default: the first)")
	only := flag.String("stage", "", "only this stage id")
	rounds := flag.Int("rounds", 60, "stop a stage after this many rounds")
	seed := flag.Uint64("seed", 1, "random seed")
	verbose := flag.Bool("v", false, "print every event the stage fires")
	flag.Parse()
	d, err := content.LoadScenario(*dir, *scenario)
	check(err)
	fmt.Printf("scenario %s: %d chapters, %d nodes, %d stages\n", d.ScenarioID, len(d.Chapters), len(d.Nodes), len(d.Stages))
	for _, st := range d.Stages {
		if *only != "" && st.ID != *only {
			continue
		}
		play(d, st, *rounds, *seed, *verbose)
	}
}

func play(d *content.Data, st content.Stage, rounds int, seed uint64, verbose bool) {
	start := ""
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == st.ID {
			start = n.ID
		}
	}
	fmt.Printf("\n== %s %s: %dx%d, %d enemies, %d party, %d allies cells, max %d, %d events, %d duels, %d areas\n", st.ID, st.Name, st.Width, st.Height, len(st.Enemies), len(st.Party), len(st.Allies), st.Max, len(st.Events), len(st.Duels), len(st.Areas))
	if start == "" {
		fmt.Println("   no battle node starts this stage")
		return
	}
	dd := *d
	dd.Campaign.Start = start
	e := core.New(&dd, seed)
	if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
		fmt.Println("   start:", err)
		return
	}
	plan, err := e.DeploymentPlan()
	if err != nil {
		fmt.Println("   deployment:", err)
		return
	}
	for _, c := range plan {
		if _, err = e.Apply(c); err != nil {
			fmt.Println("   deployment:", err)
			return
		}
	}
	steps := 0
	for ; steps < 20000; steps++ {
		o := e.Observe()
		if o.Phase != "battle" || o.Round > rounds {
			break
		}
		c, err := ai.Next(e)
		if err != nil {
			fmt.Println("   ai:", err)
			return
		}
		r, err := e.Apply(c)
		if err != nil {
			fmt.Printf("   apply %+v: %v\n", c, err)
			return
		}
		if verbose {
			for _, ev := range r.Events {
				switch ev.Kind {
				case "speech", "unit-arrival", "duel", "retreat", "death", "cutscene", "terrain", "objective", "victory", "turn":
					fmt.Printf("   r%d %s %s %s %s %d\n", o.Round, o.Turn, ev.Kind, ev.Actor, ev.Target+ev.DuelID, ev.Amount)
				}
			}
		}
	}
	o := e.Observe()
	var fired []string
	snap := e.Snapshot()
	for k, v := range snap.Executed {
		if v && strings.HasPrefix(k, "battle-"+st.ID+"-") && !strings.HasSuffix(k, "~on") {
			fired = append(fired, strings.TrimPrefix(k, "battle-"+st.ID+"-"))
		}
		if v && strings.HasPrefix(k, "duel-") {
			fired = append(fired, k)
		}
		if v && strings.HasPrefix(k, "sub-"+st.ID+"-") {
			fired = append(fired, k)
		}
	}
	sort.Strings(fired)
	alive := map[string]int{}
	for _, u := range o.UnitViews {
		if u.HP > 0 {
			alive[u.Faction]++
		}
	}
	fmt.Printf("   result=%q phase=%s round=%d allies=%d enemies=%d\n   fired: %s\n   flags: %v\n", o.Result, o.Phase, o.Round, alive["ally"], alive["enemy"], strings.Join(fired, " "), o.Flags)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
