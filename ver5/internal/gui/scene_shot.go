package gui

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/scenes"
)

// sceneShot is the arrow of a Shoot step, flown over the scene map itself. The shooter draws the bow and holds it
// (a bubble over him, "어디……"), while the camera glides, easing in and out, until he stands at the lower left of
// the screen with the way ahead open; he lets go, the camera is caught napping for a few frames, then hurries after
// the arrow, keeping it at the lower left as it flies on a shallow arc toward the halberd standing on the stage (the
// scene's halberd effect). When it strikes: a white flash, the halberd is knocked over, the arrow spins off and
// drops slantwise; the camera settles on them, holds, and the scene fades back to the shooter.
type sceneShot struct {
	plan       scenes.Shoot
	tune       ShotTuning
	shooter    *unitVis
	t          float64 // seconds since the step started
	fx, fy     float64 // canvas point the arrow leaves from (the ground under it)
	tx, ty     float64 // canvas point it strikes (the halberd's foot)
	ok         bool    // there is a halberd to aim at
	panX, panY float64 // the camera when the bow was drawn
	camX, camY float64
	tick       float64 // time not yet stepped through the per-frame camera
	velX, velY float64 // the camera's speed (canvas px per second), carried into the landing spring
	struck     bool
}

// ShotTuning holds the timings of a Shoot step; cmd/_scene15 sets them from flags to try them out.
type ShotTuning struct {
	Aim       float64    // seconds the bow is held drawn while the camera glides
	Freeze    int        // frames the camera does not move after the arrow leaves
	Chase     float64    // c in S = S*c + A*(1-c), once per frame (1/60 s), as the arrow leaves: how far behind it stays
	ChaseEnd  float64    // c as the arrow arrives: lower, so the camera hurries up after it
	Damping   float64    // the camera's spring on the struck halberd: under 1 it overshoots and swings back
	Stiff     float64    // that spring's strength
	Hold      float64    // seconds on the struck halberd before the scene goes back
	Shooter   [2]float64 // where on the screen (fractions of its width and height) the shooter stands after the glide
	Arrow     [2]float64 // where on the screen the camera keeps the arrow
	DrawHold  float64    // seconds into the attack motion where the bow is fully drawn
	Release   float64    // seconds into the attack motion where the string is let go
	Flight    float64    // seconds in the air instead of the step's own Time, when > 0
	Peak      float64    // arc height instead of the step's own Peak, when > 0
	ThrowPeak float64    // how high the thrown halberd's arc climbs, canvas px (the higher, the steeper it lands)
	ThrowTime float64    // the thrown halberd's seconds in the air
}

// ShotTune is the Shoot timing in use.
var ShotTune = ShotTuning{Aim: 3, Freeze: 10, Chase: .94, ChaseEnd: .6, Damping: .4, Stiff: 60, Hold: 4,
	ThrowPeak: 40, ThrowTime: .55, Shooter: [2]float64{.25, .75}, Arrow: [2]float64{.1, .9},
	DrawHold: .2, Release: .46, Flight: 2.5, Peak: 160}

const (
	shotFadeOut = .4
	shotLaunch  = 30.0 // canvas px above the ground: a rider's bow
	shotMark    = 34.0 // where it strikes the halberd
)

func (s *sceneShot) length() float64 { return s.tune.Aim + s.plan.Time + s.tune.Hold }
func (s *sceneShot) flight() float64 { return s.t - s.tune.Aim }
func (s *sceneShot) hit() float64    { return s.flight() - s.plan.Time }

// at is the arrow's ground point and height ft seconds after it left: steady speed on a shallow arc.
func (s *sceneShot) at(ft float64) (float64, float64, float64) {
	k := math.Max(0, math.Min(1, ft/s.plan.Time))
	return s.fx + (s.tx-s.fx)*k, s.fy + (s.ty-s.fy)*k, shotLaunch + (shotMark-shotLaunch)*k + s.plan.Peak*4*k*(1-k)
}

// halberd is the scene's halberd effect, the arrow's mark.
func (g *Game) halberd() *sceneEffect {
	for i := range g.scene.effects {
		if e := &g.scene.effects[i]; e.kind == "halberd" || e.kind == "halberd_throw" {
			return e
		}
	}
	return nil
}

// camFor is the camera point that puts canvas point (x, y) at screen fractions (fx, fy).
func (g *Game) camFor(x, y float64, at [2]float64) (float64, float64) {
	return x - (at[0]-.5)*float64(worldW)/g.zoom, y - (at[1]-.5)*float64(worldH)/g.zoom
}

