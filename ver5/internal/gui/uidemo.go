package gui

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"

	"github.com/hajimehoshi/ebiten/v2"
)

// Demo runs one piece of UI inside a real Game for cmd/_uidemo: a setup puts the game in
// the state the feature needs, then every frame steps the game by 1/60 s, draws it and
// saves the frame. The game code paths are the real ones, so a matching demo is a
// matching game.
type Demo struct {
	g             *Game
	frames, every int
	out           string
	frame         int
	saved         bool
	step          func(frame int) // optional per-frame hook, run before the game advances
	stepped       int
	overlay       func(dst *ebiten.Image)
	err           error
	// stage and duel are the data ids the demos play; the engine never names content, so
	// cmd/_uidemo passes them in
	stage, duel string
	// letterbox holds the 1280×850-aspect frame when the demo screen has another shape, the
	// way Ebitengine fits the game into a window
	letterbox *ebiten.Image
}

// DemoOptions say which demo to play and how. Frames go to Out every Every frames; Every <= 0
// saves nothing, for a host that captures the frames itself (cmd/_uidemo-vm). UIScale is the
// UI size setting in percent; 0 keeps the default.
type DemoOptions struct {
	Name, Out   string
	Stage, Duel string
	Frames      int
	Every       int
	UIScale     int
}

// demos are the UI pieces cmd/_uidemo can show, by name.
var demos = map[string]func(d *Demo) error{
	"fonts": demoFonts,
}

// DemoNames lists the demos in name order.
func DemoNames() []string {
	out := []string{}
	for k := range demos {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// NewDemo builds the game from the ver5 directory root and prepares the demo, which plays in
// battle o.Stage (and o.Duel, for the duel demo).
func NewDemo(root string, o DemoOptions) (*Demo, error) {
	name, out, stage, duel, frames, every := o.Name, o.Out, o.Stage, o.Duel, o.Frames, o.Every
	setup, ok := demos[name]
	if !ok {
		return nil, fmt.Errorf("unknown demo %q (have %v)", name, DemoNames())
	}
	d, err := content.Load(filepath.Join(root, "assets", "data"))
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "uidemo")
	if err != nil {
		return nil, err
	}
	s, err := session.New(d, dir)
	if err != nil {
		return nil, err
	}
	g, err := New(s, filepath.Join(root, "assets", "fonts", "NotoSansKR.ttf"), 1, "")
	if err != nil {
		return nil, err
	}
	if every > 0 {
		if err = os.MkdirAll(out, 0755); err != nil {
			return nil, err
		}
	}
	cursorPos = func() (int, int) { return -100, -100 }
	if stage == "" && len(d.Stages) > 0 {
		stage = d.Stages[0].ID
	}
	for _, st := range d.Stages {
		if duel == "" && len(st.Duels) > 0 {
			duel = st.Duels[0].ID
		}
	}
	demo := &Demo{g: g, frames: frames, every: every, out: out, stepped: -1, stage: stage, duel: duel}
	if err = setup(demo); err != nil {
		return nil, err
	}
	if o.UIScale > 0 {
		g.S.Settings.UIScale = o.UIScale
	}
	return demo, nil
}

// battle opens stage id straight into its first battle turn, allies deployed by plan.
func (d *Demo) battle(id string) error {
	g := d.g
	g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
	g.editorOpen(g.S.Data.StageIndex(id))
	g.editorTrial()
	e := g.S.Engine
	if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
		return err
	}
	plan, err := e.DeploymentPlan()
	if err != nil {
		return err
	}
	for _, c := range plan {
		if _, err = e.Apply(c); err != nil {
			return err
		}
	}
	g.editorReturn = nil // the trial's "back to editor" button is not part of the battle screen
	g.ensureField(e.Observe())
	g.sync(e.Observe())
	g.modals = nil
	return nil
}

// firstAlly is the first allied unit on the field.
func (d *Demo) firstAlly() string {
	for _, u := range d.g.S.Engine.Observe().UnitViews {
		if u.Faction == "ally" {
			return u.ID
		}
	}
	return ""
}

// look centres the camera on a unit.
func (d *Demo) look(id string) {
	if v := d.g.vis[id]; v != nil {
		d.g.camX, d.g.camY = d.g.field.Center(v.u, v.v)
	}
}

// Layout takes the screen as given, so a host can play the demo at a phone's size.
func (d *Demo) Layout(w, h int) (int, int) {
	scale = min(float64(w)/Width, float64(h)/Height)
	return w, h
}

func (d *Demo) Update() error {
	if d.err != nil {
		return d.err
	}
	if d.frame >= d.frames {
		return ebiten.Termination
	}
	if !d.saved && d.frame > 0 {
		return nil // wait for the previous frame to be drawn
	}
	if d.frame > 0 {
		d.saved = false
	}
	if d.step != nil && d.stepped != d.frame {
		d.stepped = d.frame
		d.step(d.frame)
	}
	d.g.advance(1.0 / 60)
	return nil
}

