package gui

import (
	"srpg/internal/content"
	"srpg/internal/core"
)

func init() { demos["cutscene"] = demoCutscene }

// demoCutscene gives the first stage a cutscene event that fires when the enemy's turn starts and
// ends the player's turn at frame 20: an enemy officer strides up, a blow lands, a line is spoken,
// and the scene ends with the units where the cutscene left them.
func demoCutscene(d *Demo) error {
	g := d.g
	if err := d.battle(d.stage); err != nil {
		return err
	}
	st := &g.S.Data.Stages[g.S.Data.StageIndex(d.stage)] // the trial battle works on its own copy of the data
	var enemy string
	for _, sp := range st.Enemies {
		if !g.S.Data.Characters[sp.Character].Template && enemy == "" {
			enemy = sp.Character
		}
	}
	ally := d.firstAlly()
	d.look(ally)
	var ax, ay int
	for _, u := range g.S.Engine.Observe().UnitViews {
		if u.ID == ally {
			ax, ay = u.X, u.Y
		}
	}
	to := &content.Point{X: ax + 2, Y: ay}
	st.Events = append(st.Events, content.BattleEvent{ID: "demo-scene", If: []string{"turnstart:enemy"}, Do: []content.Action{{Kind: "cutscene", Steps: [][]content.Step{
		{{Kind: "move", Unit: enemy, To: to}},
		{{Kind: "play", Unit: enemy, Motion: "attack"}, {Kind: "play", Unit: ally, Motion: "hit"}},
		{{Kind: "say", Unit: enemy, Text: "여기가 네 무덤이다!"}, {Kind: "face", Unit: ally, Facing: "NE"}},
		{{Kind: "wait", Seconds: 1}},
	}}}})
	// the engine was built before the event existed: restore it from its own snapshot with the new data
	e, err := core.Restore(g.S.Data, g.S.Engine.Snapshot())
	if err != nil {
		return err
	}
	g.S.Engine = e
	d.step = func(f int) {
		if f == 20 {
			g.command(core.Command{Kind: "end"})
		}
	}
	return nil
}
