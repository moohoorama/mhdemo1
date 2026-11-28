package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/scenes"
)

// titleReel plays the title screen's backdrop (assets/scenes/title/reel.json): its shots one after another on the
// story-scene player, each lingering when its beats are done, darkening to black and the next opening out of it,
// round and round while the title is up. A shot is seen as if remembered: bright in a circle, darkening past its
// rim, the circle somewhere else on the screen each time but never cut by its edge.
type titleReel struct {
	reel  *scenes.Reel
	i     int
	scene *scene  // the shot on the stage; nil until the title opens one
	hold  float64 // seconds the finished shot has lingered
	t     float64 // seconds since the shot's last group began
	out   float64 // 0 → 1: the finished shot darkening before the next
	round int     // shots opened so far: picks each circle's place
	// the remembered circle, in world px of the battle viewport
	cx, cy, r float64
	mask      *ebiten.Image
	film      *ebiten.Image // the shot drawn before it is faded like old film (title_film.go)
	grain     *ebiten.Image
}

const (
	reelFade     = .6
	reelRadius   = .42 // the circle's radius as a share of the screen height, unless the shot gives one
	reelMargin   = .04 // the least gap between the circle and the screen's edge, as a share of the screen height
	reelMaskSize = 256
	reelInner    = .62 // share of the radius that stays fully bright
	reelFlash    = .3  // share of a scene's white flash the shots show
	reelMenuEdge = .68 // the circle's right edge stays left of this share of the screen width, clear of the menu
)

func (g *Game) loadTitleReel(path string) {
	if r, err := scenes.LoadReel(path); err == nil {
		g.reel = &titleReel{reel: r}
	}
}

// onTitle reports whether the plain title screen is up: no game, and nothing opened over it from its menu.
func (g *Game) onTitle() bool {
	return (g.S.Screen == "title" || g.S.Engine == nil) && g.editor == nil && g.tour == nil && g.world == nil &&
		g.gallery == nil && g.player.Duel == nil
}

// reelPlaying reports whether the stage shows the title reel's shot.
func (g *Game) reelPlaying() bool {
	return g.reel != nil && g.reel.scene != nil && g.scene == g.reel.scene
}

// updateTitleReel opens, advances and closes the title reel's shots; the scene player itself moves the beats.
func (g *Game) updateTitleReel(dt float64) {
	r := g.reel
	if r == nil {
		return
	}
	if !g.onTitle() {
		if g.reelPlaying() { // the game starts: its first scene opens on a fresh stage
			g.scene = nil
			g.field.dispose()
			g.field = nil
		}
		r.scene = nil
		return
	}
	if !g.reelPlaying() {
		g.openShot(r.i)
		return
	}
	if s := g.scene; s.gi >= len(s.groups)-1 {
		r.t += dt
	}
	shot := r.reel.Shots[r.i]
	ending := shot.End > 0 && r.t >= shot.End // the people play on while it darkens
	if !ending && g.sceneBusy() {
		return
	}
	if !ending && r.hold < shot.Hold {
		r.hold += dt
		return
	}
	if r.out += dt / reelFade; r.out >= 1 {
		r.i = (r.i + 1) % len(r.reel.Shots)
		g.openShot(r.i)
	}
}

// openShot stages shot i, places its circle and starts its beats.
func (g *Game) openShot(i int) {
	r := g.reel
	shot := r.reel.Shots[i]
	def := shot.Def()
	if g.mapArt(def.Map) == nil {
		r.i = (i + 1) % len(r.reel.Shots)
		return
	}
	r.round++
	r.place(shot, g.battleViewport().Dx(), g.battleViewport().Dy())
	g.openScene(def)
	g.setZoomImmediate(max(1, shot.Zoom))
	g.scene.node, g.scene.held = def.Nodes[0], map[string]bool{}
	g.startLine(0)
	r.i, r.scene, r.hold, r.out, r.t = i, g.scene, 0, 0, 0
	const step = 1.0 / 30
	for t := 0.0; t < shot.Lead; t += step { // played in the dark: it opens with them already on the move
		g.updateScene(step)
		g.scene.fade = 1
		if s := g.scene; s.gi >= len(s.groups)-1 {
			r.t += step
		}
	}
	g.camX, g.camY = g.sceneCam()
}

