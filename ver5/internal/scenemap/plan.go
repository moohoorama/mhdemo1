package scenemap

import (
	"fmt"
	"image"
	"math"
	"sort"
	"strconv"
)

// Draw is one picture of a plan, in canvas pixels from the map's origin.
type Draw struct {
	Kind string `json:"k"`
	// Name is a sprite's name; for a tile, the animation it is a frame of ("water", "grass_0"), if any.
	Name string `json:"n,omitempty"`
	// Sheet is set for a sprite off the props and extra sheets: "scenery" (ground decoration), "structures" (paving).
	Sheet string `json:"s,omitempty"`
	Tile  int    `json:"t"`
	// X, Y: a picture's top left; a diamond's top vertex.
	X     int      `json:"x"`
	Y     int      `json:"y"`
	Color string   `json:"c,omitempty"`
	Lines [][4]int `json:"l,omitempty"` // x0, y0, x1, y1 of each segment, stroked 1 px wide
	// Object is 1 + the index in the map's objects of the object this draws, 0 for none (for the editor to pick it).
	Object int `json:"o,omitempty"`
}

// Kinds of Draw.
const (
	DrawTile   = "tile"   // a ground tileset piece
	DrawSprite = "sprite" // a prop or scenery picture
	DrawFill   = "fill"   // a game cell's diamond filled with Color, 64 x 32 (DiamondRow)
	DrawLines  = "lines"
)

// Piece is one item of the standing layer: a raised block, a wall piece, an object. Its draws go out together.
type Piece struct {
	Key
	Name   string     `json:"name"`
	At     [2]int     `json:"at"`     // the map's small tile it belongs to
	Anchor [2]float64 `json:"anchor"` // canvas point it stands on (raised with its ground)
	Draws  []Draw     `json:"draws"`
}

// Plan is a map's drawing list: the flat layer in order, then the standing layer in order. People and effects are
// the caller's: they go into the standing layer by Order with the pieces' keys, and over everything.
type Plan struct {
	Bounds [4]int  `json:"bounds"`
	Tall   bool    `json:"tall"`
	Flat   []Draw  `json:"flat"`
	Solid  []Piece `json:"solid"`
}

type Options struct {
	Turn  int  // quarter turns of the view (the viewer's 0°, 90°, 180°, 270°)
	Decor bool // scatter the ground decoration
}

// DiamondRow is the span [x0, x1) of row k (0..31) of a diamond whose top vertex is at x.
func DiamondRow(x, k int) (int, int) {
	hw := 2 * (k + 1)
	if k >= 16 {
		hw = 2 * (32 - k)
	}
	return x - hw, x + hw
}

// Build lays out a map's plan.
func Build(m *Map, c *Catalog, o Options) (*Plan, error) {
	w := newView(m, c, o.Turn)
	p := &Plan{Bounds: m.Bounds(), Tall: m.Tall()}
	p.Flat = append(w.ground(), w.floors()...)
	blocks, feet, err := w.blocks()
	if err != nil {
		return nil, err
	}
	walls, err := w.walls()
	if err != nil {
		return nil, err
	}
	objects, flat, err := w.objects()
	if err != nil {
		return nil, err
	}
	pieces := append(append(blocks, walls...), objects...)
	if err := w.cornerShadows(pieces); err != nil {
		return nil, err
	}
	p.Flat = append(p.Flat, flat...)
	if o.Decor {
		p.Flat = append(p.Flat, w.decoration()...)
	}
	p.Flat = append(p.Flat, feet...)
	keys := make([]Key, len(pieces))
	for i := range pieces {
		pieces[i].Rect = w.rect(pieces[i].Draws)
		keys[i] = pieces[i].Key
	}
	for _, i := range Order(keys) {
		p.Solid = append(p.Solid, pieces[i])
	}
	return p, nil
}

