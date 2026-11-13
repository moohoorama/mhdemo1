package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/scenemap"
)

// A scene map is shown as a diorama board (docs/presentation-scene.md, 판 단면): the ground ends at the board's edges,
// under its two front edges (+u, +v) lies the cross-section of the ground, and a city wall running into a front edge
// shows its cut there. tools/scene_maps/diorama.py bakes the pictures (assets/graphics/maps/board/<ID>.png and .json);
// the game only draws them: the section right over the ground, each cut with the wall piece of its tile. What would
// stand across the board's edge is left out whole when the map loads (scenemap's plan for format 2, trimToBoard for
// format 1).

// mapBoard is a map's baked board edge.
type mapBoard struct {
	Rect    [4]int // the board in small tiles: u0, v0, u1, v1
	Source  string // sha1 of the map file it was baked from
	Section []boardPiece
	Cuts    []boardCut
}

// boardPiece is a picture with its top left on the board canvas: ground point (cu, cv) at (16(cu - cv), 8(cu + cv))
// from the map's origin.
type boardPiece struct {
	Image *ebiten.Image
	At    image.Point
}

// boardCut is a wall's cut face at a front edge, drawn with what stands on its small tile.
type boardCut struct {
	boardPiece
	Tile [2]int
	Kind string // "wall" (a wall line's last tile) or "rampart" (a city wall top's edge tile)
}

// loadBoard reads maps/board/<id>.json and its picture under the graphics directory; nil when the map has none.
func loadBoard(dir, id string) (*mapBoard, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "maps", "board", id+".json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc struct {
		Board   [4]int   `json:"board"`
		Source  string   `json:"source"`
		Image   string   `json:"image"`
		Section [][6]int `json:"section"`
		Cuts    []struct {
			Rect [4]int `json:"rect"`
			At   [2]int `json:"at"`
			Tile [2]int `json:"tile"`
			Kind string `json:"kind"`
		} `json:"cuts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("board %s: %w", id, err)
	}
	pix, err := loadRGBA(filepath.Join(dir, "maps", "board", doc.Image))
	if err != nil {
		return nil, err
	}
	sheet := ebiten.NewImageFromImage(pix)
	sub := func(r [4]int) *ebiten.Image {
		return sheet.SubImage(image.Rect(r[0], r[1], r[0]+r[2], r[1]+r[3])).(*ebiten.Image)
	}
	b := &mapBoard{Rect: doc.Board, Source: doc.Source}
	for _, s := range doc.Section {
		b.Section = append(b.Section, boardPiece{sub([4]int{s[0], s[1], s[2], s[3]}), image.Pt(s[4], s[5])})
	}
	for _, c := range doc.Cuts {
		b.Cuts = append(b.Cuts, boardCut{boardPiece{sub(c.Rect), image.Pt(c.At[0], c.At[1])}, c.Tile, c.Kind})
	}
	return b, nil
}

// trimToBoard leaves out of a format 1 scene map what would stand across the edge of its board, the visual
// rectangle: props whose footprint (scene.json footprints, else the small tile round the anchor) leaves it
// (scenemap.OnBoard), and the ground decoration within a quarter tile of a front edge or off the board. A fill (a run of one prop) stays.
func (m *MapArt) trimToBoard(c *scenemap.Catalog) {
	if !m.Native || !m.Scene || m.SceneMap != nil || m.Visual[2] == 0 {
		return
	}
	board := [4]int{m.Visual[0], m.Visual[1], m.Visual[0] + m.Visual[2], m.Visual[1] + m.Visual[3]}
	tileOn := func(cu, cv float64) bool {
		u, v := int(math.Floor(cu)), int(math.Floor(cv))
		return u >= board[0] && v >= board[1] && u < board[2] && v < board[3]
	}
	in := func(cu, cv float64) bool {
		return scenemap.OnBoard([4]float64{cu - .25, cv - .25, cu + .25, cv + .25}, board, tileOn(cu, cv))
	}
	props := m.Props[:0]
	for _, p := range m.Props {
		cu, cv := 2*p.U+1, 2*p.V+1
		f := c.Footprint(p.Name)
		if p.Fill || scenemap.OnBoard([4]float64{cu + f[0], cv + f[1], cu + f[0] + f[2], cv + f[1] + f[3]}, board, tileOn(cu, cv)) {
			props = append(props, p)
		}
	}
	m.Props = props
	decor := m.Decorations[:0]
	for _, d := range m.Decorations {
		a, b := float64(d.X)/8, float64(d.Y)/4 // cu - cv, cu + cv
		cu, cv := (a+b)/2, (b-a)/2
		if in(cu, cv) {
			decor = append(decor, d)
		}
	}
	m.Decorations = decor
}

// drawBoardSection lays the board's ground section, over the ground and under everything standing.
func (f *Field) drawBoardSection(dst *ebiten.Image) {
	b := f.Map.Board
	if b == nil {
		return
	}
	ox, oy := f.OriginX*K, f.OriginY*K
	for _, p := range b.Section {
		r := p.Image.Bounds().Sub(p.Image.Bounds().Min).Add(p.At).Add(image.Pt(int(ox), int(oy)))
		if !f.DrawBounds.Empty() && !r.Overlaps(f.DrawBounds) {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
		f.renderOp(dst, op)
		dst.DrawImage(p.Image, op)
	}
}

// boardItems are the board's wall cuts as standing items: on a format 2 map sorted as the wall piece (or the rise's
// face) of their tile, just after it; on a format 1 map just after the wall column of their tile.
func (f *Field) boardItems() []drawItem {
	b := f.Map.Board
	if b == nil {
		return nil
	}
	ox, oy := f.OriginX*K, f.OriginY*K
	var items []drawItem
	for _, c := range b.Cuts {
		r := c.Image.Bounds().Sub(c.Image.Bounds().Min).Add(c.At).Add(image.Pt(int(ox), int(oy)))
		if !f.DrawBounds.Empty() && !r.Overlaps(f.DrawBounds) {
			continue
		}
		img := c.Image
		u, v := c.Tile[0], c.Tile[1]
		cx, cy := ox+float64(16*(u-v)), oy+float64(8*(u+v+1))
		it := drawItem{y: cy + .5, rect: r, hides: true, draw: func() {
			op := &ebiten.DrawImageOptions{}
			f.shadeScenery(f.Canvas, op, cx, cy)
			op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
			f.renderOp(f.Canvas, op)
			f.Canvas.DrawImage(img, op)
		}}
		if f.Map.SceneMap != nil {
			rank := scenemap.RankWall
			if c.Kind == "rampart" {
				rank = scenemap.RankFace
			}
			k := scenemap.Key{Box: [4]float64{float64(u), float64(v), float64(u + 1), float64(v + 1)}, Rank: rank, X: float64(16*(u-v)) + .5, Rect: r}
			it.key = &k
		}
		items = append(items, it)
	}
	return items
}
