// Temporary tool for the growth curve (docs/stage_lv_balance.md): plays the story's battles in order, each
// officer carrying the level it reached into the next (a newcomer at its join level), with a player AI over
// several seeds per battle, and prints every officer's level against the stage's and the enemy's.
//
//	go run ./cmd/_chain -difficulty hard
//	go run ./cmd/_chain -difficulty old_hard -stages 12
package main

import (
	"flag"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"sort"
	"strings"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/storage"
)

func main() {
	dir := flag.String("content", "assets/data", "content directory")
	player := flag.String("player", "ai", "rush or ai")
	seeds := flag.Int("seeds", 4, "seeds per stage")
	stages := flag.Int("stages", 15, "stages")
	flag.StringVar(&difficulty, "difficulty", "normal", "a difficulty id of the rules (easy, normal, hard, old_easy, …)")
	flag.Parse()
	d, err := content.Load(*dir)
	check(err)
	diff, ok := d.Rules.DifficultyOf(difficulty)
	if !ok {
		check(fmt.Errorf("unknown difficulty %q", difficulty))
	}
	R := d.ForDifficulty(difficulty).Rules
	lv := map[string]float64{}
	for i, st := range d.Stages {
		if i >= *stages {
			break
		}
		L := R.StageLevel(st.Order)
		shift := 0.0
		if R.JoinShift {
			shift = diff.EnemyShift
		}
		roster, _ := sortie(d, st)
		for _, id := range roster {
			if _, ok := lv[id]; !ok {
				lv[id] = float64(max(d.Characters[id].Level, R.JoinLevel(st.Order, shift)))
			}
		}
		troop := L + int(math.Round((R.RankLead["troop"]+diff.EnemyShift)*float64(L+R.StatLevelBase)))
		officer := L + int(math.Round((R.RankLead["officer"]+diff.EnemyShift)*float64(L+R.StatLevelBase)))
		levels := map[string]int{}
		for id, v := range lv {
			levels[id] = int(v)
		}
		var wins int
		gain := map[string]float64{}
		for seed := 1; seed <= *seeds; seed++ {
			res, _, gained, _ := play(d, i, levels, uint64(seed), *player)
			if res == "victory" {
				wins++
			}
			for id, g := range gained {
				gain[id] += g / float64(*seeds)
			}
		}
		ids := slices.Sorted(maps.Keys(lv))
		var parts []string
		for _, id := range ids {
			if slices.Contains(roster, id) {
				lv[id] += gain[id]
				parts = append(parts, fmt.Sprintf("%s %.1f", id, lv[id]))
			}
		}
		fmt.Printf("%s %-12s L%-2d 적병%-2d 적장%-2d win %d/%d | %s\n", st.ID, st.Name, L, troop, officer, wins, *seeds, strings.Join(parts, " "))
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

// difficulty is the campaign difficulty every battle is fought at.
var difficulty string

// play fights stage i with the sortie at level, and returns the result, the last round,
// the levels each of the player's officers gained (fractions from experience) and who retreated.
func play(d *content.Data, i int, levels map[string]int, seed uint64, player string) (string, int, map[string]float64, []string) {
	dd := *d
	dd.Characters = maps.Clone(d.Characters)
	for id, c := range dd.Characters {
		if l, ok := levels[id]; ok && c.Playable {
			c.Level = l
			dd.Characters[id] = c
		}
	}
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == d.Stages[i].ID {
			dd.Campaign.Start = n.ID
		}
	}
	dd.Campaign.Roster, dd.Campaign.Warehouse = sortie(d, d.Stages[i])
	dd.Campaign.Deployment = dd.Campaign.Roster
	e, err := core.NewGame(&dd, seed, difficulty)
	check(err)
	e.SetDamagePct(storage.DefaultScalePct)
	e.SetXPPct(storage.DefaultScalePct)
	show := false
	for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() {
		_, err := e.Apply(c)
		check(err)
	}
	_, err = e.Apply(core.Command{Kind: "start"})
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
		switch player {
		case "rush":
			next = ai.RushNext
		case "idle":
			next = idleNext
		}
		c, err := next(e)
		check(err)
		r, err := e.Apply(c)
		check(err)
		for _, ev := range r.Events {
			if show && (ev.Kind == "heal" || ev.Kind == "item" || ev.Kind == "recover-hp" || ev.Kind == "pouch-heal" || ev.Kind == "retreat" || ev.Kind == "death" || ev.Kind == "duel" || ev.Kind == "speech" || ev.Kind == "damage" || ev.Kind == "critical") {
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
					} else if u.ID == ev.Actor && u.Faction != "enemy" && ev.Text != "flight" { // an escort beaten, not one that got away
						retreats = append(retreats, "npc:"+u.ID)
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

// sortie is who goes out to stage st: the officers it requires, then the starting roster and everyone the
// story joins before its battle, up to the stage's limit; and the warehouse: the starting one with every piece
// of equipment the story grants before the battle, its explore talks and bonus objectives included.
func sortie(d *content.Data, st content.Stage) ([]string, map[string]int) {
	joined := append([]string{}, d.Campaign.Roster...)
	warehouse := maps.Clone(d.Campaign.Warehouse)
	if warehouse == nil {
		warehouse = map[string]int{}
	}
	grant := func(n content.Node) {
		if _, ok := d.Equipment[n.Item]; ok && n.Kind == "grant" {
			warehouse[n.Item] += n.Count
		}
	}
chapters:
	for _, c := range d.Chapters {
		for _, id := range c.Nodes {
			n := d.Node(id)
			if n.Kind == "battle" && n.Stage == st.ID {
				break chapters
			}
			grant(n)
			if n.Kind == "explore" {
				for _, ev := range c.Events {
					grant(d.Node(ev))
					if x := d.Node(ev); x.Kind == "join" && !slices.Contains(joined, x.Character) {
						joined = append(joined, x.Character)
					}
				}
			}
			if n.Kind == "join" {
				joined = append(joined, n.Character)
			}
			if n.Kind == "leave" {
				joined = slices.DeleteFunc(joined, func(id string) bool { return id == n.Character })
			}
		}
	}
	var ids []string
	for _, id := range append(append([]string{}, st.Required...), joined...) {
		if slices.Contains(joined, id) && !slices.Contains(ids, id) && !slices.Contains(st.Reserve, id) && len(ids) < st.Max {
			ids = append(ids, id)
		}
	}
	for _, id := range st.Reserve { // they sortie on top of the count
		if slices.Contains(joined, id) {
			ids = append(ids, id)
		}
	}
	return ids, warehouse
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// idleNext lets the player's officers stand where they were deployed; everyone else plays as ai.Next does.
func idleNext(e *core.Engine) (core.Command, error) {
	o := e.Observe()
	if o.Phase == "battle" && len(o.Learning) == 0 {
		for _, u := range o.UnitViews {
			if u.ActsIn(o.Turn, o.Army) && u.HP > 0 && !u.Done && u.Faction == "ally" && !u.AI {
				return core.Command{Kind: "wait", Actor: u.ID}, nil
			}
		}
	}
	return ai.Next(e)
}
