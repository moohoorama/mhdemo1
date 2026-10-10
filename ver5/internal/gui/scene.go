package gui

import (
	"encoding/json"
	"image/color"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/core"
	"srpg/internal/sprite"
)

// sceneFile is assets/scenes.json: scenario scenes played on small maps with the
// officers' scene (평복) sprites. It is GUI data, not rules content.
type sceneFile struct {
	Scenes map[string]sceneDef
}

type sceneDef struct {
	Map   string
	Cast  map[string][]any // name -> [x, y, facing, faction?, sprite?]
	Beats [][][]sceneStep  // per speaker line: groups of steps run before it
}

type sceneStep struct {
	Actor  string
	Move   []float64
	Face   string
	Play   string
	Emote  string
	Wait   float64
	Effect string    // petals or fire, until the scene changes; "none" clears them
	At     []float64 // effect position [x, y] (fire); petals fall over the whole stage
}

func loadScenes(path string) *sceneFile {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var f sceneFile
	if json.Unmarshal(b, &f) != nil {
		return nil
	}
	return &f
}

// scene is the running scene: its actors stay put across nodes that share a map.
type scene struct {
	mapID   string
	node    string
	actors  map[string]*unitVis
	groups  [][]sceneStep
	gi      int
	begun   bool
	wait    float64
	paths   map[string][][2]float64
	plays   map[string]string
	emotes  map[string]*emote
	effects []sceneEffect
	spoke   map[string]bool
}

type sceneEffect struct {
	kind string
	u, v float64
	t    float64
}

type emote struct {
	text string
	t    float64
}

const emoteTime = 1.4

// enterScene sets up the scene of a dialogue node; false when the node has none.
func (g *Game) enterScene(o core.Observation) bool {
	if g.scenes == nil {
		return false
	}
	def, ok := g.scenes.Scenes[o.Node]
	if !ok || g.assets.Maps[def.Map] == nil {
		return false
	}
	if g.scene != nil && g.scene.node == o.Node {
		return true
	}
	if g.scene == nil || g.scene.mapID != def.Map || g.field == nil || g.field.Map.ID != def.Map {
		g.field = NewField(g.assets, g.assets.Maps[def.Map])
		g.scene = &scene{mapID: def.Map, actors: map[string]*unitVis{}}
	}
	s := g.scene
	s.node = o.Node
	s.paths, s.plays, s.emotes, s.effects = map[string][][2]float64{}, map[string]string{}, map[string]*emote{}, nil
	for name, at := range def.Cast {
		if len(at) < 3 {
			continue
		}
		x, _ := at[0].(float64)
		y, _ := at[1].(float64)
		dir, _ := at[2].(string)
		sprite := ""
		if len(at) > 4 {
			sprite, _ = at[4].(string)
		}
		faction := ""
		if len(at) > 3 {
			faction, _ = at[3].(string)
		}
		art := g.actorArt(name, sprite, faction)
		if art == nil {
			continue
		}
		v := s.actors[name]
		if v == nil {
			v = &unitVis{id: name, ally: true, dir: "SW", anim: "idle", dying: -1, shownHP: 1, maxHP: 1, actor: true}
			s.actors[name] = v
		}
		if v.art != art {
			v.art, v.top = art, art.TopOffset()
			v.play("idle")
		}
		v.u, v.v, v.dir, v.walk = x, y, dir, nil
		v.lift = g.field.Lift(int(x), int(y))
	}
	g.zoom = 2
	var cx, cy float64
	for _, v := range s.actors {
		x, y := g.field.Center(v.u, v.v)
		cx, cy = cx+x/float64(len(s.actors)), cy+y/float64(len(s.actors))
	}
	g.camX, g.camY = cx, cy-20
	g.startLine(0)
	return true
}

// actorArt is the sprite a cast member stands in: the cast's own pick (any sprite id: extras,
// soldiers, or a battle sprite for a fight), else the character's scene, battle or class
// sprite, in the character's faction colour unless the cast names another.
func (g *Game) actorArt(name, pick, faction string) *sprite.Art {
	def, c, _ := g.character(name)
	if faction == "" {
		faction = c.Look.Faction
	}
	return g.sceneArt(def, pick, faction)
}

// startLine queues the beats that come before speaker line i of the current node.
func (g *Game) startLine(i int) {
	s := g.scene
	if s == nil {
		return
	}
	s.groups, s.gi, s.begun, s.spoke = nil, 0, false, map[string]bool{}
	if def := g.scenes.Scenes[s.node]; i < len(def.Beats) {
		s.groups = def.Beats[i]
	}
}

// sceneBusy reports whether beats are still playing (the line waits for them).
func (g *Game) sceneBusy() bool { return g.scene != nil && g.scene.gi < len(g.scene.groups) }

