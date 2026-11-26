package gui

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/scenes"
)

// continuityEdge is the walk handing over to an event's scene on the same map (adoptScene): the stage stays and the
// scene's cast is laid over it.
type continuityEdge struct {
	desc  string
	def   *scenes.Scene
	at    int
	area  *content.ExploreArea
	when  []string // conditions under which the event can start
	hero  string
	ident string
}

func isTalkKind(k string) bool { return k == "say" || k == "narrate" || k == "choice" }

// firstShown follows the story from node id through nodes that show nothing to the first nodes that show a line,
// answering back-with-Leave by the nodes after the explore node.
func firstShown(d *content.Data, start, explore string) []string {
	var out []string
	seen := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		n := d.Node(id)
		switch {
		case n.Kind == "back" && n.Leave:
			walk(d.After(explore))
		case isTalkKind(n.Kind):
			out = append(out, id)
		case n.Kind == "back", n.Kind == "battle", n.Kind == "explore", n.Kind == "end":
		default:
			for _, s := range d.Successors(n) {
				walk(s)
			}
		}
	}
	walk(start)
	return out
}

func contradict(a, b []string) bool {
	for _, x := range a {
		for _, y := range b {
			if x == "!"+y || y == "!"+x {
				return true
			}
		}
	}
	return false
}

// adoptedEdges lists every place where a walk's person or exit starts an event whose first line sits in a scene on
// the walk's own map.
func adoptedEdges(t *testing.T) []continuityEdge {
	t.Helper()
	d, err := content.LoadScenario("../../assets/data", "hero")
	require.NoError(t, err)
	book, err := scenes.Load("../../assets/scenes")
	require.NoError(t, err)
	var edges []continuityEdge
	for _, c := range d.Chapters {
		for _, id := range c.Nodes {
			n := d.Node(id)
			x := d.Explores[n.Explore]
			if n.Kind != "explore" || x == nil {
				continue
			}
			hero := d.Characters[x.Hero].Name
			add := func(aid string, a *content.ExploreArea, trigger, event string, when []string) {
				for _, first := range firstShown(d, event, id) {
					def, at := book.For(first)
					if def == nil || def.Cut || def.Map != a.Map {
						continue
					}
					edges = append(edges, continuityEdge{desc: fmt.Sprintf("%s/%s %s -> %s", n.Explore, aid, trigger, first), def: def, at: at, area: a, when: when, hero: hero})
				}
			}
			for aid, a := range x.Areas {
				for key, p := range a.People {
					for _, ev := range p.Event {
						add(aid, a, "person:"+key, ev.Node, slices.Concat(ev.If, p.If))
					}
				}
				for _, e := range a.Exits {
					if e.Node != "" {
						add(aid, a, "exit:"+e.Name, e.Node, e.If)
					}
				}
			}
		}
	}
	require.NotEmpty(t, edges)
	return edges
}

// Whoever stands on the walk when an event starts on its stage stays on stage: a scene may walk them to their places
// or keep them (Stay), but never take them off the stage (Off) only to bring them back, which looks like a person
// vanishing and stepping out of the air beside where they stood. Somebody only the scene has comes in by Enter
// (a walk-in), not by standing there at once. The one exception is a reveal: a person who is replaced in place by
// another who looks the same (the hatted stranger of Chapter 5 who turns out to be Cao Cao) is no vanishing.
func TestEventsOnTheWalkDoNotVanishOrPopUp(t *testing.T) {
	var bad []string
	for _, e := range adoptedEdges(t) {
		walkers := map[string]*content.ExplorePerson{}
		for key, p := range e.area.People {
			if !contradict(e.when, p.If) {
				k := content.PersonName(key)
				if strings.Contains(key, "#") {
					k = key
				}
				walkers[k] = p
			}
		}
		placed := settledCast(e.def, e.at)
		reveal := func(gone string, p *content.ExplorePerson) bool { // somebody new stands in the same place looking the same
			u, v := smallTile(p.At).cell()
			for key, a := range placed {
				if _, on := walkers[key]; !on && !a.Off && len(a.At) == 2 && a.Sprite == p.Sprite && a.Sprite != "" && a.Pose == p.Pose &&
					math.Hypot(a.At[0]-u, a.At[1]-v) <= adoptSame {
					return true
				}
			}
			return false
		}
		for key, a := range placed {
			p, on := walkers[key]
			switch {
			case e.def.Cast[key].Stay:
			case key == e.hero && a.Off:
				bad = append(bad, e.desc+": the hero is on the walk but the scene puts him Off")
			case key == e.hero:
			case on && a.Off && !reveal(key, p):
				bad = append(bad, e.desc+": "+key+" is on the walk but the scene puts them Off")
			case !on && !a.Off && !strings.Contains(key, "#") && !revealed(placed, walkers, key, a):
				bad = append(bad, e.desc+": "+key+" is not on the walk but the scene stands them on the stage at once")
			}
		}
	}
	slices.Sort(bad)
	assert.Empty(t, slices.Compact(bad))
}

// revealed reports whether a person who is on the stage at once only takes the place of an Off person of the walk
// who looks the same.
func revealed(placed map[string]scenes.Actor, walkers map[string]*content.ExplorePerson, key string, a scenes.Actor) bool {
	for name, p := range walkers {
		if !placed[name].Off || p.Sprite != a.Sprite || a.Sprite == "" || p.Pose != a.Pose {
			continue
		}
		if u, v := smallTile(p.At).cell(); len(a.At) == 2 && math.Hypot(a.At[0]-u, a.At[1]-v) <= adoptSame {
			return true
		}
	}
	return false
}
