// Temporary tool: walks the story from its start with the game AI as the player, retrying a lost
// battle up to five times, and prints the playable officers' levels against each stage's level, so
// the growth carried from battle to battle can be read, with what the lord did besides moving in the
// battle won (or last lost).
//
//	go run ./cmd/_campaign -seed 1 -stages 7
//	go run ./cmd/_campaign -difficulty hard -immortal liubei -stages 20 -report   # every unit's level, class and weapon
//
// Before each battle the officers whose level allows it are promoted with the seals the party carries, highest
// level first, into the first class the seal offers.
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
	"srpg/internal/storage"
)

func main() {
	dir := flag.String("content", "assets/data", "content directory")
	seed := flag.Uint64("seed", 1, "seed")
	stages := flag.Int("stages", 7, "battles to fight")
	xp := flag.Int("xp", storage.DefaultScalePct, "experience scale in percent, as the settings set it")
	difficulty := flag.String("difficulty", "normal", "difficulty id (game.json Difficulties)")
	subs := flag.Bool("subs", false, "after each victory, grant the rewards of the bonus objectives the AI missed (seals for -report)")
	report := flag.Bool("report", false, "print every unit's level, class and weapon when each battle starts")
	immortal := flag.String("immortal", "", "characters whose units never fall, comma-separated, or ally for the whole side: to follow the campaign past battles the AI would lose")
	eval := flag.String("eval", "", "other weights for the player's units as n,m,threat,top,w (ai.ParseEval)")
	flag.Parse()
	if *eval != "" {
		x, err := ai.ParseEval(*eval)
		check(err)
		ai.Player = x
	}
	d, err := content.Load(*dir)
	check(err)
	e, err := core.NewGame(d, *seed, *difficulty)
	check(err)
	e.SetDamagePct(storage.DefaultScalePct)
	e.SetXPPct(*xp)
	e.SetImmortal(*immortal)
	apply := func(c core.Command) {
		r, err := e.Apply(c)
		check(err)
		if os.Getenv("CAMPAIGN_DEBUG") != "" {
			for _, ev := range r.Events {
				if ev.Kind == "retreat" || ev.Kind == "turn-limit" || ev.Kind == "duel" || ev.Target == *immortal || ev.Actor == *immortal {
					fmt.Printf("        r%d %s %s %s %d %s\n", e.Observe().Round, ev.Kind, ev.Actor, ev.Target, ev.Amount, ev.Text)
				}
			}
		}
	}
	levels := func() string {
		s := ""
		for _, c := range e.Observe().Characters {
			if d.Characters[c.Def].Playable {
				if *report {
					s += fmt.Sprintf(" %s %d %s", c.ID, c.Level, d.Classes[c.Class].Name)
				} else {
					s += fmt.Sprintf(" %s %d", c.ID, c.Level)
				}
			}
		}
		return s
	}
	fought := 0
	for step := 0; step < 5000 && fought < *stages; step++ {
		o := e.Observe()
		switch {
		case o.Phase == "scenario" && (o.Dialogue.Kind == "say" || o.Dialogue.Kind == "narrate"):
			apply(core.Command{Kind: "next"})
		case e.Walking():
			for _, c := range e.ExploreNext() {
				apply(c)
			}
		case o.Phase == "scenario" && o.Dialogue.Kind == "choice":
			options := []string{}
			for k := range o.Dialogue.Choices {
				options = append(options, k)
			}
			sort.Strings(options)
			apply(core.Command{Kind: "choose", Option: options[0]})
		case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
			st := d.Stages[o.Stage]
			fmt.Printf("%s Lv%-2d before:%s\n", st.ID, d.Rules.StageLevel(st.Order), levels())
			promote(d, e, apply)
			for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() {
				apply(c)
			}
			before := e.Snapshot()
			tries := 1
			var acts map[string]int
			for {
				acts = map[string]int{}
				apply(core.Command{Kind: "start"})
				if *report && tries == 1 {
					printUnits(d, e)
				}
				for i := 0; i < 20000 && e.Observe().Phase != "result"; i++ {
					c, err := ai.Next(e)
					check(err)
					if d.Characters[c.Actor].Lord && c.Kind != "move" && c.Kind != "place" {
						acts[strings.Trim(c.Kind+" "+c.Skill+c.Item, " ")]++
					}
					apply(c)
				}
				if e.Observe().Result == "victory" || tries == 5 {
					break
				}
				before.RNG += 0x9e3779b97f4a7c15 // a retry plays out differently
				e, err = core.Restore(d, before)
				if err != nil {
					R := d.Rules
					for _, o := range before.Characters {
						if t := d.Classes[o.Class].Tier; !o.Placement && t > 0 && o.Level < R.PromoteLevels[t-1] {
							fmt.Printf("    bad tier: %s Lv%d %s (t%d)\n", o.ID, o.Level, o.Class, t)
						}
						for slot, eq := range o.Equipment {
							if !d.Carries(d.Classes[o.Class], o.Learned, d.Equipment[eq]) {
								fmt.Printf("    bad gear?: %s %s %s %s\n", o.ID, o.Class, slot, eq)
							}
						}
					}
				}
				check(err)
				e.SetImmortal(*immortal)
				tries++
			}
			fmt.Printf("    %s after %d tries (round %d):%s  lord %v\n", e.Observe().Result, tries, e.Observe().Round, levels(), acts)
			fought++
			if e.Observe().Result != "victory" {
				printUnits(d, e)
				return
			}
			for c, ok := ai.Learning(e); ok; c, ok = ai.Learning(e) {
				apply(c)
			}
			if *subs {
				grantSubs(e, st, e.Snapshot().Executed)
			}
			apply(core.Command{Kind: "continue"})
		default:
			check(fmt.Errorf("stuck in phase %s at node %s (%s)", o.Phase, o.Node, o.Dialogue.Kind))
		}
	}
}

