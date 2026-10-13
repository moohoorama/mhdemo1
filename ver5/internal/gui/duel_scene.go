package gui

import (
	"fmt"

	"srpg/internal/content"
	"srpg/internal/core"
)

// startDuelScene stages a battle in the map editor's trial mode where the stage's duel
// triggers, and begins replaying it. It does not touch the campaign save.
func (g *Game) startDuelScene(stage int, duel content.Duel, enemyAttacks bool) error {
	g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
	g.editorOpen(stage)
	g.editorTrial()
	if g.editorReturn == nil {
		return fmt.Errorf("trial failed: %s", g.editor.message)
	}
	e := g.S.Engine
	if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
		return err
	}
	plan, err := e.DeploymentPlan()
	if err != nil {
		return err
	}
	for _, cmd := range plan {
		if _, err := e.Apply(cmd); err != nil {
			return err
		}
	}
	if !g.S.Data.Characters[duel.Ally].Playable {
		snap := e.Snapshot()
		for i := range snap.Units {
			if snap.Units[i].Faction == "enemy" && snap.Units[i].ID != g.S.Data.Stages[stage].Boss {
				snap.Units[i].HP = 0
			}
		}
		e, err = core.Restore(g.S.Data, snap)
		if err != nil {
			return err
		}
		g.S.Engine = e
		if _, err = e.Apply(core.Command{Kind: "wait", Actor: g.S.Data.Campaign.Deployment[0]}); err != nil {
			return err
		}
	}
	snap := e.Snapshot()
	ai, ei := -1, -1
	for i, u := range snap.Units {
		if u.Character == duel.Ally {
			ai = i
		}
		if u.Character == duel.Enemy {
			ei = i
		}
	}
	if ai < 0 || ei < 0 {
		return fmt.Errorf("missing duel participants %s", duel.ID)
	}
	st := g.S.Data.Stages[stage]
	placed := false
	if snap.Units[ei].Guard {
		snap.Units[ai].X, snap.Units[ai].Y = snap.Units[ei].X-1, snap.Units[ei].Y
		placed = true
	}
	if snap.Units[ai].AI {
		snap.Turn = "support"
	}
	for y := 1; y < st.Height-1 && !placed; y++ {
		for x := 1; x < st.Width-2 && !placed; x++ {
			free := true
			for i, u := range snap.Units {
				if i != ai && i != ei && u.HP > 0 && u.Y == y && (u.X == x || u.X == x+1) {
					free = false
				}
			}
			if free {
				snap.Units[ai].X = x
				snap.Units[ai].Y = y
				snap.Units[ei].X = x + 1
				snap.Units[ei].Y = y
				placed = true
			}
		}
	}
	if !placed {
		return fmt.Errorf("no adjacent cells")
	}
	if enemyAttacks {
		snap.Turn = "enemy"
	}
	g.S.Engine, err = core.Restore(g.S.Data, snap)
	if err != nil {
		return err
	}
	e = g.S.Engine
	before := e.Observe()
	g.ensureField(before)
	cmd := core.Command{Kind: "attack", Actor: snap.Units[ai].ID, Target: snap.Units[ei].ID}
	if enemyAttacks {
		cmd.Actor, cmd.Target = cmd.Target, cmd.Actor
	}
	r, err := e.Apply(cmd)
	if err != nil {
		return err
	}
	found := false
	for _, event := range r.Events {
		if event.Kind == "duel" {
			found = event.DuelID == duel.ID && event.Outcome == duel.Outcome
		}
	}
	if !found {
		return fmt.Errorf("attack did not trigger %s", duel.ID)
	}
	g.replayOne(before, cmd, r.Events)
	g.player.Update(stagger)
	if g.player.Duel == nil {
		return fmt.Errorf("duel screen did not start")
	}
	if g.player.Duel.Fighters[0].Name != g.name(duel.Ally) {
		return fmt.Errorf("participants reversed")
	}
	return nil
}

// PreviewDuelTransition replays the battle-to-duel transition of a stage duel in a loop,
// without touching the campaign.
func (g *Game) PreviewDuelTransition(id string) error {
	for stage, st := range g.S.Data.Stages {
		for _, d := range st.Duels {
			if d.ID == id {
				g.duelPreview = func() error { return g.startDuelScene(stage, d, false) }
				return g.duelPreview()
			}
		}
	}
	return fmt.Errorf("unknown duel %q", id)
}

func (g *Game) updateDuelPreview() {
	if g.player.Duel != nil || g.player.Busy() {
		g.previewIdle = 0
		return
	}
	if g.previewIdle++; g.previewIdle < 90 {
		return
	}
	g.previewIdle = 0
	g.endEditorTrial()
	if err := g.duelPreview(); err != nil {
		g.duelPreview = nil
		g.toast(err.Error(), bad)
	}
}
