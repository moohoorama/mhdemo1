package core

import "srpg/internal/content"

// faceObjective turns a unit placed without a facing toward what it is there for, so that a battle
// opens with every side looking where its goal lies: a marching or fleeing unit at its goal cells,
// an ally at what the victory goals name (officers to defeat, an area to reach), an enemy at what
// the defeat goals name, and anyone else at the nearest opponents. It looks from the middle of band,
// the cells of those it stands with (its group when band is empty), so a line faces one way.
func (e *Engine) faceObjective(u *Unit, band []content.Point) {
	if u.Facing != "" {
		return
	}
	if len(band) == 0 && u.Group != "" {
		for _, v := range e.matching(u.Group) {
			if v.HP > 0 {
				band = append(band, content.Point{X: v.X, Y: v.Y})
			}
		}
	}
	at := content.Point{X: u.X, Y: u.Y}
	if len(band) > 0 {
		sx, sy := 0, 0
		for _, p := range band {
			sx, sy = sx+p.X, sy+p.Y
		}
		at = content.Point{X: (2*sx + len(band)) / (2 * len(band)), Y: (2*sy + len(band)) / (2 * len(band))}
	}
	best, score := at, -1
	for _, p := range e.objectiveCells(u) {
		if n := dist(at.X, at.Y, p.X, p.Y); n > 0 && (score < 0 || n < score) {
			best, score = p, n
		}
	}
	if score > 0 {
		u.Facing = content.Toward(at, best)
	}
}

// objectiveCells are the cells faceObjective turns a unit toward. The first win (or, for an enemy,
// loss) goal that names someone standing or an area of the unit's side is the one taken.
func (e *Engine) objectiveCells(u *Unit) []content.Point {
	stage := e.data.Stages[e.state.Stage]
	if u.Goal != nil || u.GoalArea != 0 {
		return stage.GoalCells(u.Goal, u.GoalArea)
	}
	goals := stage.Victory
	if u.Faction == "enemy" {
		goals = stage.Defeat
	}
	for _, g := range goals {
		var cells []content.Point
		for _, s := range g.If {
			c, err := content.ParseCond(s)
			if err != nil || c.Not {
				continue
			}
			switch c.Op {
			case "defeated", "alldefeated":
				for _, v := range e.matching(c.Arg) {
					if v.HP > 0 && v.Faction != u.Faction {
						cells = append(cells, content.Point{X: v.X, Y: v.Y})
					}
				}
			case "at", "allat":
				if e.sideOf(c.Arg) == u.Faction {
					cells = append(cells, stage.GoalCells(nil, c.N)...)
				}
			}
		}
		if len(cells) > 0 {
			return cells
		}
	}
	var cells []content.Point
	for _, id := range keys(e.unitIDs) {
		if v := e.unit(id); v.HP > 0 && v.Faction != u.Faction {
			cells = append(cells, content.Point{X: v.X, Y: v.Y})
		}
	}
	if u.Faction == "enemy" { // before deployment the officers' cells stand for them
		cells = append(cells, stage.Allies...)
	}
	return cells
}

// sideOf is the side a condition's subject fights on: a side by name, or that of the units it names;
// an officer not on the map yet is taken for an ally.
func (e *Engine) sideOf(target string) string {
	if target == "enemy" {
		return "enemy"
	}
	if units := e.matching(target); len(units) > 0 {
		return units[0].Faction
	}
	return "ally"
}
