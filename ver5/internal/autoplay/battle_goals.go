package autoplay

import (
	"slices"
	"srpg/internal/content"
	"srpg/internal/core"
)

const BattleGoalLen = 8

// EnableGoalInputs appends zero-weight observations without changing existing predictions.
func (m *BattleModel) EnableGoalInputs() {
	if m.GoalInputs {
		return
	}
	m.extendActionInputs(BattleGoalLen, BattleStateLen+BattleActionLen)
	m.GoalInputs = true
}

func (m *BattleModel) extendActionInputs(count, at int) {
	n := m.Actor
	old, next := n.Sizes[0], n.Sizes[0]+count
	w := make([]float64, next*n.Sizes[1])
	for row := range n.Sizes[1] {
		copy(w[row*next:row*next+at], n.W[0][row*old:row*old+at])
		copy(w[row*next+at+count:(row+1)*next], n.W[0][row*old+at:(row+1)*old])
	}
	n.W[0], n.Sizes[0] = w, next
	n.InMean = slices.Insert(n.InMean, at, make([]float64, count)...)
	std := make([]float64, count)
	for i := range std {
		std[i] = 1
	}
	n.InStd = slices.Insert(n.InStd, at, std...)
}

func (m *BattleModel) goalInputs(d *content.Data, e *core.Engine) BattleChoices {
	in := battleInputs(d, e, m.RichInputs)
	if !m.GoalInputs {
		return in
	}
	o := e.Observe()
	units := map[string]core.UnitView{}
	for _, u := range o.UnitViews {
		units[u.ID] = u
	}
	var cells []content.Point
	required := map[string]bool{}
	pending := false
	for _, mark := range e.GoalMarks() {
		if !mark.Win {
			continue
		}
		cells = append(cells, mark.Cells...)
		if mark.Unit != "" && mine(units[mark.Unit]) {
			required[mark.Unit] = true
		}
		pending = pending || mark.Pending
	}
	diag := max(1, o.Map.Width+o.Map.Height)
	for i, c := range in.Commands {
		g := make([]float64, BattleGoalLen)
		if len(cells) > 0 {
			u := units[c.Actor]
			x, y := u.X, u.Y
			if c.Kind == "move" {
				x, y = c.X, c.Y
			}
			dist, before := diag*2, diag*2
			var nearest content.Point
			for _, p := range cells {
				d := abs(x-p.X) + abs(y-p.Y)
				if d < dist {
					dist, nearest = d, p
				}
				before = min(before, abs(u.X-p.X)+abs(u.Y-p.Y))
			}
			g[0], g[1] = 1, b2f(required[c.Actor])
			g[2], g[3] = float64(dist)/float64(diag), float64(before-dist)/float64(max(1, d.Classes[u.Class].Move))
			g[4], g[5] = float64(nearest.X-x)/float64(max(1, o.Map.Width)), float64(nearest.Y-y)/float64(max(1, o.Map.Height))
			g[6], g[7] = b2f(dist == 0), b2f(pending)
		}
		in.Features[i] = append(in.Features[i], g...)
	}
	return in
}