// view is the map seen at a quarter turn: rotated small tiles (u, v) on the grid [u0, u1) x [v0, v1). The rules
// work on the rotated tiles, so every turn is drawn by the same rules; what is picked at random by place (a cliff's
// variant, decoration) is picked by the map's own tile, so it turns with the map.
type view struct {
	m              *Map
	c              *Catalog
	turn           int
	u0, v0, u1, v1 int
}

func newView(m *Map, c *Catalog, turn int) *view {
	w := &view{m: m, c: c, turn: (turn%4 + 4) % 4}
	au, av := w.rot(m.Origin[0], m.Origin[1])
	bu, bv := w.rot(m.Origin[0]+m.W()-1, m.Origin[1]+m.H()-1)
	w.u0, w.v0, w.u1, w.v1 = min(au, bu), min(av, bv), max(au, bu)+1, max(av, bv)+1
	return w
}

// rot is the view's tile of the map's small tile (u, v); orig undoes it. A turn rotates game cells about game cell
// (0, 0), so small tile u becomes 1 - u.
func (w *view) rot(u, v int) (int, int) {
	switch w.turn {
	case 1:
		return v, 1 - u
	case 2:
		return 1 - u, 1 - v
	case 3:
		return 1 - v, u
	}
	return u, v
}

func (w *view) orig(u, v int) (int, int) {
	switch w.turn {
	case 1:
		return 1 - v, u
	case 2:
		return 1 - u, 1 - v
	case 3:
		return v, 1 - u
	}
	return u, v
}

// point turns a ground point (cu, cv) of the map into the view.
func (w *view) point(cu, cv float64) (float64, float64) {
	switch w.turn {
	case 1:
		return cv, 2 - cu
	case 2:
		return 2 - cu, 2 - cv
	case 3:
		return 2 - cv, cu
	}
	return cu, cv
}

func (w *view) inside(u, v int) bool { return w.m.Inside(w.orig(u, v)) }
func (w *view) glyph(u, v int) byte  { return w.m.Glyph(w.orig(u, v)) }
func (w *view) step(u, v int) int    { return w.m.Step(w.orig(u, v)) }

// center is the canvas point of a small tile's centre.
func center(u, v int) (int, int) { return 16 * (u - v), 8 * (u + v + 1) }

// round is the drawing position's rounding (the browser's Math.round), the same in the viewer and the game.
func round(x float64) int { return int(math.Floor(x + .5)) }

func floorHalf(x int) int { return int(math.Floor(float64(x) / 2)) }

// isFloor reports whether a glyph's floor takes the place of the ground (a game cell's floor, not paving).
func (w *view) isFloor(g byte) bool {
	f, ok := w.c.Rules.Floors[string(g)]
	return ok && !f.Tile
}

func (w *view) height(step int) (lift int, faces, foot, rim string, err error) {
	if step == 0 {
		return 0, "", "", "", nil
	}
	h, ok := w.c.Rules.Heights[strconv.Itoa(step)]
	if !ok {
		return 0, "", "", "", fmt.Errorf("%s: no rule for height step %d", w.m.ID, step)
	}
	return h.Lift, h.Faces, h.Foot, h.Rim, nil
}

// layers are the tileset pieces of quarter (i, j) of small tile (u, v), bottom first: a grass or soil piece cut to
// the shape of the ground round each of its corners (build.py's mask and ground_layers). A corner counts the small
// tiles round it that stand at least as high as (u, v): what lies lower is out of sight below a cliff's edge.
func (w *view) layers(u, v, i, j int) []int {
	h := w.step(u, v)
	vertex := func(sx, sy, kind int) bool {
		for y := floorHalf(sy - 1); y <= floorHalf(sy); y++ {
			for x := floorHalf(sx - 1); x <= floorHalf(sx); x++ {
				if w.inside(x, y) && w.step(x, y) >= h && w.c.surface(w.glyph(x, y)) >= kind {
					return true
				}
			}
		}
		return false
	}
	mask := func(kind int) int {
		n := 0
		for bit, d := range [][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
			if vertex(2*u+i+d[0], 2*v+j+d[1], kind) {
				n |= 1 << bit
			}
		}
		return n
	}
	dry, soil := mask(1), mask(2)
	layers := []int{8}
	if soil != 15 {
		layers = []int{23}
		if dry != 15 {
			layers = []int{0}
			if dry != 0 {
				layers = append(layers, 23+dry)
			}
		}
		if soil != 0 {
			layers = append(layers, 8+soil)
		}
	}
	return layers
}

