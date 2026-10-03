package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"srpg/internal/session"
)

// verifyFrame exercises the rendered application's controller and persistence.
// It does not synthesize native mouse/keyboard input; that audit is separate.
func (g *Game) verifyFrame() error {
	if g.ticks < 4 {
		return nil
	}
	if g.verifyDone > 0 {
		if g.ticks > g.verifyDone+4 {
			return fmt.Errorf("verification finished")
		}
		return nil
	}
	if g.S.Engine == nil {
		g.request(session.Request{Op: "new", Seed: g.seed})
		return nil
	}
	o := g.S.Engine.Observe()
	key := fmt.Sprintf("%d-%s-%s", o.Stage, o.Phase, o.Node)
	if o.Phase == "battle" {
		key = fmt.Sprintf("%d-battle-%s", o.Stage, o.Turn)
		for _, u := range o.UnitViews {
			if u.Faction == "ally" && u.Moved && !u.Acted {
				key = fmt.Sprintf("%d-moved-before-action", o.Stage)
				break
			}
		}
	}
	if !g.verified[key] {
		g.verified[key] = true
		// Stay in this state for one draw and one saved/loaded comparison.
		before := g.S.Engine.Snapshot()
		g.request(session.Request{Op: "save", Slot: "1", Overwrite: true})
		g.request(session.Request{Op: "load", Slot: "1", Discard: true})
		if !reflect.DeepEqual(before, g.S.Engine.Snapshot()) {
			return fmt.Errorf("GUI save replay mismatch %s", key)
		}
		g.capture = true
		g.notice = "저장 재현 검증: " + key
		return nil
	}
	if o.Phase == "complete" || o.Phase == "result" && o.Result == "defeat" {
		g.verifyDone = g.ticks
		g.capture = true
		report := struct {
			Result              string
			Revision            uint64
			Boundaries          map[string]bool
			NativeInputVerified bool
			Note                string
		}{o.Phase, o.Revision, g.verified, false, "Rendered controller audit; native input requires Computer Use permission."}
		if o.Phase == "result" {
			report.Result = o.Result
		}
		b, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(filepath.Join(g.auditDir, "verification.json"), b, 0644); err != nil {
			return err
		}
		return nil
	}
	if g.ticks%2 == 0 {
		g.assist()
	}
	return nil
}
