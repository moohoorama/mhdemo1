package gui

import (
	"fmt"
	"maps"
	"math"
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/mapedit"
	"srpg/internal/scenes"
)

func init() { demos["suigong"] = demoSuigong }

// The three-second flood's ground (tools/floodsim.py simulates the same layout): the river, six cells wide, runs
// down the v axis from the screen's upper right to its lower left. Its two places lie too far apart to show
// together: up the river a gorge of the map's own cliff tiles with the dam of planks and earth across it (the dam
// is the baked surge's), far down it the ford where the army wades across.
const (
	suigongW, suigongH = 48, 44
	suigongRiver       = 12 // the river's first column
	suigongWidth       = 6  // its columns
	suigongDam         = 8  // the row the dam stands across; the gorge's cliffs are the rows above it and the dam's two
	suigongFord        = 30 // the ford's first row, and every row below it
	suigongCliffs      = 3  // columns of cliff on the left of the river; on its right they run to the map's end of the gorge
)

// suigongGround is the ground letter of cell (u, v).
func suigongGround(u, v int) byte {
	du := u - suigongRiver
	switch {
	case du >= 0 && du < suigongWidth && v >= suigongFord:
		return 's'
	case du >= 0 && du < suigongWidth:
		return 'w'
	case v <= suigongDam+1 && (du >= -suigongCliffs && du < 0 || du >= suigongWidth && du < suigongWidth+5):
		return 'h'
	}
	return 'g'
}

// suigongFordCam is the cell the ford shot looks at.
const suigongFordCamU, suigongFordCamV = suigongRiver + suigongWidth/2, suigongFord + .8

// The army marches up the river's width (NW, toward lower u) as a scene's walkers do: cell after cell, each in
// stepTime with the walk motion at its own pace and a footstep or hoof beat as each cell begins (marchCell). It
// marches while the ford is on screen: about marchLead cells during the opening shot (until it sinks to black), so
// the lattice the army stood on before is roughly where it is when the view cuts back to the ford, and on at the
// same pace until the water takes each soldier or it cries out.
const (
	marchLead  = 11
	marchSpeed = 1 / stepTime
	marchAfter = 2.4 // seconds of marching after the cut that the cells entering the screen are counted for
)

// marchPhase is how far into its first cell a soldier of row v starts, so that the rows do not step as one.
func marchPhase(v int) float64 { return float64(v*7%10) / 11 }

// suigongArmy are the soldiers' cells at the cut back to the ford: every cell of the ford's rows that the 4x ford
// shot shows (and a little past its left and bottom edges, where the heads of the soldiers below it still rise into
// view), and every one that marches into it before the flood, so that the screen holds the north-east corner of a
// host that goes on beyond it. The soldiers stand marchLead cells further down u at the start. rides marks the horsemen.
func suigongArmy() []struct {
	u, v  int
	rides bool
} {
	var out []struct {
		u, v  int
		rides bool
	}
	seen := func(u float64, v int) bool {
		dx, dy := (u-suigongFordCamU-(float64(v)-suigongFordCamV))*16, (u-suigongFordCamU+(float64(v)-suigongFordCamV))*8
		return math.Abs(dx) <= 206 && dy >= -136 && dy <= 176
	}
	for v := suigongFord; v < suigongH; v++ {
		for u := range suigongW { // its start cell; at the cut it stands marchLead cells down
			lc := u - marchLead
			if lc == suigongRiver+2 && v == suigongFord+1 {
				continue
			}
			ok := false
			for k := 0.0; k <= 1; k += .125 { // along its march: from the start to marchAfter seconds past the cut
				ok = ok || seen(float64(lc)+marchLead*(1-k)-marchSpeed*marchAfter*k, v)
			}
			if !ok {
				continue
			}
			out = append(out, struct {
				u, v  int
				rides bool
			}{u, v, (lc*7+v*13)%9 == 0})
		}
	}
	return out
}

