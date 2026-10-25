package gui

import "srpg/internal/core"

func init() { demos["opening"] = demoOpening }

// demoOpening plays a stage's opening the way the story reaches it: the battle node is up and its
// sortie is ordered at frame 5 and the briefing read from frame 40, so a forced stage's placement and
// opening cutscene play as in the game.
func demoOpening(d *Demo) error {
	g := d.g
	data := *g.S.Data
	for _, n := range data.Nodes {
		if n.Kind == "battle" && n.Stage == d.stage {
			data.Campaign.Start = n.ID
		}
	}
	g.S.Engine = core.New(&data, 1)
	g.S.Screen = "playing"
	g.modals = nil
	d.step = func(f int) {
		if f == 5 {
			g.command(core.Command{Kind: "start"})
		}
		if f > 40 && g.briefed() {
			g.modals = g.modals[1:]
		}
	}
	return nil
}