func (g *Game) updateScene(dt float64) {
	s := g.scene
	if s == nil {
		return
	}
	for _, v := range s.actors {
		v.update(dt)
		if p := s.paths[v.id]; v.walk == nil && len(p) > 0 {
			g.walkTo(v, p[0][0], p[0][1])
			s.paths[v.id] = p[1:]
		} else if v.walk == nil && len(p) == 0 && v.anim == "walk" {
			v.play("idle")
		}
	}
	for i := range s.effects {
		s.effects[i].t += dt
	}
	for k, e := range s.emotes {
		if e.t += dt; e.t > emoteTime {
			delete(s.emotes, k)
		}
	}
	if !g.sceneBusy() {
		return
	}
	if !s.begun {
		s.begun = true
		for _, st := range s.groups[s.gi] {
			g.startStep(st)
		}
		return
	}
	s.wait -= dt
	if s.wait > 0 {
		return
	}
	for name, v := range s.actors {
		if len(s.paths[name]) > 0 || v.walk != nil || s.plays[name] != "" && v.anim == s.plays[name] {
			return
		}
	}
	s.plays = map[string]string{}
	s.gi++
	s.begun = false
}

func (g *Game) startStep(st sceneStep) {
	s := g.scene
	v := s.actors[st.Actor]
	if st.Wait > 0 {
		s.wait = math.Max(s.wait, st.Wait)
	}
	switch {
	case st.Effect == "":
	case st.Effect == "none":
		s.effects = nil
	case len(st.At) == 2:
		s.effects = append(s.effects, sceneEffect{kind: st.Effect, u: st.At[0], v: st.At[1]})
	case v != nil:
		s.effects = append(s.effects, sceneEffect{kind: st.Effect, u: v.u, v: v.v})
	default:
		s.effects = append(s.effects, sceneEffect{kind: st.Effect})
	}
	if v == nil {
		return
	}
	if len(st.Move) == 2 {
		s.paths[v.id] = scenePath(v.u, v.v, st.Move[0], st.Move[1])
	}
	if st.Face != "" {
		v.dir = st.Face
	}
	if anim := motion(v.art, st.Play); anim != "" {
		v.play(anim)
		if !v.art.Animations[anim].Loop {
			s.plays[v.id] = anim
		}
	}
	if st.Emote != "" {
		s.emotes[v.id] = &emote{text: st.Emote}
	}
}

// motion is the animation a scene step asks for: the sprite's own, or its one-shot pose
// for the old non-combat motion names. A sprite without that pose just stays idle.
func motion(art *sprite.Art, play string) string {
	if _, ok := art.Animations[play]; ok && play != "" {
		return play
	}
	switch play {
	case "salute", "toast", "surprise", "nod", "talk", "exhausted":
		if _, ok := art.Animations["action"]; ok {
			return "action"
		}
	}
	return ""
}

// scenePath walks straight legs, along u then along v, one cell per step; the last step of
// a leg covers what is left when the target lies between cells.
func scenePath(u, v, tu, tv float64) [][2]float64 {
	path := [][2]float64{}
	for u != tu {
		u += math.Copysign(math.Min(1, math.Abs(tu-u)), tu-u)
		path = append(path, [2]float64{u, v})
	}
	for v != tv {
		v += math.Copysign(math.Min(1, math.Abs(tv-v)), tv-v)
		path = append(path, [2]float64{u, v})
	}
	return path
}

// skipBeats finishes the running beats at once: walkers arrive, motions end.
func (g *Game) skipBeats() {
	s := g.scene
	for s != nil && g.sceneBusy() {
		if !s.begun {
			for _, st := range s.groups[s.gi] {
				g.startStep(st)
			}
		}
		for name, v := range s.actors {
			if p := s.paths[name]; len(p) > 0 {
				last := p[len(p)-1]
				v.face(last[0], last[1])
				v.u, v.v = last[0], last[1]
			} else if v.walk != nil {
				v.u, v.v = v.walk.tu, v.walk.tv
			}
			v.walk, v.lift = nil, g.field.Lift(int(v.u), int(v.v))
			v.play("idle")
		}
		s.paths, s.plays, s.wait = map[string][][2]float64{}, map[string]string{}, 0
		s.gi++
		s.begun = false
	}
}

// drawScene draws the scene map and its actors, the speakers' bubbles and emotes.
func (g *Game) drawScene(dst *ebiten.Image, speakers []string) {
	s := g.scene
	actors := make([]*unitVis, 0, len(s.actors))
	for _, v := range s.actors {
		actors = append(actors, v)
	}
	talking := map[string]bool{}
	if !g.sceneBusy() {
		for _, name := range speakers {
			talking[name] = true
		}
	}
	for name, v := range s.actors { // a speaker gestures once when their line comes up
		if talking[name] && !s.spoke[name] && v.walk == nil && v.anim == "idle" {
			s.spoke[name] = true
			v.play("action")
		}
	}
	g.field.Draw(g.clockMS, actors, nil)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-g.camX, -g.camY)
	op.GeoM.Scale(g.zoom, g.zoom)
	op.GeoM.Translate(float64(Width/2), float64(Height/2))
	op.Filter = ebiten.FilterPixelated
	drawScaled(dst, g.field.Canvas, op)
	g.drawSceneEffects(dst)
	for name := range talking {
		if v := s.actors[name]; v != nil {
			g.bubble(dst, v, "", true)
		}
	}
	for name, e := range s.emotes {
		if v := s.actors[name]; v != nil {
			g.bubble(dst, v, e.text, false)
		}
	}
}