// promote spends the party's seals on the officers they fit, highest level first.
func promote(d *content.Data, e *core.Engine, apply func(core.Command)) {
	o := e.Observe()
	officers := append([]core.Character(nil), o.Characters...)
	sort.SliceStable(officers, func(i, j int) bool { return officers[i].Level > officers[j].Level })
	items := []string{}
	for id, it := range d.Items {
		if it.Effect == content.ItemPromote {
			items = append(items, id)
		}
	}
	sort.Strings(items)
	for _, item := range items {
		for _, c := range officers {
			if e.Observe().Inventory[item] < 1 || !d.Characters[c.Def].Playable {
				continue
			}
			if classes, err := e.PromotionsAway(item, c.ID); err == nil {
				from := e.Observe()
				apply(core.Command{Kind: "item", Target: c.ID, Item: item, Option: classes[0]})
				for _, x := range from.Characters {
					if x.ID == c.ID {
						fmt.Printf("    promote %s Lv%d %s -> %s (%s)\n", c.ID, x.Level, x.Class, classes[0], item)
					}
				}
			}
		}
	}
}

// grantSubs gives the party the rewards of the stage's bonus objectives it did not meet, as if it had.
func grantSubs(e *core.Engine, st content.Stage, done map[string]bool) {
	subs := append([]content.Objective(nil), st.Sub...)
	for _, ev := range st.Events {
		for _, a := range ev.Do {
			subs = append(subs, a.Sub...)
		}
	}
	for _, o := range subs {
		if done["sub-"+st.ID+"-"+o.ID] {
			continue
		}
		done["sub-"+st.ID+"-"+o.ID] = true
		for item, n := range o.Reward {
			e.SimGrant(item, n)
			fmt.Printf("    sub %s missed: +%s ×%d\n", o.ID, item, n)
		}
	}
}

// printUnits lists the field when the battle starts: each side's units with level, class (tier) and weapon.
func printUnits(d *content.Data, e *core.Engine) {
	o := e.Observe()
	views := append([]core.UnitView(nil), o.UnitViews...)
	sort.SliceStable(views, func(i, j int) bool {
		if views[i].Faction != views[j].Faction {
			return views[i].Faction < views[j].Faction
		}
		return views[i].Level > views[j].Level
	})
	weapons := map[string]string{}
	for _, c := range o.Characters {
		for _, id := range c.Equipment {
			if eq := d.Equipment[id]; eq.Slot == "weapon" {
				weapons[c.ID] = eq.Name
			}
		}
	}
	for _, u := range views {
		cl := d.Classes[u.Class]
		fmt.Printf("      %-7s %-20s Lv%-2d %-8s t%d %-9s HP%-5d %s\n", u.Faction, u.ID, u.Level, cl.Name, cl.Tier, u.Rank, u.HP, weapons[u.Character])
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if os.Getenv("CAMPAIGN_DEBUG") != "" {
			panic(err)
		}
		os.Exit(1)
	}
}
