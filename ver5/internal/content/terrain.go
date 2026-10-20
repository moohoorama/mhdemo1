package content

import "strings"

// NoLayer fills the structure or state of a cell that has none.
const NoLayer = '-'

// Ground is what a cell stands on. Glyph is the character the map art knows it by.
type Ground struct {
	Name    string
	Glyph   string
	Surface int  `json:",omitempty"` // how the ground is drawn: 0 water, 1 grass, 2 soil
	Water   bool `json:",omitempty"` // spells that need water look for it beside the caster or target
}

// Structure stands on a ground. Grounds lists the ground glyphs it may stand on. RestHP and RestMP,
// when given, take the place of what lies under it.
type Structure struct {
	Name           string
	Glyph          string
	Grounds        string
	Ambush         bool `json:",omitempty"`
	Castle         bool `json:",omitempty"`
	RestHP, RestMP *int `json:",omitempty"` // percent of max HP restored (or lost, below zero) at turn start; multiplier of spell point regeneration
}

// State lies over a cell and is put there by events. Then is the state it turns into when its rounds
// run out (none: the cell is clear again). On, when given, lists the structure glyphs it can lie over
// (a damaged bridge); none means any cell. Look names how the picture shows it ("ember", "flame",
// "soot", "damaged", "broken"); Blaze is the size of the fire drawn on it.
type State struct {
	Name   string
	Then   string  `json:",omitempty"`
	On     string  `json:",omitempty"`
	Look   string  `json:",omitempty"`
	Blaze  float64 `json:",omitempty"`
	RestHP *int    `json:",omitempty"`
}

// TerrainRule is what one layer says about the cost and the strength of a class. A value left out is
// what the layer below says.
type TerrainRule struct {
	Factor *float64 `json:",omitempty"`
	Cost   *int     `json:",omitempty"`
}

// TerrainSet is one movement group's rules, per layer glyph.
type TerrainSet struct {
	Ground, Structure, State map[string]TerrainRule
}

// Cell is the three layers of one map cell. A structure or state that is not there is NoLayer.
type Cell struct{ Ground, Structure, State byte }

// CellWidth is how many characters of a map row one cell takes.
const CellWidth = 3

func ParseCell(s string) (Cell, bool) {
	if len(s) != CellWidth {
		return Cell{}, false
	}
	return Cell{s[0], s[1], s[2]}, true
}
func (c Cell) String() string { return string([]byte{c.Ground, c.Structure, c.State}) }

// Cell is the cell at (x, y) of the stage's rows.
func (s Stage) Cell(x, y int) Cell {
	row := s.Tiles[y]
	return Cell{row[x*CellWidth], row[x*CellWidth+1], row[x*CellWidth+2]}
}

// CellProps is what a cell does besides costing steps.
type CellProps struct {
	RestHP, RestMP        int
	Water, Ambush, Castle bool
}

// Move is what the group pays and gets on the cell: the topmost layer that says a value wins.
func (d *Data) Move(group string, c Cell) Terrain {
	set := d.Terrain[group]
	var t Terrain
	layers := []struct {
		rules map[string]TerrainRule
		glyph byte
	}{{set.Ground, c.Ground}, {set.Structure, c.Structure}, {set.State, c.State}}
	for _, l := range layers {
		r, ok := l.rules[string(l.glyph)]
		if !ok {
			continue
		}
		if r.Factor != nil {
			t.Factor = *r.Factor
		}
		if r.Cost != nil {
			t.Cost = *r.Cost
		}
	}
	return t
}

// Props puts the layers of a cell together: recovery is the topmost layer that says one, the rest
// holds if any layer has it.
func (d *Data) Props(c Cell) CellProps {
	var p CellProps
	g := d.Grounds[string(c.Ground)]
	p.Water = g.Water
	if st, ok := d.Structures[string(c.Structure)]; ok {
		p.Ambush, p.Castle = st.Ambush, st.Castle
		if st.RestHP != nil {
			p.RestHP = *st.RestHP
		}
		if st.RestMP != nil {
			p.RestMP = *st.RestMP
		}
	}
	if s, ok := d.States[string(c.State)]; ok && s.RestHP != nil {
		p.RestHP = *s.RestHP
	}
	return p
}

