package gui

import (
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/scenes"
)

// continuityFrame is what a played flow showed in one frame: where the story stood, how dark the stage was and where
// every actor stood.
type continuityFrame struct {
	f        int
	mode     string // "walk" or "story"
	node     string
	def      string
	screen   string
	fade     float64
	leave    bool
	exitFade float64 // the walk's own darkening through an exit
	actors   map[string]continuityActor
}

type continuityActor struct {
	u, v float64
	dir  string
	gone bool
}

// playContinuity plays the explore demo's flow headlessly, as the trace test does, and records every frame.
func playContinuity(t *testing.T, stage string, frames int) []continuityFrame {
	t.Helper()
	d, err := NewDemo("../..", DemoOptions{Name: "explore", Stage: stage})
	require.NoError(t, err)
	g := d.g
	var out []continuityFrame
	for f := 0; f < frames; f++ {
		if d.step != nil {
			d.stepped = f
			d.step(f)
		}
		g.advance(1.0 / 60)
		g.turnPage()
		o := g.S.View()
		if o.Phase == "scenario" && isTalk(o.Dialogue.Kind) && !g.exploring() {
			g.enterScene(o)
		}
		if g.scene == nil {
			continue
		}
		s := g.scene
		mode := "story"
		if g.exploring() {
			mode = "walk"
		}
		fr := continuityFrame{f: f, mode: mode, node: o.Node, def: s.def.ID, screen: g.screenKey(), fade: s.fade, leave: s.leaving,
			actors: map[string]continuityActor{}}
		if g.explore != nil {
			fr.exitFade = g.explore.fade
		}
		for k, a := range s.actors {
			fr.actors[k] = continuityActor{a.u, a.v, a.dir, a.gone}
		}
		out = append(out, fr)
	}
	return out
}

// visibleJumps lists the frames where an actor that stayed on stage moved farther in one frame than any walk or
// ride can (0.25 cells) while the stage was lit enough to see it.
func visibleJumps(frames []continuityFrame, from, to string) []string {
	var jumps []string
	on := from == ""
	for i := 1; i < len(frames); i++ {
		a, b := frames[i-1], frames[i]
		if b.node == from {
			on = true
		}
		if to != "" && b.node == to {
			break
		}
		if !on || b.fade > .5 {
			continue
		}
		for k, p := range b.actors {
			q, ok := a.actors[k]
			if !ok || p.gone || q.gone || a.fade > .5 {
				continue
			}
			if math.Hypot(p.u-q.u, p.v-q.v) > .25 {
				jumps = append(jumps, b.node+" "+k)
			}
		}
	}
	slices.Sort(jumps)
	return slices.Compact(jumps)
}

func firstFrame(frames []continuityFrame, node string) int {
	for i, f := range frames {
		if f.node == node {
			return i
		}
	}
	return -1
}

// Chapter 13: the scene at the Xiaopei office goes to the walk and back to the scene after Zhao Yun has been
// talked to. Nothing may blink to black between two views of the same room, and nobody may jump to the scene's
// first places: the hero stays where the player left him, Zhao Yun where the line before put him.
func TestChapter13FarewellKeepsTheRoomContinuous(t *testing.T) {
	frames := playContinuity(t, "start:c13-xiaopei,talk:조운", 460)
	walk, after := firstFrame(frames, "c13-wait-zhaoyun"), firstFrame(frames, "c13-x13")
	require.Greater(t, walk, 0, "the walk starts")
	require.Greater(t, after, walk, "the story comes back after talking to Zhao Yun")

	var dark []int
	for _, f := range frames[walk-90 : after+1] {
		if (f.node == "c13-x12" || f.node == "c13-wait-zhaoyun") && (f.leave && f.mode == "story" || f.fade > .05) {
			dark = append(dark, f.f)
		}
	}
	assert.Empty(t, dark, "frames where the same room darkens between the scene and the walk")
	assert.Empty(t, visibleJumps(frames, "c13-x12", "c13-x15"), "nobody jumps between the scene and the walk")

	last := frames[after-1]
	first := frames[after]
	require.Equal(t, "walk", last.mode)
	for _, who := range []string{"유비", "조운"} {
		assert.InDelta(t, last.actors[who].u, first.actors[who].u, .45, "%s keeps his place", who)
		assert.InDelta(t, last.actors[who].v, first.actors[who].v, .45, "%s keeps his place", who)
	}
	assert.InDelta(t, last.actors["유비"].u, first.actors["유비"].u, .01, "the hero stays exactly where the player left him")
}