// quarters draws the ground pieces of small tile (u, v), lift pixels up.
func (w *view) quarters(u, v, lift int, out []Draw) []Draw {
	for j := range 2 {
		for i := range 2 {
			x, y := 16*(u-v)+8*(i-j), 8*(u+v)+4*(i+j)-lift
			for _, id := range w.layers(u, v, i, j) {
				d := Draw{Kind: DrawTile, Tile: id}
				if id == 0 {
					d.Name, d.Tile = "water", w.c.Animations["water"][0]
				}
				t := w.c.Tiles[d.Tile]
				d.X, d.Y = x-t.Pivot[0], y-t.Pivot[1]
				out = append(out, d)
			}
		}
	}
	return out
}

// ground is the flat ground: every small tile at height 0 that is not a floor, in screen order.
func (w *view) ground() []Draw {
	var out []Draw
	for v := w.v0; v < w.v1; v++ {
		for u := w.u0; u < w.u1; u++ {
			if w.step(u, v) == 0 && !w.isFloor(w.glyph(u, v)) {
				out = w.quarters(u, v, 0, out)
			}
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Y < out[b].Y })
	return out
}

// floors lays a floor picture on each game cell whose small tiles are mostly that floor, over its diamond filled
// with the floor's edge colour, so neighbouring floors meet without a seam; and paving on each of its small tiles,
// over the ground.
func (w *view) floors() []Draw {
	var out []Draw
	for v := w.v0; v < w.v1; v++ {
		for u := w.u0; u < w.u1; u++ {
			f, ok := w.c.Rules.Floors[string(w.glyph(u, v))]
			if !ok || !f.Tile || w.step(u, v) > 0 {
				continue
			}
			s := w.c.Structures[f.Sprite]
			x, y := center(u, v)
			out = append(out, Draw{Kind: DrawSprite, Sheet: f.Sheet, Name: f.Sprite, X: x - s.Pivot[0], Y: y - s.Pivot[1]})
		}
	}
	for gv := floorHalf(w.v0); gv <= floorHalf(w.v1-1); gv++ {
		for gu := floorHalf(w.u0); gu <= floorHalf(w.u1-1); gu++ {
			count := map[byte]int{}
			var best byte
			for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				u, v := 2*gu+d[0], 2*gv+d[1]
				if g := w.glyph(u, v); w.inside(u, v) && w.isFloor(g) {
					count[g]++
					if count[g] > count[best] || count[g] == count[best] && g < best {
						best = g
					}
				}
			}
			if best == 0 {
				continue
			}
			f := w.c.Rules.Floors[string(best)]
			s := w.c.Sprites[f.Sprite]
			x, y := 32*(gu-gv), 16*(gu+gv+1)
			out = append(out, Draw{Kind: DrawFill, X: x, Y: y - 16, Color: f.Edge},
				Draw{Kind: DrawSprite, Name: f.Sprite, X: x - s.Pivot[0], Y: y - s.Pivot[1]})
		}
	}
	return out
}