// startShot begins a Shoot step: the shooter draws the bow and holds it, the beats wait for the whole shot.
func (g *Game) startShot(v *unitVis, st scenes.Step) {
	s := g.scene
	p := *st.Shoot
	if ShotTune.Flight > 0 {
		p.Time = ShotTune.Flight
	}
	if ShotTune.Peak > 0 {
		p.Peak = ShotTune.Peak
	}
	if p.Time <= 0 {
		p.Time = 1
	}
	if p.Peak <= 0 {
		p.Peak = 60
	}
	sh := &sceneShot{plan: p, tune: ShotTune, shooter: v, panX: g.camX, panY: g.camY, camX: g.camX, camY: g.camY}
	if v != nil {
		sh.fx, sh.fy = g.field.Center(v.u, v.v)
		if anim := motion(v.art, "attack"); anim != "" {
			v.play(anim)
		}
		if p.Say != "" {
			s.emotes[v.id] = &emote{text: p.Say, life: sh.tune.Aim}
		}
	}
	if e := g.halberd(); e != nil {
		sh.tx, sh.ty = g.field.Center(e.u, e.v)
		sh.ok = true
	}
	s.shot = sh
	s.wait = math.Max(s.wait, sh.length())
}

// skipShot ends a shot at once, the halberd knocked over.
func (g *Game) skipShot() {
	if g.scene.shot == nil {
		return
	}
	if e := g.halberd(); e != nil && e.hit < 0 {
		e.hit = 10
	}
	g.scene.shot = nil
}

// updateShot runs the shot after the scene's own camera has moved: the bow held while the camera glides, the arrow's
// camera, the strike and the way back.
func (g *Game) updateShot(dt float64) {
	s := g.scene
	sh := s.shot
	if sh == nil {
		return
	}
	sh.t += dt
	tn := sh.tune
	if v := sh.shooter; v != nil && v.anim == "attack" && sh.flight() < 0 { // the bow stays drawn until the release
		v.animT = math.Min(v.animT, tn.DrawHold)
	}
	if v := sh.shooter; v != nil && v.anim == "attack" && sh.flight() >= 0 && v.animT < tn.Release {
		v.animT = tn.Release
	}
	if !sh.ok {
		if sh.t >= sh.length() {
			s.shot = nil
		}
		return
	}
	ft := sh.flight()
	if ft < 0 { // the glide speeds up and slows down again: n = (1 - cos(180° · t / T)) / 2
		ax, ay := g.camFor(sh.fx, sh.fy-shotLaunch, tn.Shooter)
		n := (1 - math.Cos(math.Pi*math.Min(1, sh.t/tn.Aim))) / 2
		sh.camX, sh.camY = sh.panX*(1-n)+ax*n, sh.panY*(1-n)+ay*n
	} else {
		var ax, ay float64
		if sh.hit() < 0 {
			x, y, h := sh.at(ft)
			ax, ay = g.camFor(x, y-h, tn.Arrow)
		} else {
			ax, ay = g.camFor(sh.tx, sh.ty-20, [2]float64{.5, .55})
			if !sh.struck {
				sh.struck = true
				s.shake = math.Max(s.shake, .3)
				if e := g.halberd(); e != nil {
					e.hit = 0
				}
			}
		}
		// in flight: S = S*c + A*(1-c) once per 1/60 s after the camera has missed the first Freeze frames, c going
		// from Chase down to ChaseEnd so the arrow first runs away and the camera then hurries after it; on the strike
		// a spring that keeps the camera's speed, so it runs past the halberd and swings back
		const frame = 1.0 / 60
		sh.tick += dt
		for sh.tick >= frame {
			sh.tick -= frame
			if sh.hit() < 0 {
				if ft < float64(tn.Freeze)/60 {
					continue
				}
				k := math.Min(1, ft/sh.plan.Time)
				c := tn.Chase + (tn.ChaseEnd-tn.Chase)*k
				nx, ny := sh.camX*c+ax*(1-c), sh.camY*c+ay*(1-c)
				sh.velX, sh.velY = (nx-sh.camX)/frame, (ny-sh.camY)/frame
				sh.camX, sh.camY = nx, ny
				continue
			}
			w := math.Sqrt(tn.Stiff)
			sh.velX += (tn.Stiff*(ax-sh.camX) - 2*tn.Damping*w*sh.velX) * frame
			sh.velY += (tn.Stiff*(ay-sh.camY) - 2*tn.Damping*w*sh.velY) * frame
			sh.camX += sh.velX * frame
			sh.camY += sh.velY * frame
		}
	}
	g.camX, g.camY = sh.camX, sh.camY
	if sh.t >= sh.length() {
		s.shot = nil
		s.fade = 1
		g.camX, g.camY = g.sceneCam()
	}
}

