package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"srpg/internal/core"
	"srpg/internal/storage"
)

func init() { demos["saves"] = demoSaves }

// demoSaves shows battle autosaves (frame 60), scenario autosaves (120), manual saves (180) and the save window (240).
func demoSaves(d *Demo) error {
	g := d.g
	st := g.S.Store
	data := g.S.Data
	battleNode := ""
	for _, n := range data.Nodes {
		if n.Kind == "battle" && n.Stage == d.stage {
			battleNode = n.ID
			break
		}
	}
	e, err := core.NewAt(data, 1, battleNode, "")
	if err != nil {
		return err
	}
	if e.Observe().Phase == "scenario" {
		if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
			return err
		}
	}
	plan, err := e.DeploymentPlan()
	if err != nil {
		return err
	}
	for _, c := range plan {
		if _, err := e.Apply(c); err != nil {
			return err
		}
	}
	if e.Observe().Phase != "battle" {
		return fmt.Errorf("saves demo did not enter battle %s", d.stage)
	}
	stamp := func(slot string, ago time.Duration) {
		t := time.Now().Add(-ago)
		_ = os.Chtimes(filepath.Join(st.Dir, "slot-"+slot+".json"), t, t)
	}
	g.S.Engine, g.S.Screen = core.New(data, 1), "playing"
	story := g.S.Engine.Snapshot()
	if err := st.Save("2", story, true); err != nil {
		return err
	}
	stamp("2", 50*time.Hour)
	if err := st.Autosave(story); err != nil {
		return err
	}
	stamp(storage.AutoScenario.Slots()[0], 2*time.Hour)
	battle := e.Snapshot()
	for round := 1; round <= 4; round++ {
		battle.Round = round
		if err := st.Autosave(battle); err != nil {
			return err
		}
		stamp(storage.AutoBattle.Slots()[round-1], time.Duration(5-round)*7*time.Minute)
	}
	if err := st.Save("1", battle, true); err != nil {
		return err
	}
	stamp("1", 3*time.Hour)
	d.step = func(f int) {
		switch f {
		case 10:
			g.closeOverlays()
			g.openOverlay("load")
		case 90:
			g.showSaveTab(storage.AutoScenario)
		case 150:
			g.showSaveTab(storage.Manual)
			g.saves.list.order = tableSort{"chapter", 1}
		case 210:
			g.closeOverlays()
			g.openOverlay("save")
		}
	}
	return nil
}
