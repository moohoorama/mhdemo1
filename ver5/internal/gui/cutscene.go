package gui

import (
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
	v := g.unitAt(id)
	if v == nil || len(path) < 1 {
		return nil
	}
	var out []timedFn
	for i, cell := range path[1:] {
		cell := cell
		out = append(out, timedFn{start + float64(i)*stepTime, func() { g.stepTo(v, cell) }})
	}
	out = append(out, timedFn{start + float64(len(path)-1)*stepTime, func() {
		if facing != "" {
			v.dir = facing
		}
		v.play(v.rest())
	}})
	return out
}

// cutsceneTimeline turns a cutscene's groups into one timeline: a group's steps start together
// and the next group begins when the longest of them is done.
func (g *Game) cutsceneTimeline(groups [][]core.CutStep) (entries []timedFn, total float64) {
	for _, group := range groups {
		longest := 0.0
		for _, st := range group {
			st := st
			v := g.unitAt(st.Unit)
			length := 0.0
			switch st.Kind {
			case "move":
				entries = append(entries, g.walkTimeline(st.Unit, st.Path, st.Facing, total)...)
				length = float64(max(0, len(st.Path)-1)) * stepTime
			case "play":
				if v != nil {
					entries = append(entries,
						timedFn{total, func() { v.play(st.Motion) }},
						timedFn{total + cutPlayTime(st), func() { v.play(v.rest()) }})
				}
				length = cutPlayTime(st)
			case "face":
				if v != nil {
					entries = append(entries, timedFn{total, func() { v.dir = st.Facing }})
				}
			case "say":
				lines := []content.Speech{{Actor: st.Unit, Text: st.Text}}
				length = max(2, 1.5+float64(len([]rune(st.Text)))*.09)
				entries = append(entries,
					timedFn{total, func() {
						g.showBattleSpeech(lines)
						g.logPending = append(g.logPending, g.name(st.Unit)+": "+st.Text)
					}},
					timedFn{total + length, func() { g.dropSpeech() }})
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
