package gui

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"srpg/internal/scenes"
)

// sceneStrike is one blow of a Strike step: the striker's motion, the target's answer as it lands, the sparks,
// the flash and the shake, and the target driven back along the blow while it still braces or reels.
type sceneStrike struct {
	by, at  string
	motion  string
	react   string
	push    float64
	crit    bool
	t       float64
	charge  float64 // seconds the striker gathers itself, whitening, before the motion
	blow    float64 // seconds from the step's start to the blow landing
	started bool
	landed  bool
	riding  bool    // the striker is on the move: the motion plays over the ride
	through float64 // cells the striker comes out beyond the target as the blow lands (a cut dealt dashing through)
	kill    bool    // the target falls as the blow lands
	puff    float64 // seconds to the next dust puff while the target slides
	tempo   float64 // the motions play this many times their speed (Step.Tempo); 1 when 0
}

// slideTime is how long a target driven back n cells slides, digging in: longer for a heavier push.
func slideTime(n float64) float64 { return .2 + .12*n }

// strikeLength is how long a Strike step holds its group: the gathering, the motion up to the blow, and the push.
func strikeLength(k *sceneStrike) float64 { return k.blow + slideTime(k.push) + .1 }

// dustPuff is a puff of dust kicked up where a pushed target's feet scrape the ground.
type dustPuff struct{ u, v, t float64 }

// startStrike begins a Strike step: the two turn to each other and the striker gathers itself (Gather) or swings.
func (g *Game) startStrike(st scenes.Step) {
	s := g.scene
	by, at := s.actors[st.Actor], s.actors[st.Strike]
	if by == nil || at == nil {
		return
	}
	k := &sceneStrike{by: st.Actor, at: st.Strike, motion: st.Motion, react: st.React, push: st.Push, crit: st.Crit, charge: st.Gather,
		through: st.Through, kill: st.Kill, tempo: max(1, st.Tempo)}
	if motion(by.art, k.motion) == "" {
		k.motion = "attack"
	}
	if k.react == "" {
		k.react = "block"
	}
	k.blow = k.charge + impactTime(by, k.motion)/k.tempo
	k.riding = by.walk != nil || len(s.paths[st.Actor]) > 0
	s.swing[st.Actor] = k.motion // a step taken during the blow does not cut it short
	if !k.riding {               // a blow dealt in passing keeps facing the way it rides
		by.face(at.u, at.v)
	}
	s.held[st.Actor] = true
	if st.Push > 0 {
		s.held[st.Strike] = true
	}
	s.strikes = append(s.strikes, k)
	s.wait = math.Max(s.wait, strikeLength(k))
	g.updateStrike(k, 0)
}

// impactTime is when a motion's blow lands: the end of its first half of frames (the gathering ones).
func impactTime(v *unitVis, anim string) float64 {
	a, ok := v.art.Animations[anim]
	if !ok {
		return 0
	}
	t := 0.0
	for _, ms := range a.MS[:len(a.MS)/2] {
		t += float64(ms) / 1000
	}
	return t
}

func (g *Game) updateStrikes(dt float64) {
	s := g.scene
	kept := s.strikes[:0]
	for _, k := range s.strikes {
		g.updateStrike(k, dt)
		if k.t < strikeLength(k) {
			kept = append(kept, k)
		}
	}
	s.strikes = kept
	dust := g.assets.Projectiles["dust"]
	life := 0.0
	for _, ms := range dust.MS {
		life += float64(ms) / 1000
	}
	puffs := s.dust[:0]
	for _, p := range s.dust {
		if p.t += dt; p.t < life {
			puffs = append(puffs, p)
		}
	}
	s.dust = puffs
	waiting := map[string]bool{} // strikers still gathering themselves: their motion has not begun
	for _, k := range s.strikes {
		if !k.started {
			waiting[k.by] = true
		}
	}
	for key, m := range s.swing { // the blow is over: a rider still on its way rides on
		if waiting[key] {
			continue
		}
		if v := s.actors[key]; v == nil || v.anim != m {
			delete(s.swing, key)
			if v != nil && v.walk != nil && v.anim != "walk" {
				v.play("walk")
			}
		}
	}
}

