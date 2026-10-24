package gui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func init() { demos["town"] = demoTown }

// demoTownTour is the walk the town demo takes when -stage names none: talk at the notice, into the tavern and
// back, the smithy, the mansion.
const demoTownTour = "sign:방문,person:장비,door:주점,person:관우,door:거리로,door:무기점,person:대장장이,door:거리로,door:저택,person:장세평,door:거리로"

// demoTown walks the hero round assets/towns/zhuo.json as clicks would (-stage
// "person:장비,door:주점,sign:방문,walk:3.5:5,grid,..."): each step is clicked once the hero stands idle for half a
// second, and each line of a talk is paged after 80 frames.
func demoTown(d *Demo) error {
	g := d.g
	if err := g.OpenTown(filepath.Join("assets", "towns", "zhuo.json")); err != nil {
		return err
	}
	g.S.Settings.InstantText = true
	acts := strings.Split(demoTownTour, ",")
	if d.stage != "" && strings.Contains(d.stage, ":") || d.stage == "grid" {
		acts = strings.Split(d.stage, ",")
	}
	idle, shown := 0, 0
	d.step = func(f int) {
		w := g.town
		if !g.townTick() {
			idle = 0
			return
		}
		hero := g.scene.actors[w.hero]
		if w.talk != nil {
			if f-shown >= 80 {
				shown = f
				g.nextTownLine()
			}
			idle = 0
			return
		}
		shown = f
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
		if err := g.demoTownAct(act); err != nil {
			d.err = err
		}
	}
	return nil
}

// demoTownAct clicks where one step of the demo's walk points.
func (g *Game) demoTownAct(act string) error {
	w := g.town
	a := w.t.Areas[w.area]
	kind, arg, _ := strings.Cut(act, ":")
	switch kind {
	case "grid":
		w.grid = !w.grid
		return nil
	case "person":
		if v := g.scene.actors[arg]; v != nil {
			x, y := g.field.Center(v.u, v.v)
			g.demoTownClick(x, y-v.top/2)
			return nil
		}
	case "sign":
		for _, s := range a.Signs {
			if s.Name == arg {
				x, y := g.field.Center(s.At[0], s.At[1])
				g.demoTownClick(x, y)
				return nil
			}
		}
	case "door":
		for _, d := range a.Doors {
			if d.Name == arg {
				x, y := g.field.Center(d.At[0], d.At[1])
				g.demoTownClick(x, y)
				return nil
			}
		}
	case "walk":
		su, sv, _ := strings.Cut(arg, ":")
		u, err1 := strconv.ParseFloat(su, 64)
		v, err2 := strconv.ParseFloat(sv, 64)
		if err1 == nil && err2 == nil {
			x, y := g.field.Center(u, v)
			g.demoTownClick(x, y)
			return nil
		}
	}
	return fmt.Errorf("town demo: no %q in area %s", act, w.area)
}

// demoTownClick clicks canvas point (x, y) through the screen, as the mouse would: the cursor moves there (and
// stays, so its hover name shows) and the click reads it back.
func (g *Game) demoTownClick(x, y float64) {
	sx, sy := worldToUI(g.toScreen(x, y))
	cursorPos = func() (int, int) { return int(sx), int(sy) }
	g.townClick(g.toCanvas(cursorWorld()))
}