// suigongCries are what the soldiers who see the water coming shout, turning up the river: where each stands (cell
// of the army), when (seconds after the cut to the ford) and what.
var suigongCries = []struct {
	u, v int
	at   float64
	text string
}{
	{suigongRiver + 1, suigongFord, .2, "물이다!"},
	{suigongRiver + 4, suigongFord + 1, .4, "피해라!"},
}

// The flood's timeline, in seconds of the capture (60 frames a second). The ford shows first, sinks to black, and
// the screen comes back on 관우 on the cliff, who orders the dam broken; once his bubble has gone the view scrolls
// to the dam in scrollSecs, the screen shakes under the sound of a boulder, flashes white at the burst and clears in
// flashSecs as the dam goes; from the burst on everything plays surgeRate times as fast as the surge was baked
// (floodsim.py SLOW): the dam for the surge's own cut (2.5 s baked, 1.25 s shown), then the ford for the rest of the
// surge (4.4 s baked, 2.2 s shown) while the water, the white and the sound of the flood swell: the white takes the
// last of the water (fordWater, fordWhite) and is full as the water's last picture shows.
const (
	tFadeOut, tBlack, tFadeIn = 2.5, 3.2, 3.9
	tOrder                    = 4.1
	tHush                     = 6.6                // the order's bubble is dropped (it fades over speechFade)
	tScroll                   = tHush + speechFade // the view starts to scroll to the dam
	scrollSecs                = 2.0
	tShake                    = tScroll + scrollSecs // the screen shakes and the boulder sounds
	shakeSecs                 = 1.8
	tBurst                    = tShake + shakeSecs // the white flash, the dam bursts, the water sounds
	flashSecs                 = .3
	aftershock                = .6  // seconds the shaking dies away after the burst
	surgeBreak                = 1.0 // the surge's own seconds the standing dam shows before it bursts
	surgeRate                 = 2.0 // the surge's own seconds that pass in one second from the burst on
	fordWater                 = 1.6 // seconds after the cut the water reaches the screen's edge and the white starts to come in
	fordWhite                 = .6  // seconds the white takes to fill the screen: the last moving picture of the water (2.2 s after the cut) shows white
	fordHold                  = .8  // seconds the white stays after
)

// demoSuigong plays the flood (suigongDemoGroup) as a battle's first picture, on a battle map (the story plays it in a
// scene, the step "Surge"), in two shots at twice zoom
// that never show the same ground: the ford where the army wades, then the dam and 관우 on the cliff over it, who
// orders it broken; the view scrolls to the dam, which shakes, flashes and bursts; the view cuts to the ford, where
// the army's north-east corner fills the left and bottom of the screen and soldiers turn and cry out as the water
// comes down on them, and the screen goes white. Its timeline is the constants above. -stage B25 (신야), 60 frames
// a second, with the sound's trace for tools/sfx/dub.py:
//
//	SRPG_SFX_TRACE=reports/uidemo/suigong.sfx go run ./cmd/_uidemo-vm -ui suigong -stage B25 -frames 937 -mp4 reports/uidemo/suigong.mp4
//	python3 tools/sfx/dub.py reports/uidemo/suigong.sfx reports/uidemo/suigong.mp4 reports/uidemo/suigong_dub.mp4 --no-music
func demoSuigong(d *Demo) error {
	g := d.g
	name, group := g.suigongDemoGroup()
	if name == "" {
		return fmt.Errorf("suigong: no scene plays a flood")
	}
	var leader string
	for _, st := range group {
		if st.Kind == "surge" {
			leader = st.Unit
		}
	}
	err := d.battleWith(d.stage, func(doc *mapedit.Document) { // the flood's own ground in place of the stage's map
		data := g.S.Data
		if err := doc.Resize(mapedit.Rect{X: 0, Y: 0, W: suigongW, H: suigongH}); err != nil {
			panic(err)
		}
		G, S, T := byte(content.LayerGround), byte(content.LayerStructure), byte(content.LayerState)
		doc.Visual = mapedit.Rect{X: -4, Y: -4, W: suigongW + 8, H: suigongH + 8}
		for y := -4; y < suigongH+4; y++ {
			for x := -4; x < suigongW+4; x++ {
				doc.Paint(data, x, y, S, '-')
				doc.Paint(data, x, y, T, '-')
				doc.Paint(data, x, y, G, suigongGround(x, y))
			}
		}
		doc.Props, doc.Decorations = nil, nil
		doc.Stage.Events, doc.Stage.Duels, doc.Stage.Recommend, doc.Stage.Areas, doc.Stage.Sub, doc.Stage.Groups = nil, nil, nil, nil, nil, nil
		doc.Stage.Forced, doc.Stage.Night = true, false
		doc.Stage.Allies = nil
		for i := range 10 {
			doc.Stage.Allies = append(doc.Stage.Allies, content.Point{X: suigongW - 1 - i%3, Y: suigongH - 1 - i/3})
		}
		for i := range doc.Stage.Enemies { // the stage's goals keep its officers; they and the soldiers stand off the flood's shots
			e := &doc.Stage.Enemies[i]
			e.X, e.Y, e.Policy, e.GoalArea, e.Group = suigongW-1-i%3, i/3, "hold", 0, ""
		}
		if !slices.ContainsFunc(doc.Stage.Enemies, func(e content.Spawn) bool { return e.Character == leader }) {
			doc.Stage.Enemies = append(doc.Stage.Enemies, content.Spawn{Character: leader, X: suigongRiver - 1, Y: suigongFord, Facing: "NW", Policy: "hold"})
		}
	})
	if err != nil {
		return err
	}
	g.beginSuigong(name, group)
	if g.show == nil {
		msg := ""
		for _, t := range g.toasts {
			msg += t.text
		}
		return fmt.Errorf("suigong: the flood did not begin: %s", msg)
	}
	return nil
}

