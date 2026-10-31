package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// worldWalk is the walk over the world map (assets/worldmap/han.json, cmd/_cnmap): the deformed Han empire drawn
// as one battle map — its ground, rivers, forests, castles and villages the battle map's own, its mountains, hills
// and roads the pieces of assets/graphics/map/terrain.png. The hero walks where the player clicks, round mountains and
// over water only by bridges; standing at a place shows what it is.
type worldWalk struct {
	file  *worldFile
	hero  string
	grid  walkGrid
	roads map[[2]int]bool
	at    *worldPlace // the place the hero stands at, nil between places
	mark  smallTile   // where the last click on the ground sent the hero
	markT float64
	show  bool // R or a right click: show which small tiles can be walked
}

// worldFile is a world map made by tools/worldmap/gen_hanmap.py: one character per cell (Legend), the road
// cells and the places.
type worldFile struct {
	Name   string
	Hero   string // the walker: a character's id or shown name
	Size   int
	Rows   []string
	Roads  [][2]int
	Places []worldPlace
	Start  string
}

type worldPlace struct {
	Name, Province, Kind string // Kind: capital, city, gate, village, camp (an army's), field (of battle), tribe
	At                   [2]int
	Note                 string // what the story does there, by chapter; empty off the story's way
}

const worldMapID = "cnmap"

// worldGround is each cell kind's battle map cell: ground, structure, state.
var worldGround = map[byte]string{'.': "g--", ',': "d--", '\'': "l--", '~': "w--", 'T': "gt-", 'h': "g--", 'M': "g--"}

// worldSite is each place kind's battle map cell.
var worldSite = map[string]string{"capital": "go-", "city": "go-", "camp": "gk-", "tribe": "gk-", "village": "gv-", "gate": "ip-",
	"field": "g--"}

var worldKindName = map[byte]string{'.': "초원", ',': "황무지", '\'': "평지", '~': "물", 'T': "숲", 'h': "언덕", 'M': "산"}

var worldPlaceKind = map[string]string{"capital": "도읍", "city": "성", "camp": "진영", "tribe": "이민족 거점", "gate": "관문",
	"village": "마을", "field": "싸움터"}

// OpenWorldMap starts the walk over the world map at path.
func (g *Game) OpenWorldMap(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	w := &worldFile{}
	if err := json.Unmarshal(raw, w); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if len(w.Rows) != w.Size {
		return fmt.Errorf("%s: %d rows for a map of %d", path, len(w.Rows), w.Size)
	}
	walk := &worldWalk{file: w, hero: w.Hero, roads: map[[2]int]bool{}}
	if c, ok := g.S.Data.Characters[walk.hero]; ok {
		walk.hero = c.Name
	}
	for _, r := range w.Roads {
		walk.roads[r] = true
	}
	g.assets.Maps[worldMapID] = artFromDocument(walk.document(), g.S.Data)
	start := w.Places[0].At
	for _, p := range w.Places {
		if p.Name == w.Start {
			start = p.At
		}
	}
	n := w.Size
	walk.grid = walkGrid{walk: scenemap.NewWalk([2]int{0, 0}, 2*n, 2*n, func(u, v int) bool { return walk.open(u/2, v/2) })}
	g.world = walk
	u, v := float64(start[0]), float64(start[1])+1
	g.openScene(&scenes.Scene{ID: worldMapID, Map: worldMapID, Cast: map[string]scenes.Actor{walk.hero: {At: []float64{u, v}, Face: "SE"}}})
	g.scene.camU, g.scene.camV = u, v
	g.camX, g.camY = g.sceneCam()
	return nil
}

func (w *worldWalk) kind(u, v int) byte {
	n := w.file.Size
	return w.file.Rows[max(0, min(n-1, v))][max(0, min(n-1, u))]
}

func (w *worldWalk) place(u, v int) *worldPlace {
	for i := range w.file.Places {
		if p := &w.file.Places[i]; p.At == [2]int{u, v} {
			return p
		}
	}
	return nil
}

// open reports whether the hero can stand on cell (u, v): anywhere but mountains and water, and on water where a
// road crosses it by a bridge.
func (w *worldWalk) open(u, v int) bool {
	n := w.file.Size
	if u < 0 || v < 0 || u >= n || v >= n {
		return false
	}
	switch w.kind(u, v) {
	case 'M':
		return w.place(u, v) != nil
	case '~':
		return w.roads[[2]int{u, v}] || w.place(u, v) != nil
	}
	return true
}

