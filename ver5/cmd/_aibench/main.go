// Temporary tool: plays one battle node from its waypoint with the rule player and shows how it ended.
//
//	go run ./cmd/_aibench c06-battle 1
package main

import (
	"fmt"
	"os"
	"strconv"

	"srpg/internal/autoplay"
	"srpg/internal/content"
	"srpg/internal/core"
)

func main() {
	d, err := content.Load("assets/data")
	check(err)
	seed, _ := strconv.ParseUint(os.Args[2], 10, 64)
	g := autoplay.DefaultGenome()
	e, err := autoplay.StartAt(d, os.Args[1], seed, "hard", g)
	check(err)
	g.Prepare(d, e)
	_, err = e.Apply(core.Command{Kind: "start"})
	check(err)
	check(autoplay.Battle(e, &autoplay.RulePlayer{Label: "rule", D: d, G: g}, 20000))
	o := e.Observe()
	fmt.Println(o.Result, "round", o.Round, autoplay.Measure(d, e, autoplay.Grow(d, e)))
	for _, u := range o.UnitViews {
		fmt.Printf("  %-6s %-22s Lv%-2d HP %5d/%-5d (%d,%d)\n", u.Faction, u.ID, u.Level, u.HP, u.Stats.MaxHP, u.X, u.Y)
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
