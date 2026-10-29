package gui

import (
	"cmp"
	"fmt"
	"image"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
)

// floorTile is a battle map small tile as drawn (docs/plan-tile-floors.md): its floor composed once per drawing list
// (one picture per water frame, the same one where it holds no water) and drawn lowered by its depth, then the walls
// on its back edges.
type floorTile struct {
	cx, cy float64 // the small tile's centre on the canvas
	depth  float64
	floor  [8]*ebiten.Image
	walls  []Sprite
	rect   image.Rectangle // all it can paint, for culling
}

// depthOf is how far below the ground a battle map small tile with glyph t lies: water (and the water under a bridge),
// ford, land.
func depthOf(t byte) int {
	switch t {
	case '~', 'b':
		return sunkDepth
	case 's':
		return fordDepth
	}
	return 0
}

// levelDepth is how far below the ground a ground surface level (content.GroundLevel) lies.
func levelDepth(level int) int {
	switch level {
	case 0:
		return sunkDepth
	case 1:
		return fordDepth
	}
	return 0
}

// makeFloors lays out a battle map's small tiles far to near: each one's floor (its four ground quarters with only
// the layers at its own depth, and a ford's stones) and the walls on its NW and NE edges where the tile behind lies
// higher, from that tile's depth down to its own, the rows below the floor of the side neighbour in front cut away.
func (f *Field) makeFloors() {
	if !f.Map.SunkWater {
		return
	}
	quarters := map[image.Point]*[4][]int{}
	for _, t := range f.Map.Ground {
		// a fragment sits on the quarter grid at q = 2x + i, r = 2y + j: X = (q - r) * 4, Y = (q + r) * 2
		q, r := (t.X+2*t.Y)/8, (2*t.Y-t.X)/8
		at := image.Pt(floorDiv(q, 2), floorDiv(r, 2))
		if quarters[at] == nil {
			quarters[at] = &[4][]int{}
		}
		quarters[at][q-2*at.X+2*(r-2*at.Y)] = t.Layers
	}
	water := f.A.Animations["water"].Frames
	cache := map[string]*ebiten.Image{}
	type placed struct {
		x, y int
		tile floorTile
	}
	var tiles []placed
	f.eachSmall(func(x, y int, t byte, px, py float64) {
		qs := quarters[image.Pt(x, y)]
		if qs == nil {
			return
		}
		d := depthOf(t)
		bed := ""
		if t == 's' {
			bed = fmt.Sprint("ford_bed_", (x*7+y*13)%8)
		}
		ft := floorTile{cx: px * K, cy: py * K, depth: float64(d), walls: f.wallsOf(x, y, t)}
		for i := range ft.floor {
			ft.floor[i] = f.floorPicture(cache, qs, d, water[i%len(water)], bed)
		}
		ft.rect = image.Rect(int(ft.cx)-16, int(ft.cy)-8, int(ft.cx)+16, int(ft.cy)+8+sunkDepth+2)
		tiles = append(tiles, placed{x, y, ft})
	})
	slices.SortStableFunc(tiles, func(a, b placed) int { return cmp.Or(cmp.Compare(a.x+a.y, b.x+b.y), cmp.Compare(a.x, b.x)) })
	f.floorTiles = make([]floorTile, len(tiles))
	for i, p := range tiles {
		f.floorTiles[i] = p.tile
	}
}

// floorPicture is the 32×16 floor of a small tile at depth with these quarters: each quarter's layers at that depth
// (the water as animation frame), then the ford's stones; composed once per drawing list.
func (f *Field) floorPicture(cache map[string]*ebiten.Image, qs *[4][]int, depth, frame int, bed string) *ebiten.Image {
	var ids [4][]int
	var key strings.Builder
	for n, layers := range qs {
		for _, id := range layers {
			if levelDepth(content.GroundLevel(id)) != depth {
				continue // the edge of a surface at another depth: a wall parts them
			}
			if id == 0 {
				id = frame
			}
			ids[n] = append(ids[n], id)
			fmt.Fprint(&key, id, ",")
		}
		key.WriteByte('|')
	}
	key.WriteString(bed)
	if im, ok := cache[key.String()]; ok {
		return im
	}
	im := ebiten.NewImage(32, 16)
	for n, list := range ids {
		i, j := n%2, n/2
		for _, id := range list {
			s := f.A.Tiles[id]
			op := &ebiten.DrawImageOptions{}
			// quarter (i, j)'s top vertex sits ((i - j) * 8, (i + j) * 4) from the small tile's top vertex
			op.GeoM.Translate(float64(16+(i-j)*8)-s.PivotX, float64((i+j)*4)-s.PivotY)
			im.DrawImage(s.Image, op)
		}
	}
	if bed != "" {
		s := f.A.Props[bed].Sprite
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(16-s.PivotX, 8-s.PivotY)
		im.DrawImage(s.Image, op)
	}
	cache[key.String()] = im
	return im
}

// wallsOf are the walls of small tile (x, y) with glyph t: on its NW and NE edges where the tile behind lies higher.
// Earth with a grass or soil lip under land, a stone abutment under a bridge's end, a low stony drop under a ford;
// each drawn for the depth of the side neighbour in front, whose floor hides the rows below it.
func (f *Field) wallsOf(x, y int, t byte) []Sprite {
	d := depthOf(t)
	var walls []Sprite
	for _, e := range []struct {
		side        string
		back, side2 [2]int // the neighbour behind the edge, and the one beside it in front on the same side
	}{{"nw", [2]int{-1, 0}, [2]int{0, 1}}, {"ne", [2]int{0, -1}, [2]int{1, 0}}} {
		b := f.small(x+e.back[0], y+e.back[1])
		db := depthOf(b)
		if db >= d {
			continue // the tile behind lies as low or lower: its face looks away
		}
		kind := "grass"
		switch {
		case b == 's':
			kind = "weir"
		case t == 'b':
			kind = "stone"
		case strings.IndexByte("dichm", b) >= 0:
			kind = "soil"
		}
		d2 := depthOf(f.small(x+e.side2[0], y+e.side2[1]))
		if p, ok := f.A.Props[fmt.Sprintf("wall_%s_%s_%d%d_%d", kind, e.side, db, d, d2)]; ok {
			walls = append(walls, p.Sprite)
		}
	}
	return walls
}

// drawFloors draws the battle map's floors far to near, each lowered by its depth, with the water at the frame of the
// clock, then its ground props.
func (f *Field) drawFloors(dst *ebiten.Image, ms float64) {
	frame := int(ms/float64(f.A.Animations["water"].MS)) % len(floorTile{}.floor)
	for i := range f.floorTiles {
		t := &f.floorTiles[i]
		if !f.DrawBounds.Empty() && !t.rect.Overlaps(f.DrawBounds) {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(t.cx-16, t.cy-8+t.depth)
		f.renderOp(dst, op)
		dst.DrawImage(t.floor[frame], op)
	}
	f.eachProp(true, func(_ string, s Sprite, x, y float64) { f.blit(dst, s, x, y, nil) })
}

// drawWalls draws the battle map's walls far to near, after all the floors.
func (f *Field) drawWalls(dst *ebiten.Image) {
	for i := range f.floorTiles {
		t := &f.floorTiles[i]
		if len(t.walls) == 0 || !f.DrawBounds.Empty() && !t.rect.Overlaps(f.DrawBounds) {
			continue
		}
		for _, s := range t.walls {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(t.cx-s.PivotX, t.cy-s.PivotY)
			f.renderOp(dst, op)
			dst.DrawImage(s.Image, op)
		}
	}
}