// The other confirmed cases: an event that opens on the walk's own stage gathers people by walking, never by
// vanishing and reappearing or by jumping.
func TestConfirmedCasesStayContinuous(t *testing.T) {
	for _, c := range []struct {
		name, stage, event string
		frames             int
	}{
		{"ch14 sword talk", "start:c14-walk,talk:미축", "c14-sw01", 400},
		{"ch25 report", "start:c25-report-walk,exit:관청,talk:제갈량", "c25-yr-00", 700},
		{"ch23 first visit", "start:c23-walk,exit:융중 가는 길,talk:동자", "c23-a01", 900},
		{"ch23 second visit", "start:c23-walk2,exit:융중 가는 길,talk:제갈균", "c23-b01", 900},
	} {
		t.Run(c.name, func(t *testing.T) {
			frames := playContinuity(t, c.stage, c.frames)
			at := firstFrame(frames, c.event)
			require.Greater(t, at, 0, "the event opens")
			assert.Empty(t, visibleJumps(frames[at-1:], c.event, ""), "nobody jumps when the event opens")
			before, after := frames[at-1], frames[at]
			for k, p := range before.actors {
				if p.gone {
					continue
				}
				q, ok := after.actors[k]
				assert.True(t, ok && !q.gone, "%s does not vanish when the event opens", k)
			}
		})
	}
}

// An event that ends where it began, with nobody moved, goes back to the walk without darkening the street.
func TestEventEndInTheSameRoomDoesNotBlink(t *testing.T) {
	frames := playContinuity(t, "start:c13-walk,talk:피난 상인", 1500)
	begin, back := firstFrame(frames, "c13-m1"), -1
	for i := begin; i > 0 && i < len(frames); i++ {
		if frames[i].mode == "walk" {
			back = i
			break
		}
	}
	require.Greater(t, begin, 0, "the event opens")
	require.Greater(t, back, begin, "the walk comes back")
	for _, f := range frames[begin : back+60] {
		assert.False(t, f.leave, "f%d: the street does not darken when the talk ends", f.f)
		assert.Less(t, f.fade, .05, "f%d: the street stays lit", f.f)
	}
	assert.Empty(t, visibleJumps(frames[begin:], "", ""), "nobody jumps when the walk comes back")
}

// The darkening stays where the story changes the room: an event that arranges people differently when it ends
// (the people go back to their posts) still goes through black, and an exit into another place fades the walk out.
func TestChangesOfPlaceAndArrangementStillFade(t *testing.T) {
	t.Run("event that moves people", func(t *testing.T) {
		frames := playContinuity(t, "start:c14-walk,talk:미축", 1200)
		dark := false
		for _, f := range frames {
			dark = dark || f.leave && f.def == "c14-e-sword"
		}
		assert.True(t, dark, "the sword talk's end rearranges the street under a fade")
	})
	t.Run("exit into another map", func(t *testing.T) {
		frames := playContinuity(t, "start:c14-walk,exit:도구점", 700)
		maps, exit := map[string]bool{}, 0.0
		for _, f := range frames {
			maps[f.screen] = true
			exit = math.Max(exit, f.exitFade)
		}
		assert.Greater(t, len(maps), 1, "the screen changes to the shop's map")
		assert.GreaterOrEqual(t, exit, 1.0, "the walk fades out through the exit")
	})
}

