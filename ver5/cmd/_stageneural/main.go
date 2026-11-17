// Temporary tool: plays one battle node from its waypoint with the learned (neural) player, the search player or
// the rule player, a few seeds each, and prints how each ended. The model's content hash is not checked: the
// weights are carried over to the current data, so this measures a stage, not the model.
//
//	go run ./cmd/_stageneural -node c16-battle -player neural -seeds 6 -difficulty normal
package main

import (
	"flag"
	"fmt"

	"srpg/internal/autoplay"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/storage"
)

func main() {
	node := flag.String("node", "", "battle node to start at")
	kind := flag.String("player", "neural", "neural | search | rule")
	seeds := flag.Int("seeds", 4, "seeds, from 1")
	difficulty := flag.String("difficulty", "normal", "difficulty id")
	level := flag.Int("level", 0, "raise the player's officers to this level (0: the stage level); the user's own hard run reached B16 near 38")
	model := flag.String("model", "assets/autoplay/battle.json", "neural: the model")
	flag.Parse()
	d, err := content.Load("assets/data")
	check(err)
	wins := 0
	for seed := 1; seed <= *seeds; seed++ {
		var p autoplay.Player
		switch *kind {
		case "neural":
			m, err := autoplay.LoadBattleModel(*model)
			check(err)
			p = &autoplay.NeuralPlayer{Searcher: autoplay.CampaignPlayer(d), Model: m}
		case "search":
			p = autoplay.CampaignPlayer(d)
		default:
			p = &autoplay.RulePlayer{Label: "rule", D: d, G: autoplay.DefaultGenome()}
		}
		g := autoplay.DefaultGenome()
		e, err := core.NewAtWith(d, uint64(seed), *node, *difficulty, core.Start{Level: *level})
		check(err)
		e.SetDamagePct(storage.DefaultScalePct)
		e.SetXPPct(storage.DefaultScalePct)
		for range 400 {
			if len(e.Observe().Learning) == 0 {
				break
			}
			_, err := e.Apply(g.Learn(d, e))
			check(err)
		}
		p.Prepare(e, 1)
		_, err = e.Apply(core.Command{Kind: "start"})
		check(err)
		check(autoplay.Battle(e, p, 20000))
		o := e.Observe()
		if o.Result == "victory" {
			wins++
		}
		fallen := ""
		for _, u := range o.UnitViews {
			if u.Faction == "ally" && !u.AI && u.HP <= 0 {
				fallen += " " + u.ID
			}
		}
		fmt.Printf("seed %d: %s round %d, fallen:%s\n", seed, o.Result, o.Round, fallen)
	}
	fmt.Printf("%s %s %s: win %d/%d\n", *node, *kind, *difficulty, wins, *seeds)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
