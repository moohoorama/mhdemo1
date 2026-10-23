// Temporary tool for stage level balance (docs/stage_lv_balance.md): starts each stage with the
// player's officers at the stage level (Rules.StageLevel), plays it with a player AI over several
// seeds and reports the win rate, the rounds and the levels the officers gained.
//
//	go run ./cmd/_balance                       every stage, careless player (rush)
//	go run ./cmd/_balance -stage B01 -player ai -seeds 10
package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"sort"
	"strings"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/storage"
)

func main() {
	dir := flag.String("content", "assets/data", "content directory")
	only := flag.String("stage", "", "only this stage id")
	player := flag.String("player", "rush", "rush (nearest enemy only) or ai (the game's own AI)")
	seeds := flag.Int("seeds", 5, "seeds per stage")
	first := flag.Int("first", 1, "first seed")
	damage := flag.Int("damage", storage.DefaultScalePct, "damage scale in percent, as the settings set it")
	counter := flag.Bool("counter", storage.DefaultSettings().CounterAll, "every unit strikes back (the counter-all setting)")
	xp := flag.Int("xp", storage.DefaultScalePct, "experience scale in percent, as the settings set it")
	offset := flag.Int("offset", 0, "the player's officers this many levels off the stage level")
	show := flag.Bool("show", false, "print every unit's numbers when the battle starts (seed 1)")
	flag.Parse()
	d, err := content.Load(*dir)
	check(err)
	for i, st := range d.Stages {
		if *only != "" && st.ID != *only {
			continue
		}
		level := d.Rules.StageLevel(st.Order) + *offset
		var wins, rounds int
		gain := map[string][]float64{}
		var lost []string
		for seed := *first; seed < *first+*seeds; seed++ {
			res, round, gained, retreats := play(d, i, level, uint64(seed), *player, *damage, *xp, *counter, *show && seed == *first)
			if res == "victory" {
				wins++
			}
			rounds += round
			for id, g := range gained {
				gain[id] = append(gain[id], g)
			}
			lost = append(lost, retreats...)
		}
		fmt.Printf("%s %-14s Lv%-2d %s: win %d/%d, rounds %.1f", st.ID, st.Name, level, *player, wins, *seeds, float64(rounds)/float64(*seeds))
		ids := make([]string, 0, len(gain))
		for id := range gain {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		var parts []string
		for _, id := range ids {
			sum := 0.0
			for _, g := range gain[id] {
				sum += g
			}
			parts = append(parts, fmt.Sprintf("%s +%.1f", id, sum/float64(len(gain[id]))))
		}
		fmt.Printf(", levels: %s", strings.Join(parts, " "))
		if len(lost) > 0 {
			fmt.Printf(", retreats: %s", strings.Join(count(lost), " "))
		}
		fmt.Println()
	}
}

func count(ids []string) []string {
	n := map[string]int{}
	for _, id := range ids {
		n[id]++
	}
	var out []string
	for id, k := range n {
		out = append(out, fmt.Sprintf("%s×%d", id, k))
	}
	sort.Strings(out)
	return out
}

// play fights stage i with every playable officer at level, and returns the result, the last round,
// the levels each of the player's officers gained (fractions from experience) and who retreated.
func play(d *content.Data, i, level int, seed uint64, player string, damage, xp int, counter bool, show bool) (string, int, map[string]float64, []string) {
	dd := *d
	dd.Characters = maps.Clone(d.Characters)
	for id, c := range dd.Characters {
		if c.Playable {
			c.Level, c.Points = level, 100*(level-1)
			dd.Characters[id] = c
		}
	}
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == d.Stages[i].ID {
			dd.Campaign.Start = n.ID
		}
	}
	e := core.New(&dd, seed)
	e.SetDamagePct(damage)
	e.SetXPPct(xp)
	e.SetCounterAll(counter)
	_, err := e.Apply(core.Command{Kind: "start"})
	check(err)
	var retreats []string
	start := map[string]float64{}
	shown := false
	for step := 0; step < 20000; step++ {
		o := e.Observe()
		if o.Phase != "battle" && o.Phase != "deployment" {
			break
		}
		if listed := show && o.Phase == "battle" && !shown; listed {
			shown = true
			for _, u := range o.UnitViews {
				fmt.Printf("  %-6s %-18s %-15s Lv%-2d HP%5d atk%4d def%4d mind%4d agi%4d mor%4d\n", u.Faction+map[bool]string{true: "+ai"}[u.AI], u.ID, u.Class, u.Level, u.Stats.MaxHP, u.Stats.Attack, u.Stats.Defense, u.Stats.Mind, u.Stats.Agility, u.Stats.Morale)
			}
		}
		for _, u := range o.UnitViews {
			if u.Faction == "ally" && !u.AI {
				if _, ok := start[u.ID]; !ok {
					start[u.ID] = float64(u.Level) + u.XPProgress/100
				}
			}
		}
		next := ai.Next
		if player == "rush" {
			next = ai.RushNext
		}
		c, err := next(e)
		check(err)
		r, err := e.Apply(c)
		check(err)
		for _, ev := range r.Events {
			if show && (ev.Kind == "retreat" || ev.Kind == "death" || ev.Kind == "duel" || ev.Kind == "speech" || ev.Kind == "damage" || ev.Kind == "critical") {
				fmt.Printf("    r%d %s %s %s%s %d", e.Observe().Round, ev.Kind, ev.Actor, ev.Target, ev.DuelID, ev.Amount)
				if ev.Text == "flight" {
					for _, u := range e.Observe().UnitViews {
						if u.ID == ev.Actor {
							fmt.Printf(" flight at (%d,%d)", u.X, u.Y)
						}
					}
				}
				fmt.Println()
			}
			if ev.Kind == "retreat" || ev.Kind == "death" {
				for _, u := range e.Observe().UnitViews {
					if u.ID == ev.Actor && u.Faction == "ally" && !u.AI {
						retreats = append(retreats, u.ID)
					}
				}
			}
		}
	}
	o := e.Observe()
	gained := map[string]float64{}
	for _, u := range o.UnitViews {
		if s, ok := start[u.ID]; ok {
			gained[u.ID] = float64(u.Level) + u.XPProgress/100 - s
		}
	}
	return o.Result, o.Round, gained, retreats
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
