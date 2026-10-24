package gui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func init() { demos["tour"] = demoTour }

// demoTourRoute is the walk the tour demo takes when -stage names none: round the town, into the hall and out, to
// the checkpoint, the castle town and the tent.
const demoTourRoute = "walk:12:13,exit:관사,walk:9:5,exit:거리로,walk:16:17,exit:관문으로,walk:17:9,exit:성 안 민가로,walk:10:12,exit:군막,walk:4:4,exit:밖으로"

// demoTour walks the hero round assets/towns/tour2.json as clicks would (-stage "walk:U:V,exit:Name,grid,..."):
// each step is clicked once the hero has stood still for half a second.
func demoTour(d *Demo) error {
	g := d.g
	if err := g.OpenTour(filepath.Join("assets", "towns", "tour2.json")); err != nil {
		return err
	}
	acts := strings.Split(demoTourRoute, ",")
	if strings.Contains(d.stage, ":") || d.stage == "grid" {
		acts = strings.Split(d.stage, ",")
	}
	idle := 0
	d.step = func(f int) {
		w := g.tour
		if !g.tourTick() {
			idle = 0
			return
		}
		hero := g.scene.actors[w.hero]
		if w.aim != nil || hero.walk != nil || len(g.scene.paths[w.hero]) > 0 {
			idle = 0
			return
		}
		if idle++; idle < 30 || len(acts) == 0 {
			return
		}
		idle = 0
		act := acts[0]
		acts = acts[1:]
		if err := g.demoTourAct(act); err != nil {
			d.err = err
		}
	}
	return nil
}

// demoTourAct clicks where one step of the demo's walk points.
func (g *Game) demoTourAct(act string) error {
	w := g.tour
	kind, arg, _ := strings.Cut(act, ":")
	switch kind {
	case "grid":
		w.show = !w.show
		return nil
	case "exit":
		for _, e := range w.t.Areas[w.area].Exits {
			if e.Name == arg {
				x, y := g.field.Center(e.At.cell())
				g.demoTourClick(x, y)
				return nil
			}
		}
	case "walk":
		su, sv, _ := strings.Cut(arg, ":")
		u, err1 := strconv.Atoi(su)
		v, err2 := strconv.Atoi(sv)
		if err1 == nil && err2 == nil {
			x, y := g.field.Center(smallTile{u, v}.cell())
			g.demoTourClick(x, y)
			return nil
		}
	}
	return fmt.Errorf("tour demo: no %q in area %s", act, w.area)
}

// demoTourClick clicks canvas point (x, y) through the screen, as the mouse would.
func (g *Game) demoTourClick(x, y float64) {
	sx, sy := worldToUI(g.toScreen(x, y))
	cursorPos = func() (int, int) { return int(sx), int(sy) }
	g.tourClick(g.toCanvas(cursorWorld()))
}
