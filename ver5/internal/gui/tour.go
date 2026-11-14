package gui

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/scenemap"
	"srpg/internal/scenes"
)

// tourWalk is the walk round the scene maps of format 2 (assets/towns/tour2.json, cmd/_town2): the hero walks
// small tile by small tile where the player clicks, and clicking an exit (a door's building, a way out) walks there
// and opens the next map. Nobody else stands in it: it is for looking at how the maps are drawn.
type tourWalk struct {
	t     *tour
	area  string
	hero  string
	grid  walkGrid
	aim   *tourExit // the exit the hero is walking to
	going *tourExit // the exit the hero is going through, while the screen fades out
	fade  float64
	mark  smallTile // where the last click on the ground sent the hero
	markT float64
	show  bool // R or a right click: show which small tiles can be walked
}

// tour is a tour file: areas on scene maps of format 2 joined by exits.
type tour struct {
	Name  string
	Hero  string // the walker: a character's id or shown name
	Start string
	Areas map[string]*tourArea
}

type tourArea struct {
	Map   string
	Name  string
	Zoom  float64   // the camera's zoom on arrival, whole numbers; 2 when 0
	At    smallTile // where the hero stands on arriving from nowhere (the start, an area without a way back)
	Exits []tourExit
}

// tourExit leads to area To: the hero walks to small tile At (or the open one nearest), then stands at the exit
// of To that leads back. Object is the small tile of the anchor of the building it belongs to, whose picture can be
// clicked too.
type tourExit struct {
	Name   string
	At     smallTile
	Object *smallTile
	To     string
}

// OpenTour starts the tour of a tour file at its start area.
func (g *Game) OpenTour(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	t := &tour{}
	if err := json.Unmarshal(raw, t); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for id, a := range t.Areas {
		art := g.mapArt(a.Map)
		if art == nil || art.SceneMap == nil {
			return fmt.Errorf("%s: area %s: %s is no scene map of format 2", path, id, a.Map)
		}
		for _, e := range a.Exits {
			if t.Areas[e.To] == nil {
				return fmt.Errorf("%s: area %s: exit %q leads to unknown area %q", path, id, e.Name, e.To)
			}
		}
	}
	if t.Areas[t.Start] == nil {
		return fmt.Errorf("%s: start area %q is missing", path, t.Start)
	}
	hero := t.Hero
	if c, ok := g.S.Data.Characters[hero]; ok {
		hero = c.Name
	}
	g.tour = &tourWalk{t: t, hero: hero}
	g.enterTourArea(t.Start, "")
	return nil
}

// enterTourArea opens area id, the hero standing at its exit back to area from (at its At when there is none).
func (g *Game) enterTourArea(id, from string) {
	w := g.tour
	a := w.t.Areas[id]
	art := g.mapArt(a.Map)
	w.area, w.aim, w.going, w.fade, w.markT = id, nil, nil, 0, math.Inf(1)
	w.grid = walkGrid{walk: scenemap.Walkable(art.SceneMap, g.assets.SceneRules)}
	at, face := a.At, "SE"
	for _, e := range a.Exits {
		if e.To == from {
			at, face = e.At, "NW"
		}
	}
	at = w.grid.nearestOpen(at)
	u, v := at.cell()
	g.openScene(&scenes.Scene{ID: "tour:" + id, Map: a.Map, Cast: map[string]scenes.Actor{w.hero: {At: []float64{u, v}, Face: face}}})
	g.zoom = 2
	if a.Zoom >= 1 {
		g.zoom = math.Round(a.Zoom)
	}
	g.scene.camU, g.scene.camV = u, v
	g.camX, g.camY = g.sceneCam()
}

func (g *Game) updateTour() error {
	g.advance(1.0 / 60)
	if g.tourTick() {
		g.tourInput()
	}
	return nil
}