// CellName is how a cell is called: its structure, else its ground, and its state if it has one.
func (d *Data) CellName(c Cell) string {
	name := d.Grounds[string(c.Ground)].Name
	if st, ok := d.Structures[string(c.Structure)]; ok {
		name = st.Name
	}
	if s, ok := d.States[string(c.State)]; ok {
		name += " · " + s.Name
	}
	return name
}

// Glyph is the map art's character for the cell: its structure's, else its ground's.
func (d *Data) Glyph(c Cell) byte {
	if st, ok := d.Structures[string(c.Structure)]; ok {
		return st.Glyph[0]
	}
	return d.Grounds[string(c.Ground)].Glyph[0]
}

// ValidCell reports why a cell cannot exist: an unknown layer, or a structure on a ground it does not stand on.
func (d *Data) ValidCell(c Cell) string {
	if _, ok := d.Grounds[string(c.Ground)]; !ok {
		return "unknown ground " + string(c.Ground)
	}
	if c.Structure != NoLayer {
		st, ok := d.Structures[string(c.Structure)]
		if !ok {
			return "unknown structure " + string(c.Structure)
		}
		if !strings.Contains(st.Grounds, string(c.Ground)) {
			return st.Name + " cannot stand on " + d.Grounds[string(c.Ground)].Name
		}
	}
	if c.State != NoLayer {
		s, ok := d.States[string(c.State)]
		if !ok {
			return "unknown state " + string(c.State)
		}
		if s.On != "" && !strings.Contains(s.On, string(c.Structure)) {
			return s.Name + " cannot lie over " + string(c.Structure)
		}
	}
	return ""
}

// validateTerrain checks the layer tables of the bundle.
func (d *Data) validateTerrain() string {
	if len(d.Grounds) == 0 || len(d.Terrain) == 0 {
		return "missing terrain definitions"
	}
	glyphs := map[string]string{}
	single := func(k string) bool { return len([]rune(k)) == 1 && k != string(NoLayer) }
	for k, g := range d.Grounds {
		if !single(k) || len(g.Glyph) != 1 || glyphs[g.Glyph] != "" {
			return "ground " + k
		}
		glyphs[g.Glyph] = k
	}
	for k, st := range d.Structures {
		if !single(k) || len(st.Glyph) != 1 || glyphs[st.Glyph] != "" || st.Grounds == "" {
			return "structure " + k
		}
		glyphs[st.Glyph] = k
		for _, g := range st.Grounds {
			if _, ok := d.Grounds[string(g)]; !ok {
				return "structure " + k + " stands on unknown ground " + string(g)
			}
		}
	}
	for k, s := range d.States {
		if !single(k) || s.Name == "" {
			return "state " + k
		}
		if _, ok := d.States[s.Then]; s.Then != "" && !ok {
			return "state " + k + " turns into unknown state " + s.Then
		}
		for _, st := range s.On {
			if _, ok := d.Structures[string(st)]; !ok {
				return "state " + k + " lies over unknown structure " + string(st)
			}
		}
	}
	if c, ok := ParseCell(d.Rules.EdgeTile); !ok || d.ValidCell(c) != "" || c.State != NoLayer {
		return "edge tile " + d.Rules.EdgeTile
	}
	for group, set := range d.Terrain {
		for k := range d.Grounds {
			if r, ok := set.Ground[k]; !ok || r.Factor == nil || r.Cost == nil {
				return "terrain " + group + " lacks ground " + k
			}
		}
		for k := range set.Ground {
			if _, ok := d.Grounds[k]; !ok {
				return "terrain " + group + ": unknown ground " + k
			}
		}
		for k := range set.Structure {
			if _, ok := d.Structures[k]; !ok {
				return "terrain " + group + ": unknown structure " + k
			}
		}
		for k := range set.State {
			if _, ok := d.States[k]; !ok {
				return "terrain " + group + ": unknown state " + k
			}
		}
	}
	return ""
}
