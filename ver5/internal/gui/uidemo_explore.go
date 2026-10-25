package gui

import (
	"fmt"
	"strconv"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["explore"] = demoExplore }

// demoExploreRoute is the walk the explore demo takes when -stage names none: chapter 1 from the street to the
// sortie.
const demoExploreRoute = "exit:주점,talk:관우,exit:거리로,exit:성문 밖,talk:장비,exit:의병 주둔지,talk:관우"

// demoExplore plays a new game's story walk as clicks would (-stage "start:Node,talk:Name,exit:Name,sign:Name,
// walk:U:V,grid,shop:Name,buy:Item"; start, first, begins the story at that node): each step is clicked once the
// hero has stood still for half a second. Lines page by themselves, the sortie question is answered yes, the sortie
// screen goes on to the battle, its briefing closes and a shop closes after a second.
func demoExplore(d *Demo) error {
	g := d.g
	acts := strings.Split(demoExploreRoute, ",")
	if strings.Contains(d.stage, ":") || d.stage == "grid" {
		acts = strings.Split(d.stage, ",")
	}
	if node, ok := strings.CutPrefix(acts[0], "start:"); ok {
		g.S.Data.Campaign.Start, acts = node, acts[1:]
	}
	if r := g.request(session.Request{Op: "new", Seed: 1, Discard: true}); !r.OK {
		return fmt.Errorf("explore demo: %s", r.Error)
	}
	cursorPos = func() (int, int) { return -1000, -1000 } // off the map until the first click
	idle, wait := 0, 0
	d.step = func(f int) {
		wait++
		if g.narrationLeaving && g.uiSeconds-g.narrationExitAt >= .43 { // as Update ends a narration that faded out
			g.narrationLeaving = false
			g.command(core.Command{Kind: "next"})
		}
		o := g.S.Engine.Observe()
		switch {
		case g.overlay == "confirm" && wait > 40:
			answer := g.answer
			g.overlay, wait = "", 0
			answer()
			return
		case g.overlay == "shop" && wait > 60:
			g.overlay, wait = "", 0
			return
		case g.overlay != "":
			return
		case len(g.modals) > 0:
			if wait > 90 {
				g.closeModal() // the battle's briefing
				wait = 0
			}
			return
		case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
			if wait > 60 {
				g.command(core.Command{Kind: "start"}) // the sortie screen: go, and the battle's opening plays
				wait = 0
			}
			return
		case o.Phase == "scenario" && isTalk(o.Dialogue.Kind):
			if wait > 50 {
				g.advanceDialogue()
				wait = 0
			}
			return
		case !g.exploring():
			return
		}
		g.updateExplore() // the demo host only advances the clock
		w := g.explore
		if w.talk != nil {
			if wait > 50 {
				g.nextExploreLine()
				wait = 0
			}
			return
		}
		hero := g.scene.actors[w.hero]
		if w.aim != nil || w.going != nil || hero.walk != nil || len(g.scene.paths[w.hero]) > 0 {
			idle = 0
			return
		}
		if idle++; idle < 30 || len(acts) == 0 {
			return
		}
		idle, wait = 0, 0
		act := acts[0]
		acts = acts[1:]
		if err := g.demoExploreAct(act); err != nil {
			d.err = err
		}
		cursorPos = func() (int, int) { return -1000, -1000 } // off again, so the plates show the guide's colours
	}
	return nil
}

// demoExploreAct clicks where one step of the demo's walk points.
func (g *Game) demoExploreAct(act string) error {
	w := g.explore
	a := g.S.Engine.ExploreFile().Areas[w.area]
	kind, arg, _ := strings.Cut(act, ":")
	click := func(u, v float64) {
		x, y := g.field.Center(u, v)
		sx, sy := worldToUI(g.toScreen(x, y))
		cursorPos = func() (int, int) { return int(sx), int(sy) }
		g.exploreClick(g.toCanvas(cursorWorld()))
	}
	switch kind {
	case "grid":
		w.show = !w.show
		return nil
	case "talk", "shop":
		for key, v := range g.scene.actors {
			if _, ok := w.people[key]; ok && content.PersonName(key) == arg && !v.gone {
				x, y := g.field.Center(v.u, v.v)
				sx, sy := worldToUI(g.toScreen(x, y-v.top/2))
				cursorPos = func() (int, int) { return int(sx), int(sy) }
				g.exploreClick(g.toCanvas(cursorWorld()))
				return nil
			}
		}
	case "exit":
		for i := range a.Exits {
			if e := &a.Exits[i]; e.Name == arg || g.exitName(a, e) == arg {
				click(smallTile(e.At).cell())
				return nil
			}
		}
	case "sign":
		for _, s := range a.Signs {
			if s.Name == arg {
				click(smallTile(s.At).cell())
				return nil
			}
		}
	case "walk":
		su, sv, _ := strings.Cut(arg, ":")
		u, err1 := strconv.Atoi(su)
		v, err2 := strconv.Atoi(sv)
		if err1 == nil && err2 == nil {
			click(smallTile{u, v}.cell())
			return nil
		}
	case "buy":
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "buy", Target: g.shopFrom, Item: arg, Count: 1}})
		return nil
	}
	return fmt.Errorf("explore demo: no %q in area %s", act, w.area)
}