// seamlessGame is a game standing at the Chapter 13 street walk, lit, with the walk's own cast as the scene.
func seamlessGame(t *testing.T) (*Game, *scenes.Scene, string) {
	t.Helper()
	d, err := NewDemo("../..", DemoOptions{Name: "explore", Stage: "start:c13-walk"})
	require.NoError(t, err)
	g := d.g
	g.ensureExplore(g.S.View())
	g.scene.fade = 0
	hero := g.explore.hero
	def := g.exploreDef(g.S.Engine, g.S.View(), &exploreWalk{hero: hero, people: map[string]string{}, rest: map[string]float64{}})
	return g, def, hero
}

func TestSeamlessWalkRules(t *testing.T) {
	someone := func(def *scenes.Scene, hero string) string {
		for _, k := range slices.Sorted(func(yield func(string) bool) {
			for k := range def.Cast {
				if !yield(k) {
					return
				}
			}
		}) {
			if k != hero {
				return k
			}
		}
		t.Fatal("nobody on the street")
		return ""
	}
	for _, c := range []struct {
		name string
		edit func(g *Game, def *scenes.Scene, hero string)
		want bool
	}{
		{"same stage", func(*Game, *scenes.Scene, string) {}, true},
		{"the hero is where the player left him", func(_ *Game, def *scenes.Scene, hero string) {
			a := def.Cast[hero]
			a.At = []float64{a.At[0] + 5, a.At[1] + 5}
			def.Cast[hero] = a
		}, true},
		{"a person within a small tile's centring", func(g *Game, def *scenes.Scene, hero string) {
			k := someone(def, hero)
			g.scene.actors[k].u += .3
		}, true},
		{"the stage is still fading", func(g *Game, _ *scenes.Scene, _ string) { g.scene.fade = .5 }, false},
		{"the stage is darkening", func(g *Game, _ *scenes.Scene, _ string) { g.scene.leaving = true }, false},
		{"another map", func(_ *Game, def *scenes.Scene, _ string) { def.Map = "S_xiaopei_office" }, false},
		{"another time of day", func(_ *Game, def *scenes.Scene, _ string) { def.Time = "night" }, false},
		{"a person stands elsewhere", func(g *Game, def *scenes.Scene, hero string) {
			g.scene.actors[someone(def, hero)].u += 1
		}, false},
		{"a person has gone", func(g *Game, def *scenes.Scene, hero string) {
			g.scene.actors[someone(def, hero)].gone = true
		}, false},
		{"somebody new comes to the walk", func(_ *Game, def *scenes.Scene, _ string) {
			def.Cast["새 인물"] = scenes.Actor{At: []float64{3, 3}}
		}, false},
		{"somebody leaves the walk", func(_ *Game, def *scenes.Scene, hero string) {
			delete(def.Cast, someone(def, hero))
		}, false},
		{"the hero is off the stage", func(g *Game, _ *scenes.Scene, hero string) { g.scene.actors[hero].gone = true }, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			g, def, hero := seamlessGame(t)
			require.True(t, g.seamless(def, hero), "the untouched walk is seamless")
			c.edit(g, def, hero)
			assert.Equal(t, c.want, g.seamless(def, hero))
		})
	}
}

