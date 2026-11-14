// Temporary tool: walks the story from its start with the game AI as the player, retrying a lost
// battle up to five times, and prints the playable officers' levels against each stage's level, so
// the growth carried from battle to battle can be read, with what the lord did besides moving in the
// battle won (or last lost).
//
//	go run ./cmd/_campaign -seed 1 -stages 7
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
	apply := func(c core.Command) {
		_, err := e.Apply(c)
		check(err)
	}
	levels := func() string {
		s := ""
		for _, c := range e.Observe().Characters {
			if d.Characters[c.Def].Playable {
				s += fmt.Sprintf(" %s %d", c.ID, c.Level)
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
			for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() {
				apply(c)
			}
			before := e.Snapshot()
			tries := 1
			var acts map[string]int
			for {
				acts = map[string]int{}
				apply(core.Command{Kind: "start"})
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
				check(err)
				tries++
			}
			fmt.Printf("    %s after %d tries:%s  lord %v\n", e.Observe().Result, tries, levels(), acts)
			fought++
			if e.Observe().Result != "victory" {
				return
			}
			for c, ok := ai.Learning(e); ok; c, ok = ai.Learning(e) {
				apply(c)
			}
			apply(core.Command{Kind: "continue"})
		default:
			check(fmt.Errorf("stuck in phase %s at node %s (%s)", o.Phase, o.Node, o.Dialogue.Kind))
		}
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