// bubble is a small speech balloon at the actor's upper right: animated dots while
// talking, or an emote mark.
func (g *Game) bubble(dst *ebiten.Image, v *unitVis, mark string, talking bool) {
	x, y := g.field.Center(v.u, v.v)
	sx, sy := g.toScreen(x, y-v.lift-v.top)
	z := float32(g.zoom)
	bx, by := float32(sx)+6*z, float32(sy)-14*z
	w, h := 18*z, 11*z
	var p vector.Path
	p.MoveTo(bx+2*z, by)
	p.LineTo(bx+w-2*z, by)
	p.QuadTo(bx+w, by, bx+w, by+2*z)
	p.LineTo(bx+w, by+h-2*z)
	p.QuadTo(bx+w, by+h, bx+w-2*z, by+h)
	p.LineTo(bx+6*z, by+h)
	p.LineTo(bx+1*z, by+h+4*z) // tail toward the head
	p.LineTo(bx+3*z, by+h)
	p.LineTo(bx+2*z, by+h)
	p.QuadTo(bx, by+h, bx, by+h-2*z)
	p.LineTo(bx, by+2*z)
	p.QuadTo(bx, by, bx+2*z, by)
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(color.NRGBA{R: 250, G: 246, B: 232, A: 240})
	fillPath(dst, &p, op)
	op = &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(color.NRGBA{R: 60, G: 40, B: 30, A: 255})
	strokePath(dst, &p, vector.StrokeOptions{Width: 1.2}, op)
	if talking {
		for i := 0; i < 3; i++ {
			k := math.Sin(g.clockMS/140 - float64(i)*.9)
			fillCircle(dst, bx+(5+float32(i)*4)*z, by+h/2-float32(max(0, k))*1.5*z, 1.2*z, color.NRGBA{R: 60, G: 40, B: 30, A: 255})
		}
		return
	}
	g.centered(dst, mark, float64(bx+w/2), float64(by)-1, float64(10*z), color.NRGBA{R: 160, G: 40, B: 30, A: 255})
}

var (
	petalColors = []color.NRGBA{{R: 242, G: 167, B: 184, A: 255}, {R: 255, G: 214, B: 222, A: 255}, {R: 215, G: 119, B: 144, A: 255}}
	flameColors = []color.NRGBA{{R: 255, G: 232, B: 154, A: 255}, {R: 255, G: 160, B: 50, A: 255}, {R: 214, G: 64, B: 40, A: 255}}
)

// drawSceneEffects draws the stage effects: peach petals drifting across the whole screen,
// and fires burning at map positions. Particles are pure functions of the effect's age, so
// skipping beats or replaying a line shows the same thing.
func (g *Game) drawSceneEffects(dst *ebiten.Image) {
	z := float32(g.zoom)
	for _, e := range g.scene.effects {
		switch e.kind {
		case "petals":
			for i := 0; i < 48; i++ {
				k := float64(i)
				fall, drift := 34+11*math.Mod(k*7.3, 5), 26+9*math.Mod(k*3.1, 4)
				x := math.Mod(k*97.13+drift*e.t+12*math.Sin(e.t*1.3+k), Width+40) - 20
				y := math.Mod(k*61.7+fall*e.t, Height+40) - 20
				w := 2.5 + float32(math.Abs(math.Sin(e.t*3+k)))*2 // tumbling
				c := petalColors[i%len(petalColors)]
				rect(dst, float32(x), float32(y), w*z, 1.6*z, c)
			}
		case "fire":
			x, y := g.field.Center(e.u, e.v)
			sx, sy := g.toScreen(x, y)
			glow(dst, float32(sx), float32(sy)-8*z, 22*z, flameColors[1], .7+.2*math.Sin(e.t*9))
			for i := 0; i < 14; i++ {
				k := float64(i)
				life := math.Mod(e.t*1.6+k*.37, 1) // 0 at the base, 1 burnt out at the top
				px := float32(sx) + float32(math.Sin(k*2.4)*7+math.Sin(e.t*5+k)*2*(life))*z
				py := float32(sy) - float32(life*26)*z
				r := float32((1-life)*4.5+1) * z
				c := flameColors[min(2, int(life*3))]
				fillCircle(dst, px, py, r, with8(c, 1-life*.6))
			}
			for i := 0; i < 5; i++ { // smoke
				k := float64(i)
				life := math.Mod(e.t*.5+k*.2, 1)
				px := float32(sx) + float32(math.Sin(k*1.7+e.t)*6)*z
				py := float32(sy) - float32(26+life*30)*z
				fillCircle(dst, px, py, float32(3+life*6)*z, color.NRGBA{R: 60, G: 56, B: 52, A: uint8(110 * (1 - life))})
			}
		}
	}
}
