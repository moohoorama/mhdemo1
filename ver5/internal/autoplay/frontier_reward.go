package autoplay

import (
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
)

// frontierProgress distinguishes failed training battles; it never counts as a stage victory.
func frontierProgress(e *core.Engine) float64 {
	o := e.Observe()
	targets := o.Map.VictoryTargets()
	left, total, targetLeft, targetTotal := 0.0, 0.0, 0.0, 0.0
	units := map[string]core.UnitView{}
	for _, u := range o.UnitViews {
		units[u.ID] = u
		if u.Faction != "enemy" {
			continue
		}
		left += float64(max(0, u.HP))
		total += float64(u.Stats.MaxHP)
		if slices.Contains(targets, u.ID) {
			targetLeft += float64(max(0, u.HP))
			targetTotal += float64(u.Stats.MaxHP)
		}
	}
	progress := 1 - left/max(1, total)
	if targetTotal > 0 {
		return max(0, min(1, .8*(1-targetLeft/targetTotal)+.2*progress))
	}
	var cells []content.Point
	var actors []string
	for _, mark := range e.GoalMarks() {
		if !mark.Win {
			continue
		}
		cells = append(cells, mark.Cells...)
		if u := units[mark.Unit]; u.HP > 0 && mine(u) {
			actors = append(actors, u.ID)
		}
	}
	if len(cells) > 0 && len(actors) > 0 {
		diag := max(1, o.Map.Width+o.Map.Height)
		near := diag
		for _, id := range actors {
			u := units[id]
			for _, c := range cells {
				near = min(near, abs(u.X-c.X)+abs(u.Y-c.Y))
			}
		}
		progress = .8*(1-float64(near)/float64(diag)) + .2*progress
	}
	return max(0, min(1, progress))
}
