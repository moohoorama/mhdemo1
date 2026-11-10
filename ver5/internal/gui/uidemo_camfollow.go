package gui

import (
	"fmt"
	"math"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() {
	demos["camfollow"] = func(d *Demo) error { return demoCamFollow(d, 1) }
	demos["camfollow-x4"] = func(d *Demo) error { return demoCamFollow(d, 4) } // zoomed in, so the walk outruns the screen
	demos["textspeed"] = func(d *Demo) error { return demoTextSpeed(d, 1) }
	demos["textspeed-x4"] = func(d *Demo) error { return demoTextSpeed(d, 4) }
}

// demoCamFollow puts the camera on the first officer and gives the stage a cutscene, when the enemy's
// turn starts, played by the enemy officer farthest off: he looks around, walks to the officer and
// speaks. The player's turn ends at frame 20; the camera should glide to him, follow the walk, then
// show the enemy turn and come back to the lord when the player's turn returns. camfollow-x4 plays it
// at ×4 and zoomed in.
func demoCamFollow(d *Demo, speed int) error {
	g := d.g
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g.speed = speed
	if speed > 1 {
		g.setZoomImmediate(4)
	}
	st := &g.S.Data.Stages[g.S.Data.StageIndex(d.stage)]
	ally := d.firstAlly()
	d.look(ally)
	var ax, ay int
	for _, u := range g.S.View().UnitViews {
		if u.ID == ally {
			ax, ay = u.X, u.Y
		}
	}
	far, farD := "", -1.0
	for _, u := range g.S.View().UnitViews {
		if dist := math.Hypot(float64(u.X-ax), float64(u.Y-ay)); u.Faction == "enemy" && dist > farD {
			far, farD = u.ID, dist
		}
	}
	to := &content.Point{X: ax + 2, Y: ay}
	st.Events = append(st.Events, content.BattleEvent{ID: "demo-camfollow", If: []string{"turnstart:enemy"}, Do: []content.Action{{Kind: "cutscene", Steps: [][]content.Step{
		{{Kind: "face", Unit: far, Facing: "NE"}, {Kind: "wait", Seconds: .7}},
		{{Kind: "face", Unit: far, Facing: "SW"}, {Kind: "wait", Seconds: .7}},
		{{Kind: "move", Unit: far, To: to}},
		{{Kind: "say", Unit: far, Text: "멀리서 여기까지 달려왔다! 네 목을 내놓아라!"}},
		{{Kind: "wait", Seconds: 1}},
	}}}})
	e, err := core.Restore(g.S.Data, g.S.Engine.Snapshot())
	if err != nil {
		return err
	}
	g.S.Engine = e
	d.step = func(f int) {
		if f == 20 {
			g.command(core.Command{Kind: "end"})
		}
		if o := g.S.View(); f > 20 && o.Phase == "battle" && o.Turn != "ally" && !g.busy() {
			g.resolvePhase()
		}
	}
	return nil
}

// demoTextSpeed jumps to the -node waypoint (a scene, e.g. c03-start) and types it at the given replay speed: at ×4 the
// line comes out four times as fast as at ×1,
// and the auto setting turns the page after a quarter of the wait.
func demoTextSpeed(d *Demo, speed int) error {
	if d.node == "" {
		return fmt.Errorf("textspeed needs -node")
	}
	g := d.g
	g.request(session.Request{Op: "jump", Seed: 1, Node: d.node})
	for l := g.S.View().Learners; len(l) > 0; l = g.S.View().Learners {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn_close", Actor: l[0].ID}})
	}
	g.speed = speed
	g.S.Settings.AutoText = true
	d.step = func(int) { g.turnPage() }
	return nil
}