// marchCell sends a soldier on toward the next cell NW as a scene's walker goes: in stepTime a cell, the walk motion
// playing at its own pace, a footstep (a hoof beat for a rider) as it begins, and the share the last cell ran over
// carried into this one. A soldier standing on its start cell begins phase of the way into its first cell.
func (g *Game) marchCell(u *unitVis, from, phase float64) {
	over, first := u.over, u.u == from
	if first {
		u.u -= phase
	}
	cue := "step"
	if u.mounted {
		cue = "hoof"
	}
	g.sfxVolAtCell(cue, u.u, u.v, 1)
	to := math.Floor(u.u - 1e-6)
	g.walkTo(u, to, u.v)
	if dist := u.u - to; dist < 1 {
		u.walk.dur = dist * stepTime
	}
	u.walk.t = over * math.Max(u.walk.dur, stepTime*math.Min(1, u.walk.dur))
	if first {
		u.animT = phase * stepTime
	}
}

// suigongDemoGroup is the flood of the story's scene (the step "Surge" of the scenes) as the cast of a battle
// cutscene's group, for the dev demo on a battle map: the step's actor leads the army, the rest of the scene's cast
// stand on the cliff, and the scene's Say steps are the bubbles at their After seconds.
func (g *Game) suigongDemoGroup() (string, []core.CutStep) {
	for _, sc := range g.scenes.Scenes {
		for _, lines := range sc.Beats {
			for _, groups := range lines {
				for _, group := range groups {
					for _, st := range group {
						if st.Surge == "" {
							continue
						}
						id := func(key string) string { id, _, _ := g.character(scenes.Base(key)); return id }
						out := []core.CutStep{{Kind: "surge", Unit: id(st.Actor), Text: st.Surge}}
						for _, key := range slices.Sorted(maps.Keys(sc.Cast)) {
							if key != st.Actor && !scenes.Extra(key) {
								out = append(out, core.CutStep{Kind: "face", Unit: id(key), Facing: sc.Cast[key].Face, NoFocus: true})
							}
						}
						for _, say := range group {
							if say.Say != "" {
								out = append(out, core.CutStep{Kind: "say", Unit: id(say.Actor), Text: say.Say, Seconds: say.After, NoFocus: true})
							}
						}
						return st.Surge, out
					}
				}
			}
		}
	}
	return "", nil
}