// How an event's scene takes over the walk's stage: Stay keeps a person where they are, Off takes one off, a person
// far from their place walks there and turns on arrival instead of jumping, a Cut starts a fresh scene.
func TestAdoptSceneStaysWalksOrCuts(t *testing.T) {
	g, walk, hero := seamlessGame(t)
	var names []string
	for k := range walk.Cast {
		if k != hero {
			names = append(names, k)
		}
	}
	slices.Sort(names)
	require.GreaterOrEqual(t, len(names), 3)
	stayer, goner, walker := names[0], names[1], names[2]
	s := g.scene
	stayAt := [2]float64{s.actors[stayer].u, s.actors[stayer].v}
	from := [2]float64{s.actors[walker].u, s.actors[walker].v}
	to := []float64{from[0] + 3, from[1]}
	def := &scenes.Scene{ID: "t", Map: walk.Map, Nodes: []string{"n"}, Cast: map[string]scenes.Actor{
		stayer: {Stay: true, At: []float64{0, 0}},
		goner:  {At: []float64{0, 0}, Off: true},
		walker: {At: to, Face: "NE"},
	}}

	cut := *def
	cut.Cut = true
	assert.False(t, g.adoptScene(&cut, 0), "a Cut never takes over the walk's stage")

	require.True(t, g.adoptScene(def, 0))
	assert.Equal(t, stayAt, [2]float64{s.actors[stayer].u, s.actors[stayer].v}, "Stay keeps the place")
	assert.True(t, s.actors[goner].gone, "Off takes the person off the stage")
	assert.Equal(t, from, [2]float64{s.actors[walker].u, s.actors[walker].v}, "a far-off person does not jump")
	assert.NotEmpty(t, s.paths[walker], "they walk to their place")
	assert.True(t, s.settle, "the first line waits for them")
	for i := 0; i < 600 && (s.actors[walker].walk != nil || len(s.paths[walker]) > 0); i++ {
		g.updateScene(1.0 / 60)
	}
	assert.InDelta(t, to[0], s.actors[walker].u, .01)
	assert.InDelta(t, to[1], s.actors[walker].v, .01)
	g.updateScene(1.0 / 60)
	assert.Equal(t, "NE", s.actors[walker].dir, "they turn as the cast says once they stand there")
}

// Every same-room return the audit found with nobody moving (the story goes on in the room it stood in) stays lit:
// from the scene or event's first line until the walk is back, the stage never darkens.
func TestSameRoomReturnsStayLit(t *testing.T) {
	for _, c := range []struct{ name, stage, from string }{
		{"ch05 elder in the town", "start:c05-walk-town,talk:노인", "c05-h1"},
		{"ch13 woman at the street", "start:c13-walk,talk:약방 아낙", "c13-w2"},
	} {
		t.Run(c.name, func(t *testing.T) {
			frames := playContinuity(t, c.stage, 1500)
			begin := firstFrame(frames, c.from)
			require.GreaterOrEqual(t, begin, 0, "the flow reaches %s", c.from)
			back := -1
			for i := begin; i < len(frames); i++ {
				if frames[i].mode == "walk" && frames[i].node != "" && i > begin+10 {
					back = i
					break
				}
			}
			require.Greater(t, back, begin, "the walk comes back")
			var dark []int
			for _, f := range frames[begin:min(back+60, len(frames))] {
				if f.leave || f.fade > .05 {
					dark = append(dark, f.f)
				}
			}
			assert.Empty(t, dark, "frames where the room darkens")
		})
	}
}

// Where a scene hands the story back to the walk, the walk is the stage on screen unchanged (nothing to hide) only
// when the same people stand where they stood; a walk that adds guards, or an event whose party is not on the walk,
// still goes through black.
func TestWalkSeamlessAtSceneEnds(t *testing.T) {
	for _, c := range []struct {
		name, stage string
		want        bool
	}{
		{"ch08 office: the walk adds two guards", "start:c08-p12", false},
		{"ch24 office: the scene's party is not on the walk", "start:c24-s08", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			d, err := NewDemo("../..", DemoOptions{Name: "explore", Stage: c.stage})
			require.NoError(t, err)
			g := d.g
			for f := 0; f < 90; f++ { // the scene opens out of black and its beats play out
				d.step(f)
				g.advance(1.0 / 60)
				if o := g.S.View(); o.Phase == "scenario" && isTalk(o.Dialogue.Kind) && !g.exploring() {
					g.enterScene(o)
				}
			}
			require.NotNil(t, g.scene)
			g.scene.node = g.S.View().Node
			assert.Equal(t, c.want, g.walkSeamless())
		})
	}
}