func (g *Game) updateStrike(k *sceneStrike, dt float64) {
	s := g.scene
	by, at := s.actors[k.by], s.actors[k.at]
	if by == nil || at == nil {
		return
	}
	k.t += dt
	if !k.started {
		if k.charge > 0 {
			by.charging = true
		}
		if k.t >= k.charge {
			k.started = true
			by.charging = false
			by.playQuick(k.motion, k.tempo)
			cue := "swing"
			if k.crit {
				cue = "swing-crit"
			}
			g.sfxAtCell(cue, by.u, by.v)
		}
	}
	if k.landed && at.walk != nil && k.push > 0 { // the target's feet scrape up dust as it slides
		if k.puff -= dt; k.puff <= 0 {
			k.puff = .05
			s.dust = append(s.dust, dustPuff{u: at.u, v: at.v})
		}
	}
	if k.landed || k.t < k.blow {
		return
	}
	k.landed = true
	if k.push > 0 { // a target driven back braces facing the blow
		at.face(by.u, by.v)
	}
	if motion(at.art, k.react) != "" && !at.fallen() {
		at.playQuick(motion(at.art, k.react), k.tempo)
	}
	if k.kill {
		at.startFall()
	}
	at.flash = .25
	at.shakeSign, at.shakeT, at.shakeAmp = 1, shakeTime, 1
	cue, flash, shake := "block", "pop", .2
	if k.react == "hit" {
		cue = "hit"
	}
	if k.crit {
		cue, flash, shake = "crit", "strobe", .45
		g.spawn("critical", by, at, .7, hot)
	} else {
		g.spawn("block", by, at, .5, spark)
	}
	g.sfxAtCell(cue, at.u, at.v)
	if !strings.Contains(k.by, "#") { // an extra's blow (a rider in the melee round the duel) neither flashes nor shakes the screen
		s.effects = append(s.effects, sceneEffect{kind: flash, u: at.u, v: at.v, hit: -1})
		s.shake = math.Max(s.shake, shake)
	}
	du, dv := at.u-by.u, at.v-by.v
	if w := by.walk; w != nil && k.riding { // a blow dealt riding drives the target on along the ride
		du, dv = w.tu-w.fu, w.tv-w.fv
	}
	d := math.Hypot(du, dv)
	if d == 0 {
		return
	}
	du, dv = du/d, dv/d
	if k.through > 0 { // the striker is through and out the other side before the flash clears
		by.u, by.v = math.Round(at.u+du*k.through), math.Round(at.v+dv*k.through)
		by.walk, s.paths[k.by] = nil, nil
		by.lift = g.field.Lift(int(by.u), int(by.v))
		by.face(by.u+du, by.v+dv)
	}
	if k.push > 0 {
		tu, tv := math.Round(at.u+du*k.push), math.Round(at.v+dv*k.push) // people stand on whole cells
		at.walk = &walkState{fu: at.u, fv: at.v, tu: tu, tv: tv, fl: at.lift, tl: g.field.Lift(int(math.Round(tu)), int(math.Round(tv))),
			dur: slideTime(k.push), ease: true}
	}
}

// drawDust draws the dust puffs kicked up by targets driven back.
func (g *Game) drawDust(dst *ebiten.Image) {
	sh := g.assets.Projectiles["dust"]
	for _, p := range g.scene.dust {
		t, i := p.t*1000, 0
		for i < len(sh.MS)-1 && t >= float64(sh.MS[i]) {
			t -= float64(sh.MS[i])
			i++
		}
		if i >= len(sh.Frames) {
			continue
		}
		a := sh.anchor(i)
		x, y := g.field.Center(p.u, p.v)
		g.drawCanvasSprite(dst, Sprite{Image: sh.Frames[i], PivotX: a[0], PivotY: a[1]}, x, y-g.field.Lift(int(math.Round(p.u)), int(math.Round(p.v))), .85)
	}
}