// blocks are the raised small tiles: each its top (its ground, lifted), the outline of the top's back edges where
// they drop, and the picture of its faces that look at lower ground; feet are the shadow bands along those faces'
// feet on the ground in front of them.
func (w *view) blocks() (blocks []Piece, feet []Draw, err error) {
	for v := w.v0; v < w.v1; v++ {
		for u := w.u0; u < w.u1; u++ {
			s := w.step(u, v)
			if s == 0 {
				continue
			}
			lift, faces, foot, rim, err := w.height(s)
			if err != nil {
				return nil, nil, err
			}
			open := 0
			for bit, d := range [][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}} {
				if w.step(u+d[0], v+d[1]) < s {
					open |= 1 << bit
				}
			}
			x, y := center(u, v)
			ou, ov := w.orig(u, v)
			p := Piece{Key: Key{Box: [4]float64{float64(u), float64(v), float64(u + 1), float64(v + 1)}, Rank: RankFace,
				X: float64(x)}, Name: "top", At: [2]int{ou, ov}, Anchor: [2]float64{float64(x), float64(y - lift)}}
			p.Draws = w.quarters(u, v, lift, nil)
			if open&3 != 0 {
				d := Draw{Kind: DrawLines, Color: rim}
				top := y - 8 - lift
				if open&1 != 0 {
					d.Lines = append(d.Lines, [4]int{x, top, x - 16, top + 8})
				}
				if open&2 != 0 {
					d.Lines = append(d.Lines, [4]int{x, top, x + 16, top + 8})
				}
				p.Draws = append(p.Draws, d)
			}
			if open&12 != 0 {
				name := fmt.Sprintf("%s_%02d_%d", faces, open&12, ((ou*7+ov*3)%3+3)%3)
				sp, ok := w.c.Sprites[name]
				if !ok {
					return nil, nil, fmt.Errorf("%s: no picture %s", w.m.ID, name)
				}
				p.Name = name
				p.Draws = append(p.Draws, Draw{Kind: DrawSprite, Name: name, X: x - sp.Pivot[0], Y: y - sp.Pivot[1]})
			}
			blocks = append(blocks, p)
			for _, f := range []struct {
				bit, du, dv int
				side        string
			}{{8, 0, 1, "sw"}, {4, 1, 0, "se"}} {
				if open&f.bit != 0 && w.inside(u+f.du, v+f.dv) && foot != "" {
					name := foot + "_" + f.side
					sp := w.c.Sprites[name]
					feet = append(feet, Draw{Kind: DrawSprite, Name: name, X: x - sp.Pivot[0], Y: y - sp.Pivot[1]})
				}
			}
		}
	}
	sort.SliceStable(feet, func(a, b int) bool { return feet[a].Y < feet[b].Y })
	return blocks, feet, nil
}