func (d *Demo) Draw(dst *ebiten.Image) {
	if d.saved {
		return
	}
	frame := dst
	if fw, fh := int(math.Ceil(Width*scale)), int(math.Ceil(Height*scale)); fw != dst.Bounds().Dx() || fh != dst.Bounds().Dy() {
		if d.letterbox == nil || d.letterbox.Bounds().Dx() != fw || d.letterbox.Bounds().Dy() != fh {
			d.letterbox = ebiten.NewImage(fw, fh)
		}
		d.letterbox.Clear()
		frame = d.letterbox
	}
	d.g.Draw(frame)
	if d.overlay != nil {
		d.overlay(frame)
	}
	if frame != dst {
		dst.Fill(color.Black)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(dst.Bounds().Dx()-frame.Bounds().Dx())/2, float64(dst.Bounds().Dy()-frame.Bounds().Dy())/2)
		dst.DrawImage(frame, op)
	}
	if d.every > 0 && d.frame%d.every == 0 {
		img := image.NewRGBA(dst.Bounds())
		dst.ReadPixels(img.Pix)
		f, err := os.Create(filepath.Join(d.out, fmt.Sprintf("frame-%04d.png", d.frame)))
		if err == nil {
			err = png.Encode(f, img)
			f.Close()
		}
		if err != nil {
			d.err = err
		}
	}
	d.saved = true
	d.frame++
}

// demoFonts shows the bundled faces over the battle screen.
func demoFonts(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	d.look(d.firstAlly())
	g := d.g
	d.overlay = func(dst *ebiten.Image) {
		rect(dst, 300, 250, 680, 330, color.NRGBA{R: 16, G: 12, B: 10, A: 230})
		write(dst, "제7턴 일기토 402 +220", g.brush(84), 330, 262, color.White)
		write(dst, "황건적의 난 · 유비군 차례", g.serif(30, 900), 330, 400, gold)
		write(dst, "형님, 화웅의 목은 술이 식기 전에 베어 오겠습니다.", g.serif(22, 700), 330, 460, ink)
		g.label(dst, "본문 글꼴(Noto Sans KR)은 그대로", 330, 520, 18, muted)
	}
	return nil
}

// pair moves the first enemy next to the first ally and frames them, like the
// mockup's 장비 and 황건 도적 pair.
func (d *Demo) pair() (ally, enemy string, err error) {
	g := d.g
	snap := g.S.Engine.Snapshot()
	var ax, ay int
	for _, u := range snap.Units {
		if u.Faction == "ally" && ally == "" {
			ally, ax, ay = u.ID, u.X, u.Y
		}
	}
	for i, u := range snap.Units {
		if u.Faction == "enemy" && enemy == "" {
			enemy = u.ID
			snap.Units[i].X, snap.Units[i].Y = ax+1, ay
		}
	}
	if g.S.Engine, err = core.Restore(g.S.Data, snap); err != nil {
		return
	}
	g.sync(g.S.Engine.Observe())
	g.camX, g.camY = g.field.Center(float64(ax)+.5, float64(ay)+.5)
	return
}

// setHP changes a unit's HP in the engine, so the screen sync keeps it, and returns its max.
func (d *Demo) setHP(id string, k float64) (hp, maxHP int) {
	g := d.g
	snap := g.S.Engine.Snapshot()
	for i, u := range snap.Units {
		if u.ID == id {
			if v := g.unitView(g.S.Engine.Observe(), id); v != nil {
				maxHP = v.Stats.MaxHP
			}
			hp = int(float64(maxHP) * k)
			snap.Units[i].HP = hp
		}
	}
	if e, err := core.Restore(g.S.Data, snap); err == nil {
		g.S.Engine = e
	} else {
		d.err = err
	}
	return hp, maxHP
}

// hit applies an HP change to unit id from fraction from to fraction to and shows it.
func (d *Demo) hit(actor, id string, from, to float64, crit bool) {
	before, _ := d.setHP(id, from)
	d.g.sync(d.g.S.Engine.Observe())
	after, _ := d.setHP(id, to)
	events := []core.Event{}
	if crit {
		events = append(events, core.Event{Kind: "critical", Actor: actor, Target: id})
	}
	if after < before {
		events = append(events, core.Event{Kind: "damage", Actor: actor, Target: id, Amount: before - after})
	} else {
		events = append(events, core.Event{Kind: "heal", Actor: actor, Target: id, Amount: after - before})
	}
	d.g.strike(nil, events)
}

// edit changes the engine state through a snapshot and resyncs the screen.
func (d *Demo) edit(f func(s *core.State)) {
	g := d.g
	snap := g.S.Engine.Snapshot()
	f(&snap)
	e, err := core.Restore(g.S.Data, snap)
	if err != nil {
		d.err = err
		return
	}
	g.S.Engine = e
	g.sync(e.Observe())
}