// tourTick follows the hero with the camera and goes through the exit the hero arrived at; false while the
// screen fades to the next area.
func (g *Game) tourTick() bool {
	w := g.tour
	w.markT += 1.0 / 60
	s := g.scene
	hero := s.actors[w.hero]
	s.camU, s.camV = hero.u, hero.v
	if w.going != nil {
		if w.fade = math.Min(1, w.fade+1.0/60/walkFade); w.fade >= 1 {
			g.enterTourArea(w.going.To, w.area)
		}
		return false
	}
	if w.aim != nil && hero.walk == nil && len(s.paths[w.hero]) == 0 {
		w.going, w.aim = w.aim, nil
	}
	return true
}

func (g *Game) tourInput() {
	w := g.tour
	if keyJustPressed(ebiten.KeyR) || mouseJustPressed(ebiten.MouseButtonRight) {
		w.show = !w.show
	}
	if _, wheel := ebiten.Wheel(); wheel != 0 {
		g.zoom = math.Max(1, math.Min(4, g.zoom+math.Copysign(1, wheel)))
	}
	if mouseJustPressed(ebiten.MouseButtonLeft) {
		g.tourClick(g.toCanvas(cursorWorld()))
	}
}

// tourClick sends the hero to the exit under canvas point (x, y), or else to the small tile there.
func (g *Game) tourClick(x, y float64) {
	w := g.tour
	w.aim = nil
	if e := g.tourExitAt(x, y); e != nil {
		if g.walkTour(w.grid.nearestOpen(e.At)) {
			w.aim = e
		}
		return
	}
	if to, ok := w.grid.nearestReachable(g.actorTile(w.hero), smallTileAt(g.cellAt(x, y))); ok && g.walkTour(to) {
		w.mark, w.markT = to, 0
	}
}

// tourExitAt is the exit whose spot (or a small tile next to it) or whose building's picture lies under (x, y).
func (g *Game) tourExitAt(x, y float64) *tourExit {
	w := g.tour
	at := smallTileAt(g.cellAt(x, y))
	a := w.t.Areas[w.area]
	for i := range a.Exits {
		e := &a.Exits[i]
		if abs(at[0]-e.At[0]) <= 1 && abs(at[1]-e.At[1]) <= 1 {
			return e
		}
		if e.Object != nil && g.objectHit(*e.Object, x, y) {
			return e
		}
	}
	return nil
}

// walkTour sends the hero along the shortest way to small tile to; false when it cannot be reached.
func (g *Game) walkTour(to smallTile) bool {
	route := g.tour.grid.path(g.actorTile(g.tour.hero), to)
	if route == nil {
		return false
	}
	g.walkActor(g.tour.hero, route, heroStep)
	return true
}

// drawTour draws the area, the exits' name plates, the small tile under the cursor and the walkable tiles (R).
func (g *Game) drawTour(dst *ebiten.Image) {
	w := g.tour
	a := w.t.Areas[w.area]
	g.clickRing(w.mark, w.markT)
	g.drawScene(dst, nil)
	if w.show {
		var exits []smallTile
		for _, e := range a.Exits {
			exits = append(exits, e.At)
		}
		g.drawWalkGrid(dst, &w.grid, exits)
	}
	x, y := g.toCanvas(cursorWorld())
	hover := g.tourExitAt(x, y)
	if w.going == nil && hover == nil {
		if t := smallTileAt(g.cellAt(x, y)); w.grid.open(t) {
			g.smallDiamond(dst, t, cursorFill, cursorRim)
		}
	}
	for i := range a.Exits {
		e := &a.Exits[i]
		px, py := g.field.Center(e.At.cell())
		py -= 24
		if e.Object != nil {
			if ox, oy, ok := g.objectTop(*e.Object); ok {
				px, py = ox, oy
			}
		}
		sx, sy := worldToUI(g.toScreen(px, py))
		g.namePlate(dst, e.Name, sx, sy-6, plateDim, e == hover)
	}
	g.label(dst, w.t.Name+" · "+a.Name+" ("+a.Map+")", 24, 18, 18, gold)
	g.label(dst, "클릭: 이동, 이름판·건물·출구: 다음 장면   휠: 확대   R·오른쪽 클릭: 걸을 수 있는 칸", 24, 46, 13, muted)
	g.blackOut(dst, math.Max(w.fade, g.scene.fade))
}