// walls lays each wall line one block per small tile, and sets the gates standing on it into it: a gate is a run of
// slices along the line, one per small tile (index.json gate_offset), each in place of the wall block where it
// stands. A piece on a tile higher than the line's foot is inside the rise and is not drawn. A block takes a parapet
// on each side, in the view, that no other wall or gate tile touches: the sides behind (-u, -v) before those in
// front, so lines meet at corners and run on through junctions without their parapets crossing. Every tile first
// lays its shadow on the ground out of its open front sides (WallFamily).
func (w *view) walls() ([]Piece, error) {
	type tile struct {
		at   [2]int // original small tile
		name string // the gate slice, or "" for a block
		fam  WallFamily
	}
	laidAt := map[[2]int]tile{} // by view tile
	seated := map[int]bool{}
	for _, line := range w.m.Walls {
		family, ok := w.c.Rules.Walls[line.Family]
		if !ok {
			return nil, fmt.Errorf("%s: no wall family %s", w.m.ID, line.Family)
		}
		axis, fixed, t0, t1 := "u", line.From[1], line.From[0], line.To[0]
		if line.From[0] == line.To[0] {
			axis, fixed, t0, t1 = "v", line.From[0], line.From[1], line.To[1]
		}
		t0, t1 = min(t0, t1), max(t0, t1)
		at := func(t int) (int, int) {
			if axis == "u" {
				return t, fixed
			}
			return fixed, t
		}
		type laid struct {
			gate   string // "" for a wall block; else the gate's slices
			slice  int
			mirror int // the slice in its place when the turn runs the gate the other way along the line
		}
		pieces := map[int]laid{}
		for t := t0; t <= t1; t++ {
			pieces[t] = laid{}
		}
		for gi, ob := range w.m.Objects {
			if !w.c.isGate(ob.Name) {
				continue
			}
			on, gt := ob.U == fixed, ob.V
			if axis == "u" {
				on, gt = ob.V == fixed, ob.U
			}
			if !on || gt < t0 || gt > t1 {
				continue
			}
			seated[gi] = true
			gate := w.c.Rules.Gates[ob.Name]
			slices := w.c.gateSlices(gate, axis)
			if len(slices) == 0 {
				return nil, fmt.Errorf("%s: no slices of gate %s along %s", w.m.ID, gate, axis)
			}
			for k, s := range slices {
				pieces[gt+round(2**s.GateOffset)] = laid{gate, k, len(slices) - 1 - k}
			}
		}
		fu, fv := at(t0)
		foot := w.m.Step(fu, fv)
		turned := axis
		if w.turn%2 == 1 {
			turned = map[string]string{"u": "v", "v": "u"}[axis]
		}
		// a gate's slices run the other way along the line when the turn reverses its direction on screen
		reverse := axis == "v" && w.turn >= 2 || axis == "u" && (w.turn == 1 || w.turn == 2)
		for t, l := range pieces {
			ou, ov := at(t)
			if w.m.Step(ou, ov) > foot {
				continue
			}
			u, v := w.rot(ou, ov)
			name := ""
			if l.gate != "" {
				k := l.slice
				if reverse {
					k = l.mirror
				}
				name = fmt.Sprintf("%s_%s_%d", l.gate, turned, k)
			}
			if prev, ok := laidAt[[2]int{u, v}]; ok && (prev.name != "" || name == "") {
				continue // where lines cross, one block; a gate slice before a block
			}
			laidAt[[2]int{u, v}] = tile{[2]int{ou, ov}, name, family}
		}
	}
	for i, ob := range w.m.Objects {
		if w.c.isGate(ob.Name) && !seated[i] {
			return nil, fmt.Errorf("%s: gate %s at (%d, %d) stands on no wall line", w.m.ID, ob.Name, ob.U, ob.V)
		}
	}
	var out []Piece
	for vt, l := range laidAt {
		u, v := vt[0], vt[1]
		x, y := center(u, v)
		_, closedU := laidAt[[2]int{u + 1, v}]
		_, closedV := laidAt[[2]int{u, v + 1}]
		var names []string
		for _, sh := range []struct {
			open   bool
			suffix string
		}{{!closedU, "_pu"}, {!closedV, "_pv"}, {!closedU && !closedV, "_pc"}} {
			if sh.open && l.fam.Shade != "" {
				names = append(names, l.fam.Shade+sh.suffix)
			}
		}
		if l.name != "" {
			names = append(names, l.name)
		} else {
			names = append(names, l.fam.Block)
			for _, side := range []struct {
				du, dv int
				suffix string
			}{{-1, 0, "_mu"}, {0, -1, "_mv"}, {1, 0, "_pu"}, {0, 1, "_pv"}} {
				if _, ok := laidAt[[2]int{u + side.du, v + side.dv}]; !ok {
					names = append(names, l.fam.Parapet+side.suffix)
				}
			}
		}
		name := l.name
		if name == "" {
			name = l.fam.Block
		}
		p := Piece{Key: Key{Box: [4]float64{float64(u), float64(v), float64(u + 1), float64(v + 1)}, Rank: RankWall,
			X: float64(x)}, Name: name, At: l.at, Anchor: [2]float64{float64(x), float64(y)}}
		for _, name := range names {
			sp, ok := w.c.Sprites[name]
			if !ok {
				return nil, fmt.Errorf("%s: no picture %s", w.m.ID, name)
			}
			p.Draws = append(p.Draws, Draw{Kind: DrawSprite, Name: name, X: x - sp.Pivot[0], Y: y - sp.Pivot[1]})
		}
		out = append(out, p)
	}
	sort.Slice(out, func(a, b int) bool {
		return out[a].At[0] < out[b].At[0] || out[a].At[0] == out[b].At[0] && out[a].At[1] < out[b].At[1]
	})
	return out, nil
}

