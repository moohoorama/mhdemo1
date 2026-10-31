package gui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func init() { demos["cnmap"] = demoWorldMap }

// demoWorldMapRoute is the walk the world map demo takes when -stage names none: from Zhuo to Ji and Beiping.
const demoWorldMapRoute = "place:계,place:북평"

// demoWorldMap walks the hero over assets/worldmap/han.json as clicks would (-stage "place:Name,warp:Name,cell:U:V,zoom:N,
// grid,..."): each step is taken once the hero has stood still for half a second.
func demoWorldMap(d *Demo) error {
	g := d.g
	if err := g.OpenWorldMap(filepath.Join("assets", "worldmap", "han.json")); err != nil {
		return err
	}
	acts := strings.Split(demoWorldMapRoute, ",")
	if d.stage != "" && !strings.HasPrefix(d.stage, "B") {
		acts = strings.Split(d.stage, ",")
	}
	idle := 0
	d.step = func(f int) {
		w := g.world
		g.worldTick()
		hero := g.scene.actors[w.hero]
		if hero.walk != nil || len(g.scene.paths[w.hero]) > 0 {
			idle = 0
			return
		}
		if idle++; idle < 30 || len(acts) == 0 {
			return
		}
		idle = 0
		act := acts[0]
		acts = acts[1:]
		if err := g.demoWorldAct(act); err != nil {
			d.err = err
		}
	}
	return nil
}

// demoWorldAct clicks where one step of the demo's walk points.
func (g *Game) demoWorldAct(act string) error {
	w := g.world
	kind, arg, _ := strings.Cut(act, ":")
	switch kind {
	case "grid":
		w.show = !w.show
		return nil
	case "zoom":
		z, err := strconv.ParseFloat(arg, 64)
		g.zoom = z
		return err
	case "warp": // stand the hero just south of a place at once, to look at another part of the map
		for _, p := range w.file.Places {
			if p.Name == arg {
				hero := g.scene.actors[w.hero]
				t := w.grid.nearestOpen(smallTile{2*p.At[0] + 1, 2*p.At[1] + 3})
				hero.u, hero.v = t.cell()
				g.worldTick()
				return nil
			}
		}
	case "place":
		for _, p := range w.file.Places {
			if p.Name == arg {
				g.demoWorldClick(g.field.Center(float64(p.At[0]), float64(p.At[1])))
				return nil
			}
		}
	case "cell":
		su, sv, _ := strings.Cut(arg, ":")
		u, err1 := strconv.Atoi(su)
		v, err2 := strconv.Atoi(sv)
		if err1 == nil && err2 == nil {
			g.demoWorldClick(g.field.Center(float64(u), float64(v)))
			return nil
		}
	}
	return fmt.Errorf("world map demo: no %q", act)
}

// demoWorldClick clicks canvas point (x, y) through the screen, as the mouse would.
func (g *Game) demoWorldClick(x, y float64) {
	sx, sy := worldToUI(g.toScreen(x, y))
	cursorPos = func() (int, int) { return int(sx), int(sy) }
	g.worldClick(g.toCanvas(cursorWorld()))
}