// document is the world as a battle map document: the cells (the ring outside the play area repeats the edge),
// the mountains and hills standing on their cells, the roads laid on theirs.
func (w *worldWalk) document() *content.MapDocument {
	n := w.file.Size
	d := &content.MapDocument{Version: content.MapVersion, Play: content.MapRect{W: n, H: n},
		Visual: content.MapRect{X: -4, Y: -4, W: n + 8, H: n + 8}, Cells: map[string]string{}, Default: "g--"}
	d.Stage.ID, d.Stage.Name = worldMapID, w.file.Name
	for v := -4; v < n+4; v++ {
		for u := -4; u < n+4; u++ {
			cu, cv := max(0, min(n-1, u)), max(0, min(n-1, v))
			cell := worldGround[w.kind(cu, cv)]
			if p := w.place(cu, cv); p != nil && (cu == u && cv == v) {
				cell = worldSite[p.Kind]
			} else if w.roads[[2]int{cu, cv}] && cell[0] == 'w' && cu == u && cv == v {
				cell = "wb-"
			} else if cell == "gt-" && w.roads[[2]int{cu, cv}] {
				cell = "g--" // the road clears its way through the forest
			}
			d.Cells[content.Key(u, v)] = cell
		}
	}
	for v := 0; v < n; v++ {
		for u := 0; u < n; u++ {
			if name := w.relief(u, v); name != "" {
				d.Props = append(d.Props, []any{name, float64(u), float64(v)})
			}
			if !w.roads[[2]int{u, v}] || w.kind(u, v) == '~' || w.place(u, v) != nil {
				continue
			}
			mask := 0
			for bit, s := range [4][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}} {
				if w.roads[[2]int{u + s[0], v + s[1]}] {
					mask |= 1 << bit
				}
			}
			d.Props = append(d.Props, []any{fmt.Sprintf("wm_road_%02d", mask), float64(u), float64(v)})
		}
	}
	return d
}

// relief names the mountain or hill picture of cell (u, v), or "": stone and snow in the high north-west, green
// mountains in the wet south, bare rock between; a mountain ringed by mountains may be a big one.
func (w *worldWalk) relief(u, v int) string {
	k := w.kind(u, v)
	if k != 'M' && k != 'h' || w.place(u, v) != nil {
		return ""
	}
	h := (u*7 + v*13 + u*v*3) % 97
	if k == 'h' {
		return fmt.Sprintf("wm_hill_%d", h%3)
	}
	ringed := true
	for _, s := range [4][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}} {
		ringed = ringed && w.kind(u+s[0], v+s[1]) == 'M'
	}
	switch {
	case u+v < 40 && u < 30:
		if ringed && h%3 == 0 {
			return fmt.Sprintf("wm_peak_%d", h%2)
		}
		if h%4 == 0 {
			return fmt.Sprintf("wm_mtn_snow_%d", h%2)
		}
		return fmt.Sprintf("wm_mtn_stone_%d", h%3)
	case v > 58:
		return fmt.Sprintf("wm_mtn_green_%d", h%3)
	}
	if ringed && h%3 == 0 {
		return fmt.Sprintf("wm_mtn_big_%d", h%2)
	}
	return fmt.Sprintf("wm_mtn_rock_%d", h%3)
}

func (g *Game) updateWorld() error {
	g.advance(1.0 / 60)
	g.worldTick()
	w := g.world
	if keyJustPressed(ebiten.KeyR) || mouseJustPressed(ebiten.MouseButtonRight) {
		w.show = !w.show
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.zoom = math.Max(1, math.Min(4, g.zoom+math.Copysign(1, wheel)))
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) {
		g.worldClick(g.toCanvas(cursorWorld()))
	}
	return nil
}

// worldTick follows the hero with the camera and notes the place the hero has stopped at.
func (g *Game) worldTick() {
	w := g.world
	w.markT += 1.0 / 60
	s := g.scene
	hero := s.actors[w.hero]
	s.camU, s.camV = hero.u, hero.v
	g.camX, g.camY = g.sceneCam()
	w.at = nil
	if hero.walk == nil && len(s.paths[w.hero]) == 0 {
		at := g.actorTile(w.hero)
		w.at = w.place(at[0]/2, at[1]/2)
	}
}

// worldClick sends the hero to the place under canvas point (x, y), or else to the small tile there.
func (g *Game) worldClick(x, y float64) {
	w := g.world
	to := smallTileAt(g.cellAt(x, y))
	if p := g.worldPlaceAt(x, y); p != nil {
		to = smallTile{2*p.At[0] + 1, 2*p.At[1] + 1}
	}
	to, ok := w.grid.nearestReachable(g.actorTile(w.hero), to)
	if !ok {
		return
	}
	if route := w.grid.path(g.actorTile(w.hero), to); route != nil {
		g.walkActor(w.hero, route, heroStep*2)
		w.mark, w.markT = to, 0
	}
}

// worldPlaceAt is the place whose cell lies under canvas point (x, y), else the place whose name plate does.
func (g *Game) worldPlaceAt(x, y float64) *worldPlace {
	u, v := g.cellAt(x, y)
	for i := range g.world.file.Places {
		p := &g.world.file.Places[i]
		if math.Abs(u-float64(p.At[0])) <= .6 && math.Abs(v-float64(p.At[1])) <= .6 {
			return p
		}
	}
	for i := range g.world.file.Places {
		p := &g.world.file.Places[i]
		px, py := g.field.Center(float64(p.At[0]), float64(p.At[1]))
		if math.Abs(x-px) < 30 && y < py && y > py-plateLift(p.Kind)-24 {
			return p
		}
	}
	return nil
}