// gateSlices are a gate's slices along axis, by index.
func (c *Catalog) gateSlices(gate, axis string) []Sprite {
	var out []Sprite
	for k := 0; ; k++ {
		s, ok := c.Sprites[fmt.Sprintf("%s_%s_%d", gate, axis, k)]
		if !ok || s.GateOffset == nil {
			return out
		}
		out = append(out, s)
	}
}

// objects stand at their anchors, on the height of the tile under the anchor; objects that lie flat (layer
// "ground": paving, mats) go to the flat layer in screen order.
func (w *view) objects() (pieces []Piece, flat []Draw, err error) {
	type flatDraw struct {
		d Draw
		y float64
	}
	var flats []flatDraw
	for i, ob := range w.m.Objects {
		if w.c.isGate(ob.Name) {
			continue // set into its wall by walls
		}
		s, ok := w.c.Sprites[ob.Name]
		if !ok {
			return nil, nil, fmt.Errorf("%s: no picture %s", w.m.ID, ob.Name)
		}
		cu := float64(ob.U) + .5 + float64(ob.DX)/32 + float64(ob.DY)/16
		cv := float64(ob.V) + .5 - float64(ob.DX)/32 + float64(ob.DY)/16
		ru, rv := w.point(cu, cv)
		x, y := 16*(ru-rv), 8*(ru+rv)
		lift, _, _, _, err := w.height(w.step(int(math.Floor(ru)), int(math.Floor(rv))))
		if err != nil {
			return nil, nil, err
		}
		d := Draw{Kind: DrawSprite, Name: ob.Name, X: round(x - float64(s.Pivot[0])), Y: round(y - float64(s.Pivot[1]) - float64(lift)), Object: i + 1}
		if s.Ground {
			flats = append(flats, flatDraw{d, y})
			continue
		}
		f := w.c.footprint(ob.Name)
		au, av := w.point(cu+f[0], cv+f[1])
		bu, bv := w.point(cu+f[0]+f[2], cv+f[1]+f[3])
		pieces = append(pieces, Piece{Key: Key{Box: [4]float64{min(au, bu), min(av, bv), max(au, bu), max(av, bv)},
			Lift: lift, Rank: RankObject, X: x}, Name: ob.Name, At: [2]int{ob.U, ob.V}, Anchor: [2]float64{x, y - float64(lift)},
			Draws: []Draw{d}})
	}
	sort.SliceStable(flats, func(a, b int) bool { return flats[a].y < flats[b].y })
	for _, f := range flats {
		flat = append(flat, f.d)
	}
	return pieces, flat, nil
}

