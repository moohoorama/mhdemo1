package gui

import (
	"math"

	"srpg/internal/scenes"
)

// Falls, hops and brawls of scene actors: a fighter struck down reels, sinks to its knees and topples over (a
// commoner sheet lies on its own prone frame instead); a cheering one jumps where it stands; brawling pairs keep
// trading blows in the background of a melee while the beats go on.

const (
	fallReel   = .25 // seconds of the hit motion
	fallKneel  = .55 // seconds on its knees, panting
	fallTopple = .3  // seconds toppling over
	hopTime    = .32 // seconds of one jump
	hopHeight  = 5.0 // canvas px at the top of a jump
)

// fallState is a struck-down actor going to the ground.
type fallState struct {
	t    float64
	sign float64 // which way it topples on screen: +1 to the right
}

func (u *unitVis) startFall() {
	if u.fall != nil {
		return
	}
	sign := 1.0
	if u.dir == "NW" || u.dir == "SW" || u.dir == "W" {
		sign = -1
	}
	u.fall = &fallState{sign: sign}
	u.walk, u.hops = nil, 0
	u.play("hit")
}

// fallen reports whether the actor is down or going down: it plays no more motions.
func (u *unitVis) fallen() bool { return u.fall != nil }

// advanceFall moves a fall on: the reel, the knees, then the topple or the prone frame.
func (u *unitVis) advanceFall(dt float64) {
	f := u.fall
	if f == nil {
		return
	}
	f.t += dt
	switch {
	case f.t < fallReel:
		u.anim = "hit"
	case f.t < fallReel+fallKneel:
		if u.anim != "exhausted" {
			u.anim, u.animT = "exhausted", 0
		}
	default:
		if _, ok := u.art.Animations["prone"]; ok {
			u.pose, u.anim, u.animT = "prone", "prone", 0
		} else {
			u.anim, u.animT = "exhausted", 0
		}
	}
}

// toppled is how far over a falling actor lies, 0 standing to 1 flat, and the angle it is drawn at.
func (u *unitVis) toppled() (float64, float64) {
	f := u.fall
	if f == nil || u.pose == "prone" {
		return 0, 0
	}
	k := (f.t - fallReel - fallKneel) / fallTopple
	if k <= 0 {
		return 0, 0
	}
	k = math.Min(1, k)
	k = k * k // gathering speed as it goes over
	return k, f.sign * k * math.Pi / 2
}

// hopLift is how high a jumping actor is off the ground, in canvas px.
func (u *unitVis) hopLift() float64 {
	if u.hops <= 0 {
		return 0
	}
	k := math.Mod(u.hopT, hopTime) / hopTime
	return hopHeight * 4 * k * (1 - k)
}

func (u *unitVis) advanceHop(dt float64) {
	if u.hops <= 0 {
		return
	}
	u.hopT += dt
	if u.hopT >= hopTime {
		u.hopT -= hopTime
		u.hops--
	}
	if u.hops <= 0 {
		u.hopT = 0
	}
}

// brawl is an actor trading blows with another in the background: a swing now and then, the other blocking or
// reeling as it lands.
type brawl struct {
	at   string
	next float64 // seconds to the next swing
	land float64 // seconds to the swing landing; < 0 none in the air
	n    int
}

const brawlBase = 1.05

// startBrawl sets an actor brawling with another; "-" stops it.
func (g *Game) startBrawl(key, at string) {
	s := g.scene
	if s.brawls == nil {
		s.brawls = map[string]*brawl{}
	}
	if at == "-" {
		delete(s.brawls, key)
		return
	}
	s.brawls[key] = &brawl{at: at, next: chatHash(key+at, 0) * brawlBase, land: -1}
}

func (g *Game) updateBrawls(dt float64) {
	s := g.scene
	for key, b := range s.brawls {
		by, at := s.actors[key], s.actors[b.at]
		if by == nil || at == nil || by.gone || at.gone || by.fallen() || at.fallen() {
			delete(s.brawls, key)
			continue
		}
		busy := by.walk != nil || len(s.paths[key]) > 0
		if b.land >= 0 {
			if b.land -= dt; b.land < 0 && !at.fallen() {
				react := "block"
				if chatHash(key, b.n) < .35 {
					react = "hit"
				}
				if at.walk == nil && len(s.paths[b.at]) == 0 {
					at.play(react)
				}
				at.flash = .15
				g.spawn("block", by, at, .4, spark)
				if chatHash(b.at, b.n) < .3 {
					g.sfxAtCell(react, at.u, at.v)
				}
			}
			continue
		}
		apart := math.Max(math.Abs(by.u-at.u), math.Abs(by.v-at.v)) > 1.5 // one of them has gone on: no blows into the air
		if b.next -= dt; b.next > 0 || busy || apart {
			continue
		}
		b.n++
		by.face(at.u, at.v)
		by.play("attack")
		b.land = impactTime(by, "attack")
		b.next = brawlBase*.8 + chatHash(key, b.n)*brawlBase
	}
}

// meleeStep does a step's falls, hops, brawls, thrown riders and end facings.
func (g *Game) meleeStep(v *unitVis, st scenes.Step) {
	s := g.scene
	if st.Fall {
		v.startFall()
	}
	if st.Hop > 0 && !v.fallen() {
		v.hops, v.hopT = st.Hop, 0
	}
	if st.Brawl != "" {
		g.startBrawl(st.Actor, st.Brawl)
	}
	if d := s.actors[st.Drop]; d != nil { // the rider comes off where the horse is now; the horse runs on
		d.u, d.v, d.dir, d.gone, d.leave, d.arriving = v.u, v.v, v.dir, false, nil, 0
		d.lift = v.lift
		d.startFall()
	}
	if st.End != "" && len(st.Move) == 2 {
		if s.arrive == nil {
			s.arrive = map[string]string{}
		}
		s.arrive[st.Actor] = st.End
	}
}
