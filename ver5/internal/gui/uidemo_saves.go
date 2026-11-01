package gui

import (
	"os"
	"path/filepath"
	"time"

	"srpg/internal/core"
)

func init() { demos["saves"] = demoSaves }

// demoSaves fills the slots with a story save, a battle save and battle autosaves, then shows the load window's
// automatic tab (frame 60), its manual tab sorted by chapter (120) and the save window (180).
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
	e, err := core.NewAt(data, 1, battleNode)
	if err != nil {
		return err
	}
	for i := 0; i < 50 && e.Observe().Phase == "scenario"; i++ {
		if _, err := e.Apply(core.Command{Kind: "next"}); err != nil {
			break
		}
	}
	plan, _ := e.DeploymentPlan()
	for _, c := range plan {
		if _, err := e.Apply(c); err != nil {
			return err
		}
	}
	g.S.Engine, g.S.Screen = e, "playing"
	stamp := func(slot string, ago time.Duration) {
		t := time.Now().Add(-ago)
		_ = os.Chtimes(filepath.Join(st.Dir, "slot-"+slot+".json"), t, t)
	}
	story := core.New(data, 1).Snapshot()
	if err := st.Save("2", story, true); err != nil {
		return err
	}
	stamp("2", 50*time.Hour)
	battle := e.Snapshot()
	for round := 1; round <= 4; round++ {
		battle.Round = round
		if err := st.Autosave(battle); err != nil {
			return err
		}
		stamp(fmtSlot("auto", round), time.Duration(5-round)*7*time.Minute)
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
			g.showSaveTab(false)
			g.saves.list.order = tableSort{"chapter", 1}
		case 150:
			g.closeOverlays()
			g.openOverlay("save")
		}
	}
	return nil
}

func fmtSlot(prefix string, n int) string { return prefix + string(rune('0'+n)) }