// cornerShadows lays the triangular shadow a standing face throws on the face to its right where the two meet in a
// back corner: a face looking to +u (a wall along v, a block's SE face) whose far end (toward -v) meets a face
// looking to +v (a wall along u, a block's SW face) that runs on toward +u. The shadow is as high as the throwing
// face and goes on in slices, one per tile of the lit face, each drawn with that face's piece.
func (w *view) cornerShadows(pieces []Piece) error {
	type face struct {
		piece  int
		height int
	}
	se, sw := map[[2]int]face{}, map[[2]int]face{}
	wall := map[[2]int]bool{}
	for _, p := range pieces {
		if p.Rank == RankWall {
			wall[[2]int{int(p.Box[0]), int(p.Box[1])}] = true
		}
	}
	for i, p := range pieces {
		u, v := int(p.Box[0]), int(p.Box[1])
		switch p.Rank {
		case RankFace:
			open := 0
			s := w.step(u, v)
			if w.step(u+1, v) < s {
				open |= 4
			}
			if w.step(u, v+1) < s {
				open |= 8
			}
			lift, _, _, _, _ := w.height(s)
			if open&4 != 0 {
				se[[2]int{u, v}] = face{i, lift}
			}
			if open&8 != 0 {
				sw[[2]int{u, v}] = face{i, lift}
			}
		case RankWall:
			sp := w.c.Sprites[p.Name]
			if !wall[[2]int{u + 1, v}] {
				se[[2]int{u, v}] = face{i, sp.Pivot[1]}
			}
			if !wall[[2]int{u, v + 1}] {
				sw[[2]int{u, v}] = face{i, sp.Pivot[1]}
			}
		}
	}
	cs := w.c.Rules.CornerShadow
	for at, f := range se {
		if _, ok := se[[2]int{at[0], at[1] - 1}]; ok {
			continue
		}
		h := max(8, min(160, f.height/8*8))
		for j := 0; ; j++ {
			name := fmt.Sprintf("%s_%d_%d", cs.Sprite, h, j)
			sp, ok := w.c.Sprites[name]
			if !ok {
				break
			}
			lit, ok := sw[[2]int{at[0] + 1 + j, at[1] - 1}]
			if !ok {
				break
			}
			x, y := center(at[0]+1+j, at[1]-1)
			p := &pieces[lit.piece]
			p.Draws = append(p.Draws, Draw{Kind: DrawSprite, Name: name, X: x + cs.Offset[0] - sp.Pivot[0], Y: y + cs.Offset[1] - sp.Pivot[1]})
		}
	}
	return nil
}

// decoration is the ground decoration (Scenery.Decorate) over the whole grid, scattered by the map's own small
// tiles and turned with the map. Raised tiles take none and keep it off their feet, like cliffs; what lies on a
// cliff block's top belongs to battle maps' blocks and is left out.
func (w *view) decoration() []Draw {
	sc := w.c.Scenery
	if sc == nil {
		return nil
	}
	glyph := func(u, v int) byte {
		if !w.m.Inside(u, v) {
			return '~'
		}
		if w.m.Step(u, v) > 0 {
			return 'h'
		}
		return w.m.Glyph(u, v)
	}
	stamps, _ := sc.Decorate(w.m.ID, glyph, w.m.Origin[0], w.m.Origin[1], w.m.Origin[0]+w.m.W(), w.m.Origin[1]+w.m.H(),
		func(u, v int) (float64, float64) {
			x, y := center(u, v)
			return float64(x), float64(y)
		}, 0)
	var out []Draw
	for _, st := range stamps {
		if st.Top {
			continue
		}
		ru, rv := w.point((st.X/16+st.Y/8)/2, (st.Y/8-st.X/16)/2)
		s := sc.Sprites[st.Name]
		out = append(out, Draw{Kind: DrawSprite, Sheet: "scenery", Name: st.Name,
			X: round(16*(ru-rv)) - s.Pivot[0], Y: round(8*(ru+rv)) - s.Pivot[1]})
	}
	return out
}

// rect is the canvas area a piece's draws cover.
func (w *view) rect(draws []Draw) image.Rectangle {
	var r image.Rectangle
	for _, d := range draws {
		var b image.Rectangle
		switch d.Kind {
		case DrawTile:
			t := w.c.Tiles[d.Tile]
			b = image.Rect(d.X, d.Y, d.X+t.Rect[2], d.Y+t.Rect[3])
		case DrawSprite:
			s := w.c.Sprites[d.Name]
			switch d.Sheet {
			case "scenery":
				s = w.c.Scenery.Sprites[d.Name]
			case "structures":
				s = w.c.Structures[d.Name]
			}
			b = image.Rect(d.X, d.Y, d.X+s.Rect[2], d.Y+s.Rect[3])
		case DrawFill:
			b = image.Rect(d.X-32, d.Y, d.X+32, d.Y+32)
		case DrawLines:
			for _, l := range d.Lines {
				b = b.Union(image.Rect(min(l[0], l[2]), min(l[1], l[3]), max(l[0], l[2])+1, max(l[1], l[3])+1))
			}
		}
		r = r.Union(b)
	}
	return r
}
