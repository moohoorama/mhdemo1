// Temporary tool: how many rounds enemies standing by the way cost the player's AI on its walk to a goal
// cell. The B09 sortie (stage level 15) starts at the west end of an open grass field and Liu Bei must reach
// the east end; k enemy footmen stand in the middle, on the way or a few cells off it.
//
//	go run ./cmd/_routesim -seeds 10
package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"strings"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

const w, h = 26, 9

var (
	narrow, rush bool
	goal         string
)

func main() {
	seeds := flag.Int("seeds", 10, "seeds per case")
	flag.BoolVar(&rush, "rush", false, "with -trace: the careless player (nearest enemy only)")
	flag.BoolVar(&narrow, "narrow", false, "a three-cell-wide road between mountains instead of an open field")
	trace := flag.String("trace", "", "instead: play this stage once (seed 1) and print where Liu Bei stands each round")
	flag.StringVar(&goal, "goal", "", "with -trace: the stage's victory conditions instead, comma-separated, to send the AI somewhere")
	flag.Parse()
	d, err := content.Load("assets/data")
	check(err)
	if *trace != "" {
		traceStage(d, *trace)
		return
	}
	base := d.StageIndex("B09")
	for _, policy := range []string{"hold", "assault"} {
		for _, off := range []int{0, 3} {
			if narrow && off > 0 {
				continue
			}
			fmt.Printf("== enemies %s, %d cells off the way\n", policy, off)
			var r0 float64
			for _, k := range []int{0, 1, 2, 3, 4, 6, 8} {
				var wins, rounds int
				for seed := 1; seed <= *seeds; seed++ {
					res, round := play(d, base, k, off, policy, uint64(seed))
					if res == "victory" {
						wins++
						rounds += round
					}
				}
				mean := float64(rounds) / float64(max(wins, 1))
				if k == 0 {
					r0 = mean
				}
				fmt.Printf("  k=%d  win %2d/%d  rounds %.1f  delay %+.1f\n", k, wins, *seeds, mean, mean-r0)
			}
		}
	}
}

func play(d *content.Data, base, k, off int, policy string, seed uint64) (string, int) {
	dd := *d
	dd.Characters = maps.Clone(d.Characters)
	level := d.Rules.StageLevel(d.Stages[base].Order)
	for id, c := range dd.Characters {
		if c.Playable {
			c.Level, c.Points = level, 100*(level-1)
			dd.Characters[id] = c
		}
	}
	st := d.Stages[base]
	st.Map, st.Width, st.Height, st.TurnLimit = "", w, h, 30
	st.Tiles = make([]string, h)
	for y := range st.Tiles {
		st.Tiles[y] = strings.Repeat("g--", w)
		if narrow && (y < 3 || y > 5) {
			st.Tiles[y] = strings.Repeat("n--", w)
		}
	}
	st.Allies = []content.Point{{X: 1, Y: 4}, {X: 1, Y: 3}, {X: 1, Y: 5}, {X: 0, Y: 4}, {X: 0, Y: 3}}
	st.Areas = []content.Area{{ID: 1, Rect: content.MapRect{X: w - 2, Y: 4, W: 1, H: 1}, Name: "goal"}}
	st.Victory = []content.Goal{{If: []string{"at:liubei:1"}, Text: "goal"}}
	st.Sub, st.Events, st.Duels, st.Party = nil, nil, nil, nil
	st.Enemies = nil
	for i := range k {
		y := 4 + off
		if i%2 == 1 {
			y = 4 - off
		}
		if off == 0 {
			y = 4 + []int{0, -1, 1, 0, -1, 1, 0, -1}[i]
			if !narrow {
				y = 4 + []int{0, -1, 1, -2, 2, -3, 3, -4}[i]
			}
		}
		st.Enemies = append(st.Enemies, content.Spawn{Character: "troop_footman", X: 12 + i/3, Y: y, Facing: "SW", Policy: policy})
	}
	dd.Stages = append([]content.Stage(nil), d.Stages...)
	dd.Stages[base] = st
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == st.ID {
			dd.Campaign.Start = n.ID
		}
	}
	dd.Campaign.Roster = []string{"liubei", "guanyu", "zhangfei", "fangong", "jianyong"}
	dd.Campaign.Deployment = dd.Campaign.Roster
	e := core.New(&dd, seed)
	_, err := e.Apply(core.Command{Kind: "start"})
	check(err)
	for step := 0; step < 20000; step++ {
		o := e.Observe()
		if o.Phase != "battle" && o.Phase != "deployment" {
			break
		}
		c, err := ai.Next(e)
		check(err)
		_, err = e.Apply(c)
		check(err)
	}
	o := e.Observe()
	return o.Result, o.Round
}

func traceStage(d *content.Data, id string) {
	next := ai.Next
	if rush {
		next = ai.RushNext
	}
	dd := *d
	dd.Characters = maps.Clone(d.Characters)
	i := d.StageIndex(id)
	level := d.Rules.StageLevel(d.Stages[i].Order)
	for cid, c := range dd.Characters {
		if c.Playable {
			c.Level, c.Points = level, 100*(level-1)
			dd.Characters[cid] = c
		}
	}
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == id {
			dd.Campaign.Start = n.ID
		}
	}
	if goal != "" {
		dd.Stages = append([]content.Stage(nil), d.Stages...)
		dd.Stages[i].Victory = []content.Goal{{If: strings.Split(goal, ","), Text: "trace"}}
	}
	dd.Campaign.Roster = []string{"liubei", "guanyu", "zhangfei", "fangong", "jianyong"}
	dd.Campaign.Deployment = dd.Campaign.Roster
	e := core.New(&dd, 1)
	_, err := e.Apply(core.Command{Kind: "start"})
	check(err)
	last := -1
	for step := 0; step < 20000; step++ {
		o := e.Observe()
		if o.Phase != "battle" && o.Phase != "deployment" {
			fmt.Println(o.Result, "round", o.Round)
			return
		}
		if o.Round != last {
			last = o.Round
			for _, u := range o.UnitViews {
				if u.ID == "liubei" || u.ID == "yanliang" {
					fmt.Printf("r%d %s (%d,%d) HP %d\n", o.Round, u.ID, u.X, u.Y, u.HP)
				}
			}
		}
		c, err := next(e)
		check(err)
		r, err := e.Apply(c)
		check(err)
		for _, ev := range r.Events {
			if (ev.Kind == "damage" || ev.Kind == "speech") && (ev.Target == "liubei" || ev.Kind == "speech") {
				for _, u := range e.Observe().UnitViews {
					if u.ID == ev.Actor {
						fmt.Printf("   %s %s from (%d,%d) %d\n", ev.Kind, ev.Actor, u.X, u.Y, ev.Amount)
					}
				}
				if ev.Kind == "speech" && len(ev.Speech) > 0 {
					fmt.Printf("   say %s: %s\n", ev.Speech[0].Actor, ev.Speech[0].Text)
				}
			}
		}
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