// drawShot draws the arrow in flight with its shadow and afterimages, then thrown off the halberd spinning and lying
// slantwise on the ground.
func (g *Game) drawShot(dst *ebiten.Image) {
	sh := g.scene.shot
	if sh == nil || !sh.ok || sh.flight() < 0 {
		return
	}
	fx := g.assets.FX["arrow"]
	if len(fx.Frames) == 0 {
		return
	}
	n := float64(len(fx.Frames))
	frame := func(dx, dy float64) int {
		return int(math.Round(math.Atan2(dy, dx)/(2*math.Pi)*n+n)) % len(fx.Frames)
	}
	dx, dy := sh.tx-sh.fx, sh.ty-sh.fy
	l := math.Max(1, math.Hypot(dx, dy))
	shadow := func(x, y float64) { // a thin dark line on the ground under the arrow, along its way
		for p := -4.0; p <= 4; p++ {
			px, py := g.toScreen(math.Round(x+dx/l*p*1.6), math.Round(y+dy/l*p*1.6))
			inWorld(func() {
				rect(dst, float32(px), float32(py), float32(g.zoom), float32(g.zoom), color.NRGBA{20, 26, 12, 90})
			})
		}
	}
	ft := sh.flight()
	if hit := sh.hit(); hit < 0 {
		x, y, h := sh.at(ft)
		x2, y2, h2 := sh.at(ft + .01)
		i := frame(x2-x, y2-h2-(y-h))
		shadow(x, y)
		for k := 3; k >= 1; k-- { // afterimages
			px, py, ph := sh.at(ft - float64(k)*.012)
			fx.drawAlpha(dst, i, px, py-ph, g.toScreen, g.zoom, .4/float64(k))
		}
		fx.draw(dst, i, x, y-h, g.toScreen, g.zoom)
		return
	}
	// thrown back off the blades, spinning, then down on the grass beside the halberd
	hit := sh.hit()
	const fall, up = .8, 140.0
	gravity := 2 * (shotMark + up*fall) / (fall * fall)
	k := math.Min(hit, fall)
	x, y := sh.tx-dx/l*22*k/fall, sh.ty-dy/l*22*k/fall
	h := math.Max(0, shotMark+up*k-gravity*k*k/2)
	i := frame(dx, dy+dx*.25) // lying along its way, tilted
	if hit < fall {
		i = (frame(-dx, -dy) + int(hit*30)) % len(fx.Frames)
	}
	shadow(x, y)
	fx.draw(dst, i, x, y-h, g.toScreen, g.zoom)
}

// drawShotScreen lays the strike's white flash and the closing fade over the whole screen.
func (g *Game) drawShotScreen(dst *ebiten.Image) {
	sh := g.scene.shot
	if sh == nil || !sh.ok {
		return
	}
	if h := sh.hit(); h >= 0 && h < .5 {
		rect(dst, 0, 0, float32(worldW), float32(worldH), color.NRGBA{255, 255, 255, uint8(255 * (1 - h/.5))})
	}
	if k := sh.t - (sh.length() - shotFadeOut); k > 0 {
		rect(dst, 0, 0, float32(worldW), float32(worldH), color.NRGBA{A: uint8(255 * math.Min(1, k/shotFadeOut))})
	}
}

// halberdArt is 방천화극, drawn dot by dot: the spear point, the two crescent blades, the red tassel and the shaft.
var halberdArt = []string{
	"......k......",
	".....ksk.....",
	".....ksk.....",
	"....ksmsk....",
	"....ksmdk....",
	"....ksmdk....",
	"....ksmdk....",
	".kk.ksmdk.kk.",
	"ksskksmdkkdsk",
	"ksmmkkmdkmmdk",
	".ksmmsmdmmdk.",
	"..ksmmmmmdk..",
	".ksmmsmdmmdk.",
	"ksmmkkmdkmmdk",
	"ksskkgggkkddk",
	".kk.kgggk.kk.",
	"....krrrk....",
	"...krrRrrk...",
	"...krRrRrk...",
	"..krrRrRrrk..",
	"...krRrRrk...",
	"....kRrRk....",
	".....kwk.....",
	".....kwk.....",
	".....kWk.....",
	".....kwk.....",
	".....kwk.....",
	".....kwk.....",
	".....kWk.....",
	".....kwk.....",
	".....kwk.....",
	".....kwk.....",
	".....kWk.....",
	".....kwk.....",
	".....kwk.....",
	".....kwk.....",
	".....kWk.....",
	".....kwk.....",
	".....kwk.....",
	".....kwk.....",
	".....kWk.....",
	".....kwk.....",
	".....kwk.....",
	".....kwk.....",
	".....kWk.....",
	".....kwk.....",
	".....kwk.....",
	".....kwk.....",
	".....kWk.....",
	".....kwk.....",
	".....kwk.....",
	".....kwk.....",
	".....kgk.....",
	".....kgk.....",
	".....kWk.....",
	".....kWk.....",
	"......k......",
}