// place picks where the shot's circle stands: anywhere it fits whole on a w×h screen.
func (r *titleReel) place(shot *scenes.Shot, w, h int) {
	k := shot.Radius
	if k <= 0 {
		k = reelRadius
	}
	r.r = k * float64(h)
	m := reelMargin * float64(h)
	rnd := rand.New(rand.NewPCG(uint64(r.round), 7))
	pick := func(size, most float64) float64 {
		lo, hi := r.r+m, min(most-r.r, size-r.r-m)
		if hi <= lo {
			return lo
		}
		return lo + rnd.Float64()*(hi-lo)
	}
	r.cx, r.cy = pick(float64(w), reelMenuEdge*float64(w)), pick(float64(h), float64(h))
}

// reelOffset is how far the camera looks past the shot's own point, in canvas px, so that point sits in the
// middle of the circle.
func (g *Game) reelOffset() (float64, float64) {
	r, vp := g.reel, g.battleViewport()
	return (float64(vp.Dx())/2 - r.cx) / g.zoom, (float64(vp.Dy())/2 - r.cy) / g.zoom
}

// drawTitleReel draws the playing shot like an old film in its remembered circle, then a veil deepening toward the menu.
func (g *Game) drawTitleReel(dst *ebiten.Image) {
	r := g.reel
	if b := dst.Bounds(); r.film == nil || r.film.Bounds() != b {
		if r.film != nil {
			r.film.Deallocate()
		}
		r.film = ebiten.NewImage(b.Dx(), b.Dy())
	}
	r.film.Clear()
	g.drawScene(r.film, nil)
	g.drawFilm(dst, r.film)
	if r.mask == nil {
		r.mask = reelMask()
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2*r.r*worldPx/reelMaskSize, 2*r.r*worldPx/reelMaskSize)
	op.GeoM.Translate((r.cx-r.r)*worldPx+drawOX, (r.cy-r.r)*worldPx+drawOY)
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(r.mask, op)
	inWorld(func() { // past the circle's square: black
		w, h := float32(worldW), float32(worldH)
		x0, y0, x1, y1 := float32(r.cx-r.r), float32(r.cy-r.r), float32(r.cx+r.r), float32(r.cy+r.r)
		rect(dst, 0, 0, w, y0+3, color.Black) // overlapping the mask's rim, which the filtering softens
		rect(dst, 0, y1-3, w, h-y1+3, color.Black)
		rect(dst, 0, y0, x0+3, y1-y0, color.Black)
		rect(dst, x1-3, y0, w-x1+3, y1-y0, color.Black)
	})
	g.blackOut(dst, max(g.scene.fade, r.out))
	const strips = 64
	for i := range strips {
		x0, x1 := uiW*i/strips, uiW*(i+1)/strips
		k := float32(i) / (strips - 1)
		a := .55 * max(0, (k-.45)/.55)
		rect(dst, float32(x0), 0, float32(x1-x0), float32(uiH), color.NRGBA{A: uint8(255 * a)})
	}
}

// reelMask is the remembered circle's darkness: clear in the middle, deepening past reelInner of the radius to
// black at the rim and beyond.
func reelMask() *ebiten.Image {
	const n = reelMaskSize
	pix := make([]byte, n*n*4)
	for y := range n {
		for x := range n {
			d := math.Hypot(float64(x)+.5-n/2, float64(y)+.5-n/2) / (n / 2)
			k := math.Max(0, math.Min(1, (d-reelInner)/(1-reelInner)))
			pix[(y*n+x)*4+3] = byte(255 * k * k * (3 - 2*k))
		}
	}
	img := ebiten.NewImage(n, n)
	img.WritePixels(pix)
	return img
}

// reelPicker is the debug build's row on the title to step through the reel's shots: the shot playing, and the
// shots before and after it a click away.
func (g *Game) reelPicker(dst *ebiten.Image) {
	r := g.reel
	if !g.Debug || r == nil || !g.reelPlaying() {
		return
	}
	n := len(r.reel.Shots)
	jump := func(d int) func() { return func() { g.openShot((r.i + d + n) % n) } }
	const x, y, h = 96, Height - 70, 40
	g.btn(dst, image.Rect(x, y, x+48, y+h), "◀", jump(-1))
	g.label(dst, fmt.Sprintf("%d/%d %s", r.i+1, n, r.reel.Shots[r.i].ID), x+62, y+8, 18, gold)
	g.btn(dst, image.Rect(x+250, y, x+298, y+h), "▶", jump(1))
}