// plateLift is how far over its cell's centre a place's name plate stands, clear of its picture.
func plateLift(kind string) float64 {
	switch kind {
	case "capital", "city":
		return 56
	case "gate":
		return 44
	case "field":
		return 16
	}
	return 36
}

func (g *Game) drawWorld(dst *ebiten.Image) {
	w := g.world
	g.clickRing(w.mark, w.markT)
	g.drawWorldField(dst)
	if w.show {
		g.drawWalkGrid(dst, &w.grid, nil)
	}
	x, y := g.toCanvas(cursorWorld())
	hover := g.worldPlaceAt(x, y)
	for _, k := range []string{"field", "village", "tribe", "gate", "camp", "city", "capital"} { // the greater on top
		for i := range w.file.Places {
			p := &w.file.Places[i]
			if p.Kind != k {
				continue
			}
			px, py := g.field.Center(float64(p.At[0]), float64(p.At[1]))
			sx, sy := worldToUI(g.toScreen(px, py-plateLift(k)))
			if sx < -60 || sy < -20 || sx > float64(uiW)+60 || sy > float64(uiH)+40 {
				continue
			}
			style := plateDim // off the story's way
			if p.Note != "" {
				style = plateMain
			}
			g.namePlate(dst, p.Name, sx, sy, style, p == hover || p == w.at)
		}
	}
	g.label(dst, w.file.Name, 24, 18, 20, gold)
	g.label(dst, g.worldWhere(), 24, 48, 15, ink)
	g.label(dst, "클릭: 이동 · 이름판: 그곳으로 · 휠: 확대 · R·오른쪽 클릭: 갈 수 있는 칸", 24, float64(uiH)-34, 13, muted)
	if p := w.at; p != nil {
		r := image.Rect(24, uiH-176, 344, uiH-56)
		g.titled(dst, r, p.Name)
		g.label(dst, p.Province+" · "+worldPlaceKind[p.Kind], float64(r.Min.X)+22, float64(r.Min.Y)+26, 16, ink)
		if p.Note != "" {
			g.label(dst, p.Note, float64(r.Min.X)+22, float64(r.Min.Y)+50, 15, gold)
		}
		g.label(dst, g.worldNeighbours(p), float64(r.Min.X)+22, float64(r.Min.Y)+80, 14, muted)
	}
}

// drawWorldField draws the part of the map the camera sees, as the battle screen does (battleMap): a story scene
// draws its whole map every frame, too much for a map this size.
func (g *Game) drawWorldField(dst *ebiten.Image) {
	vp := g.battleViewport()
	hw, hh := float64(vp.Dx())/2/g.zoom, float64(vp.Dy())/2/g.zoom
	g.field.prepareRender(g.zoom, g.camX-hw, g.camY-hh, vp.Dx(), vp.Dy())
	g.field.DrawBounds = image.Rect(int(g.camX-hw)-64, int(g.camY-hh)-96, int(g.camX+hw)+64, int(g.camY+hh)+96)
	actors := make([]*unitVis, 0, len(g.scene.actors))
	for _, v := range g.scene.actors {
		actors = append(actors, v)
	}
	g.field.Draw(g.clockMS, actors, nil)
	inWorld(func() { drawScaled(dst, g.field.Canvas, &ebiten.DrawImageOptions{Filter: ebiten.FilterPixelated}) })
}

// worldWhere names the hero's ground and the province it lies in (the nearest city's).
func (g *Game) worldWhere() string {
	w := g.world
	at := g.actorTile(w.hero)
	u, v := at[0]/2, at[1]/2
	ground := worldKindName[w.kind(u, v)]
	if w.roads[[2]int{u, v}] {
		ground = "길"
		if w.kind(u, v) == '~' {
			ground = "다리"
		}
	}
	near, best := (*worldPlace)(nil), math.Inf(1)
	for i := range w.file.Places {
		p := &w.file.Places[i]
		if d := math.Hypot(float64(p.At[0]-u), float64(p.At[1]-v)); (p.Kind == "city" || p.Kind == "capital") && d < best {
			near, best = p, d
		}
	}
	if near == nil {
		return ground
	}
	return fmt.Sprintf("%s · %s 근처 · %s", near.Province, near.Name, ground)
}

// worldNeighbours lists the cities nearest a place, the next stops along the roads.
func (g *Game) worldNeighbours(p *worldPlace) string {
	type far struct {
		name string
		d    float64
	}
	var list []far
	for _, q := range g.world.file.Places {
		if q.Name != p.Name && (q.Kind == "city" || q.Kind == "capital") {
			list = append(list, far{q.Name, math.Hypot(float64(q.At[0]-p.At[0]), float64(q.At[1]-p.At[1]))})
		}
	}
	s := "이웃:"
	for k := 0; k < 3 && len(list) > 0; k++ {
		best := 0
		for i := range list {
			if list[i].d < list[best].d {
				best = i
			}
		}
		s += " " + list[best].name
		list = append(list[:best], list[best+1:]...)
	}
	return s
}