var halberdColors = map[rune]color.RGBA{
	'k': {0x1c, 0x18, 0x14, 255}, 's': {0xe8, 0xec, 0xf0, 255}, 'm': {0xa8, 0xb0, 0xb8, 255}, 'd': {0x68, 0x70, 0x78, 255},
	'g': {0xd0, 0xa0, 0x30, 255}, 'r': {0xc8, 0x30, 0x30, 255}, 'R': {0x88, 0x20, 0x20, 255}, 'w': {0x8a, 0x5a, 0x2a, 255},
	'W': {0x5a, 0x38, 0x18, 255},
}

// halberdBury is how many rows of the head, from the point, go into the ground when it lands blade first: the point
// and the lower half of the crescents.
const halberdBury = 11

// halberdImage is the whole halberd; halberdStuck the part above the ground once its blades are driven in.
var halberdImage, halberdStuck = func() (*ebiten.Image, *ebiten.Image) {
	w, h := len(halberdArt[0]), len(halberdArt)
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for y, row := range halberdArt {
		for x, ch := range row {
			if c, ok := halberdColors[ch]; ok {
				im.SetRGBA(x, y, c)
			}
		}
	}
	whole := ebiten.NewImageFromImage(im)
	return whole, whole.SubImage(image.Rect(0, halberdBury, w, h)).(*ebiten.Image)
}()

// halberdThrow is the thrown halberd's seconds in the air.
func halberdThrow() float64 { return max(.05, ShotTune.ThrowTime) }

// drawHalberdAt draws img turned angle radians clockwise about its point (px, py), that point at canvas (x, y).
func (g *Game) drawHalberdAt(dst, img *ebiten.Image, px, py, x, y, angle float64) {
	sx, sy := g.toScreen(x, y)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest}
	op.GeoM.Translate(-px, -py)
	op.GeoM.Rotate(angle)
	op.GeoM.Scale(g.zoom, g.zoom)
	op.GeoM.Translate(math.Round(sx), math.Round(sy))
	drawScaled(dst, img, op)
}

// drawThrownHalberd draws the "halberd_throw" scene effect at canvas point (x, y), t seconds after it was thrown: it
// comes in from the south-west on an arc, point first, and drives its blades slantwise into the ground; once the
// arrow strikes it (hit >= 0) it is knocked out and falls flat.
func (g *Game) drawThrownHalberd(dst *ebiten.Image, x, y, t, hit float64) {
	sx, sy := x-12*32, y+12*16 // twelve cells south-west, where 여포 rides in from
	pos := func(k float64) (float64, float64) {
		return sx + (x-sx)*k, sy + (y-sy)*k - ShotTune.ThrowPeak*4*k*(1-k)
	}
	head := func(k float64) float64 { // clockwise from straight up, toward where it is going
		x1, y1 := pos(k - .01)
		x2, y2 := pos(k)
		return math.Atan2(x2-x1, -(y2 - y1))
	}
	cx := float64(len(halberdArt[0])) / 2
	if k := t / halberdThrow(); k < 1 {
		px, py := pos(k)
		g.drawHalberdAt(dst, halberdImage, cx, 0, px, py, head(math.Max(.01, k)))
		return
	}
	stuck := head(1)
	if hit < 0 {
		g.drawHalberdAt(dst, halberdStuck, cx, 0, x, y, stuck)
		return
	}
	// knocked out of the ground and over on its side, the head toward the north-east
	k := math.Min(1, hit/.5)
	lie := math.Atan2(32, 16)
	if lie < stuck {
		lie += 2 * math.Pi
	}
	g.drawHalberdAt(dst, halberdImage, cx, halberdBury, x, y, stuck+(lie-stuck)*k*k)
}

// halberdDrop is the straight drop of the "halberd" effect: thrown in from the sky, it stands upright.
const halberdDrop = .22

// drawPlantedHalberd draws the "halberd" scene effect at canvas point (x, y), t seconds after it was thrown.
func (g *Game) drawPlantedHalberd(dst *ebiten.Image, x, y, t float64) {
	k := math.Min(1, t/halberdDrop)
	foot := Sprite{Image: halberdImage, PivotX: float64(len(halberdArt[0])) / 2, PivotY: float64(len(halberdArt) - 1)}
	g.drawCanvasSprite(dst, foot, x, y-(1-k*k)*260, 1)
}

// sceneFlash is how white the screen is from the scene's flashes ("flash" effects and landing halberds).
func (g *Game) sceneFlash() float64 {
	a := 0.0
	for _, e := range g.scene.effects {
		switch e.kind {
		case "flash":
			a = math.Max(a, 1-e.t/.5)
		case "halberd":
			if h := e.t - halberdDrop; h >= 0 {
				a = math.Max(a, .9*(1-h/.45))
			}
		case "halberd_throw":
			if h := e.t - halberdThrow(); h >= 0 {
				a = math.Max(a, .9*(1-h/.45))
			}
		}
	}
	return math.Max(0, a)
}
