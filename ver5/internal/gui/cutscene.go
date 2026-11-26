package gui

import (
	"image"
	"math"
	"sort"

	"srpg/internal/content"
	"srpg/internal/core"
)

// timedFn is something to do at a moment (seconds from the start of a stretch of beats).
type timedFn struct {
	at float64
	fn func()
}

// addTimeline lays steps that happen at their own moments, some together, out as the player's
// sequential beats; total is when the stretch ends.
func addTimeline(add func(float64, func()), entries []timedFn, total float64) {
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].at < entries[j].at })
	now := 0.0
	for _, e := range entries {
		if e.at > now {
			add(e.at-now, func() {})
			now = e.at
		}
		add(0, e.fn)
	}
	if total > now {
		add(total-now, func() {})
	}
}

func timelineEnd(entries []timedFn) float64 {
	end := 0.0
	for _, e := range entries {
		end = max(end, e.at)
	}
	return end + stepTime
}

// walkTimeline has a unit walk a path cell by cell from the moment start, then stand at rest turned
// to facing (none: the way its last step went).
func (g *Game) walkTimeline(id string, path []content.Point, facing string, start float64) []timedFn {
	if len(path) < 1 {
		return nil
	}
	var out []timedFn
	for i, cell := range path[1:] {
		cell := cell
		out = append(out, timedFn{start + float64(i)*stepTime, g.onUnit(id, func(v *unitVis) { g.stepTo(v, cell) })})
	}
	out = append(out, timedFn{start + float64(len(path)-1)*stepTime, g.onUnit(id, func(v *unitVis) {
		if facing != "" {
			v.dir = facing
		}
		v.play(v.rest())
	})})
	return out
}

// onUnit runs f on the unit as it stands when the beat comes: a unit that arrives earlier in the same
// replay is not on the field yet when the timeline is built.
func (g *Game) onUnit(id string, f func(*unitVis)) func() {
	return func() {
		if v := g.unitAt(id); v != nil {
			f(v)
		}
	}
}

// walkCells are the cells the camera shows a walk on: where it starts and ends, or only where it
// starts when the two do not fit on one screen (the camera then follows the walker, followWalk).
func (g *Game) walkCells(path []content.Point) []content.Point {
	n := len(path)
	if n == 0 || g.field == nil {
		return nil
	}
	both := []content.Point{path[0], path[n-1]}
	if x, y := g.actionCenter(both); g.onScreenAt(both, x, y) {
		return both
	}
	return both[:1]
}

// cutsceneTimeline turns a cutscene's groups into one timeline: a group's steps start together
// and the next group begins when the longest of them is done. A group whose people are off screen
// first glides the camera to them.
func (g *Game) cutsceneTimeline(groups [][]core.CutStep) (entries []timedFn, total float64) {
	cam := [2]float64{g.camX, g.camY}
	at := map[string]content.Point{} // where the cutscene has walked its people so far
	for _, group := range groups {
		var cells []content.Point
		for _, st := range group {
			switch st.Kind {
			case "move", "leave":
				if !st.NoFocus {
					cells = append(cells, g.walkCells(st.Path)...)
				}
				if n := len(st.Path); n > 0 {
					at[st.Unit] = st.Path[n-1]
				}
			case "play", "face", "say":
				if st.NoFocus {
					continue
				}
				if p, ok := at[st.Unit]; ok {
					cells = append(cells, p)
				} else if v := g.unitAt(st.Unit); v != nil {
					cells = append(cells, content.Point{X: int(math.Round(v.u)), Y: int(math.Round(v.v))})
				}
			}
		}
		glide, wait := g.glideFirst(&cam, cells, total)
		entries = append(entries, glide...)
		total += wait
		longest := 0.0
		for _, st := range group {
			st := st
			length := 0.0
			switch st.Kind {
			case "move":
				entries = append(entries, g.walkTimeline(st.Unit, st.Path, st.Facing, total)...)
				length = float64(max(0, len(st.Path)-1)) * stepTime
				if st.NoFocus {
					entries = append(entries, timedFn{total, g.onUnit(st.Unit, func(v *unitVis) { v.noFocus = true })},
						timedFn{total + length, g.onUnit(st.Unit, func(v *unitVis) { v.noFocus = false })})
				}
			case "play":
				entries = append(entries,
					timedFn{total, g.onUnit(st.Unit, func(v *unitVis) { v.play(st.Motion) })},
					timedFn{total + cutPlayTime(st), g.onUnit(st.Unit, func(v *unitVis) { v.play(v.rest()) })})
				length = cutPlayTime(st)
			case "face":
				entries = append(entries, timedFn{total, g.onUnit(st.Unit, func(v *unitVis) { v.dir = st.Facing })})
			case "say":
				lines := []content.Speech{{Actor: st.Unit, Text: st.Text}}
				length = max(2, 1.5+float64(len([]rune(st.Text)))*.09)
				entries = append(entries,
					timedFn{total, func() {
						g.showBattleSpeech(lines)
						g.logPending = append(g.logPending, g.name(st.Unit)+": "+st.Text)
					}},
					timedFn{total + length, func() { g.dropSpeech() }})
			case "leave":
				entries = append(entries, g.walkTimeline(st.Unit, st.Path, "", total)...)
				walked := float64(max(0, len(st.Path)-1)) * stepTime
				dir := image.Point{}
				if n := len(st.Path); n > 1 {
					dir = image.Pt(st.Path[n-1].X-st.Path[n-2].X, st.Path[n-1].Y-st.Path[n-2].Y)
				}
				entries = append(entries, timedFn{total + walked, g.onUnit(st.Unit, func(v *unitVis) { g.leaveField(v, dir) })})
				length = walked + leaveTime
			case "look":
				if n := len(st.Path); n > 0 && g.field != nil {
					p := st.Path[0]
					x, y := g.field.Center(float64(p.X), float64(p.Y))
					cam = [2]float64{x, y}
					entries = append(entries, timedFn{total, func() { g.panToCell(p.X, p.Y) }})
					length = panTime + st.Seconds
				}
			case "dust":
				if n := len(st.Path); n > 0 && g.field != nil {
					p := st.Path[0]
					x, y := g.field.Center(float64(p.X), float64(p.Y))
					cam = [2]float64{x, y}
					entries = append(entries,
						timedFn{total, func() { g.panToCell(p.X, p.Y) }},
						timedFn{total + panTime, func() {
							g.dustAt(float64(p.X), float64(p.Y), g.field.Lift(p.X, p.Y))
						}})
					length = panTime + st.Seconds
				}
			case "fall":
				entries = append(entries, timedFn{total, func() {
					g.strike(nil, []core.Event{{Kind: "retreat", Actor: st.Unit}})
				}})
				length = blinkTime
			case "wait":
				length = st.Seconds
			}
			longest = max(longest, length)
		}
		total += longest
	}
	return entries, total
}

// cutPlayTime is how long a motion held by a cutscene step takes.
func cutPlayTime(st core.CutStep) float64 {
	if st.Seconds > 0 {
		return st.Seconds
	}
	return .7
}
