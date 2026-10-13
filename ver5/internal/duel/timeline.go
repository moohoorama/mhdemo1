// Package duel samples a scripted duel without changing battle state.
package duel

import (
	"math"
	"strings"
)

var Endings = []string{"아군 승리", "아군 패배", "무승부", "아군 도주 성공", "아군 도주 실패", "적 도주 성공", "적 도주 실패"}

type Actor struct {
	Visible                   bool
	Position                  float64
	Lane                      float64
	Facing, Animation, Speech string
	MS                        float64
	Hold                      int     // -1 animates; otherwise a fixed frame
	Fade                      float64 // 0 opaque, 1 invisible
	Gray                      float64 // 0 coloured, 1 grayscale
}

// ScreenPosition projects the duel axis and its perpendicular lane onto the arena.
func (a Actor) ScreenPosition() (float64, float64) {
	return 640 + a.Position*.89 - a.Lane*.45, 440 + a.Position*.45 + a.Lane*.89
}

func (a Actor) facingToward(other Actor) string {
	x, y := a.ScreenPosition()
	tx, ty := other.ScreenPosition()
	angle := math.Mod(math.Atan2(ty-y, tx-x)*180/math.Pi+360, 360)
	return [8]string{"E", "SE", "S", "SW", "W", "NW", "N", "NE"}[int((angle+22.5)/45)%8]
}

type State struct {
	Actors               [4]Actor
	Stage                string
	Index                int
	Local, Flash, Impact float64
	Critical             bool
	Attacker             int
	ShakeSide            int // -1 shakes both; otherwise only this actor
	Done                 bool
}

type Beat struct {
	Kind, Title, Speech string
	Side                int
	Duration            float64
}

type Story struct {
	Beats []Beat
}

const attackReach = 120.0
const chargeReach = 190.0

// rankDepth is where a retreating fighter meets the front rank of its own troops.
const rankDepth, fadeSpan = 500.0, 100.0

func strikeKind(kind string) string {
	return strings.TrimPrefix(kind, "rush-")
}

func defender(side int) int {
	if side == 1 {
		return 0
	}
	return 1
}

func isStrike(kind string) bool {
	switch strikeKind(kind) {
	case "hit", "block", "critical-block", "yielding-block", "dodge", "critical-dodge", "critical", "backhit":
		return true
	}
	return false
}

func initialState() State {
	return State{Actors: [4]Actor{{Visible: true, Position: -260}, {Visible: true, Position: 260}}}
}

// prepareStory connects each exchange to the positions left by the preceding one.
func prepareStory(beats []Beat) Story {
	story := Story{}
	state := initialState()
	for _, beat := range beats {
		if isStrike(beat.Kind) || beat.Kind == "press" {
			a, d := state.Actors[beat.Side], state.Actors[defender(beat.Side)]
			distance := math.Hypot(d.Position-a.Position, d.Lane-a.Lane)
			reach, kind, title := attackReach, "engage", "공격 거리까지 접근"
			if strikeKind(beat.Kind) != beat.Kind {
				reach, kind, title = chargeReach, "charge-ready", "상대를 보며 전진 공격의 간격 확보"
			}
			if distance > reach+1e-6 || kind == "charge-ready" && math.Abs(distance-reach) > 1e-6 {
				step := Beat{Kind: kind, Title: title, Side: beat.Side, Duration: max(.25, math.Abs(distance-reach)/220)}
				story.Beats = append(story.Beats, step)
				apply(&state, step, step.Duration)
			}
		}
		story.Beats = append(story.Beats, beat)
		apply(&state, beat, beat.Duration)
	}
	return story
}

func screenOffset(x, y float64) (float64, float64) {
	return (.89*x + .45*y) / .9946, (-.45*x + .89*y) / .9946
}

func New(ending int) Story {
	ending = max(0, min(6, ending))
	b := []Beat{
		{"talk", "진영에서 마주 서다", "내 칼을 받아 보아라!", 0, 2.2},
		{"talk", "적장의 도발", "말보다 실력을 보여라!", 1, 1.8},
		{"approach", "대열 앞으로 · 접근", "", 0, 1.4},
		{"block", "첫 공방 · 방어", "어림없다!", 0, 1.15},
		{"hit", "반격 · 피격", "", 1, 1.15},
		{"dodge", "한 발 뒤로 · 회피", "너무 느리다!", 1, 1.15},
		{"hit", "연속 공격 · 첫 타", "", 0, 1.05},
		{"block", "연속 공격 · 두 번째", "", 0, 1.05},
		{"hit", "연속 공격 · 세 번째", "받아라!", 0, 1.05},
		{"rest", "숨을 고르다", "헉… 헉…", 1, 1.8},
		{"backstep", "서로를 보며 뒷걸음질", "이번에는 끝을 보자!", 0, 1.6},
		{"talk", "돌진 전의 정적", "간다!", 1, .8},
		{"crossing-charge", "동시 돌진 · 섬광 · 교차", "", 0, 1.6},
		{"turn", "관성으로 지나친 뒤 돌아서다", "", 0, .8},
	}
	switch ending {
	case 0, 1:
		b = append(b, Beat{"close", "다시 맞붙다", "", 0, 1.1}, Beat{"critical", "결정타", "이것으로 끝이다!", ending, 1.3}, Beat{"win", Endings[ending], "내가 이겼다!", ending, 3})
	case 2:
		b = append(b, Beat{"draw", Endings[ending], "오늘은 여기까지다…", 0, 3})
	default:
		runner := 0
		if ending >= 5 {
			runner = 1
		}
		b = append(b, Beat{"close", "진영 쪽으로 물러날 틈", "", 0, 1.1})
		kind := "escape"
		if ending == 4 || ending == 6 {
			kind = "catch"
		}
		b = append(b, Beat{kind, "도주와 추격", "물러난다!", runner, 2.4})
		if kind == "catch" {
			b = append(b, Beat{"backhit", "진영 직전 · 추격자의 일격", "놓칠 줄 아느냐!", 1 - runner, 1.3}, Beat{"win", Endings[ending], "승부는 끝났다!", 1 - runner, 3})
		} else {
			b = append(b, Beat{"escaped", Endings[ending], "다음에 다시 겨루자!", runner, 3})
		}
	}
	return prepareStory(b)
}

func (s Story) Duration() float64 {
	t := 0.0
	for _, b := range s.Beats {
		t += b.Duration
	}
	return t
}

func (s Story) Start(index int) float64 {
	t := 0.0
	for i := 0; i < min(index, len(s.Beats)); i++ {
		t += s.Beats[i].Duration
	}
	return t
}

func (s Story) Sample(t float64) State {
	r := initialState()
	t = max(0, t)
	r.Done = t >= s.Duration()
	elapsed := 0.0
	for i, b := range s.Beats {
		local := max(0, min(t-elapsed, b.Duration))
		apply(&r, b, local)
		r.Stage, r.Index, r.Local = b.Title, i, local
		elapsed += b.Duration
		if t < elapsed {
			return r
		}
	}
	r.Done = true
	return r
}

func apply(s *State, b Beat, t float64) {
	rush := strikeKind(b.Kind) != b.Kind
	b.Kind = strikeKind(b.Kind)
	from := [4]float64{}
	facing := [4]string{}
	lanes := [4]float64{}
	for i, a := range s.Actors {
		from[i], facing[i], lanes[i] = a.Position, a.Facing, a.Lane
	}
	for i := range s.Actors {
		a := &s.Actors[i]
		a.Animation, a.Speech, a.MS, a.Hold = "idle", "", t*1000, -1
		target := 1
		if i == 1 {
			target = 0
		}
		a.Facing = a.facingToward(s.Actors[target])
	}
	s.Flash, s.Impact, s.Critical = 0, 0, false
	s.ShakeSide = -1
	s.Attacker = b.Side
	s.Actors[b.Side].Speech = b.Speech
	if b.Kind == "block" || b.Kind == "dodge" || b.Kind == "critical-dodge" || b.Kind == "yielding-block" {
		s.Actors[b.Side].Speech = ""
		defender := 1
		if b.Side == 1 {
			defender = 0
		}
		s.Actors[defender].Speech = b.Speech
	}
	p := t / b.Duration
	move := func(i int, to, k float64) {
		s.Actors[i].Position = from[i] + (to-from[i])*max(0, min(1, k))
		if from[i] != to && k > 0 && k < 1 {
			s.Actors[i].Animation = "walk"
		}
	}
	switch b.Kind {
	case "engage", "charge-ready":
		a, d := &s.Actors[b.Side], s.Actors[defender(b.Side)]
		delta, laneDelta := d.Position-from[b.Side], d.Lane-lanes[b.Side]
		distance := math.Hypot(delta, laneDelta)
		reach := attackReach
		if b.Kind == "charge-ready" {
			reach = chargeReach
		}
		if distance > 0 && (distance > reach || b.Kind == "charge-ready") {
			travel := (distance - reach) * p
			a.Position += delta / distance * travel
			a.Lane += laneDelta / distance * travel
			if p > 0 && p < 1 {
				a.Animation = "walk"
			}
		}
	case "press":
		direction := 1.0
		if from[b.Side] > from[defender(b.Side)] {
			direction = -1
		}
		for i := range 2 {
			move(i, from[i]+direction*80, p)
		}
	case "reinforcements":
		for i := 2; i < 4; i++ {
			s.Actors[i].Visible = true
			sign := -1.0
			if i == 3 {
				sign = 1
			}
			// Match the map's (u-v)*16, (u+v)*8 isometric grid axis.
			offset, lane := screenOffset(sign*120, -sign*60)
			startOffset, startLane := screenOffset(sign*1200, -sign*600)
			s.Actors[i].Position = from[1] + startOffset + (offset-startOffset)*p
			s.Actors[i].Lane = lanes[1] + startLane + (lane-startLane)*p
			if p > 0 && p < 1 {
				s.Actors[i].Animation = "walk"
			}
			s.Actors[i].Facing = s.Actors[i].facingToward(s.Actors[1])
		}
	case "retreat", "withdrawn":
		sign := -1.0
		if b.Side == 1 {
			sign = 1
		}
		s.Actors[b.Side].Facing = "NW"
		if sign > 0 {
			s.Actors[b.Side].Facing = "SE"
		}
		if b.Kind == "retreat" {
			// Walk into the rearward ranks and vanish among them.
			target := -rankDepth - fadeSpan
			if b.Side == 1 {
				target = rankDepth + fadeSpan
			}
			move(b.Side, target, p)
		}
		s.Actors[b.Side].Fade = max(0, min(1, (math.Abs(s.Actors[b.Side].Position)-rankDepth)/fadeSpan))
	case "withdraw":
		direction := 1.0
		if from[b.Side] < from[1-b.Side] {
			direction = -1
		}
		move(b.Side, from[b.Side]+direction*60, p)
	case "advance":
		direction := 1.0
		if from[b.Side] > from[1-b.Side] {
			direction = -1
		}
		move(b.Side, from[1-b.Side]-direction*120, p)
	case "approach", "close":
		for i := range 2 {
			target := -60.0
			if from[i] > from[1-i] {
				target = 60
			}
			move(i, target, p)
		}
	case "backstep":
		move(0, -270, p)
		move(1, 270, p)
	case "crossing-charge":
		// Freeze at contact for 120 ms, then carry both actors beyond each other.
		q := t / .6
		if t >= .6 {
			q = 1 + max(0, t-.72)/.6
		}
		for i := range 2 {
			if q <= 1 {
				move(i, 0, q)
			} else {
				target := 170.0
				if i == 1 {
					target = -170
				}
				s.Actors[i].Position = target * min(1, q-1)
			}
			s.Actors[i].Animation = "attack"
			s.Actors[i].MS = max(0, t-.14) * 1000
		}
		if t >= .6 && t < .82 {
			s.Flash = 1 - (t-.6)/.22
			s.Impact = s.Flash
		}
	case "turn":
		// The default facing now follows the exchanged positions.
	case "hit", "block", "critical-block", "yielding-block", "dodge", "critical-dodge", "critical", "backhit":
		att, def := b.Side, 1
		if att == 1 {
			def = 0
		}
		a, d := &s.Actors[att], &s.Actors[def]
		a.Facing, d.Facing = a.facingToward(*d), d.facingToward(*a)
		a.Animation = "attack"
		a.MS = max(0, t-.1) * 1000
		if rush {
			delta, laneDelta := d.Position-from[att], d.Lane-lanes[att]
			distance := math.Hypot(delta, laneDelta)
			if distance > attackReach {
				progress := max(0, min(1, (t-.1)/.46))
				travel := (distance - attackReach) * progress
				a.Position += delta / distance * travel
				a.Lane += laneDelta / distance * travel
			}
		}
		direction := 1.0
		if from[att] > from[def] {
			direction = -1
		}
		if b.Kind == "backhit" {
			d.Facing = a.Facing
		}
		if (b.Kind == "block" || b.Kind == "critical-block" || b.Kind == "yielding-block") && t >= .38 {
			d.Animation = "block"
			d.MS = (t - .38) * 1000
		}
		if t >= .56 {
			d.MS = (t - .56) * 1000
			switch b.Kind {
			case "block", "critical-block", "yielding-block":
				d.Animation = "block"
				d.MS = (t - .38) * 1000
				if b.Kind == "yielding-block" {
					d.Position += direction * 55 * min(1, (t-.56)/.35)
				}
			case "dodge", "critical-dodge":
				progress := min(1, (t-.56)/.35)
				d.Position += direction * 64 * progress
				if progress < 1 {
					d.Animation = "walk"
				}
			default:
				d.Animation = "hit"
				push := 20.0
				if b.Kind == "critical" || b.Kind == "backhit" {
					push = 52
				}
				d.Position += direction * push * math.Sin(math.Pi/2*min(1, (t-.56)/.35))
			}
			if b.Kind != "dodge" && b.Kind != "critical-dodge" {
				s.Impact = max(0, 1-(t-.56)/.28)
			}
			if b.Kind == "critical" || b.Kind == "backhit" || b.Kind == "critical-block" || b.Kind == "critical-dodge" {
				s.Critical = true
				s.Flash = s.Impact * .65
				if b.Kind == "critical-dodge" {
					s.Flash = max(0, 1-(t-.56)/.28) * .5
					s.ShakeSide = att
				}
				if b.Kind == "critical-block" {
					s.ShakeSide = att
				}
			}
		}
	case "slash-through":
		// Dash in, flash, and reappear behind the target, which has already been cut.
		att, def := b.Side, defender(b.Side)
		a, d := &s.Actors[att], &s.Actors[def]
		delta, laneDelta := d.Position-from[att], d.Lane-lanes[att]
		distance := math.Hypot(delta, laneDelta)
		ux, uy := delta/distance, laneDelta/distance
		a.Animation, a.MS = "attack", max(0, t-.1)*1000
		d.Facing = d.facingToward(Actor{Position: from[att], Lane: lanes[att]})
		const contact = .5
		if t < contact {
			travel := max(0, distance-attackReach) * max(0, min(1, (t-.1)/(contact-.1)))
			a.Position += ux * travel
			a.Lane += uy * travel
			a.Facing = a.facingToward(*d)
			s.Flash = max(0, (t-.4)/.1)
			break
		}
		a.Position, a.Lane = d.Position+ux*110, d.Lane+uy*110
		a.Facing = a.facingToward(Actor{Position: a.Position + ux*200, Lane: a.Lane + uy*200})
		d.Animation, d.MS = "hit", (t-contact)*1000
		d.Position += ux * 20 * math.Sin(math.Pi/2*min(1, (t-contact)/.35))
		d.Lane += uy * 20 * math.Sin(math.Pi/2*min(1, (t-contact)/.35))
		s.Critical = true
		s.Flash = max(0, 1-(t-contact)/.35)
		s.Impact = max(0, 1-(t-contact)/.28)
	case "rest", "draw":
		for i := range 2 {
			s.Actors[i].Animation = "exhausted"
		}
	case "win":
		for i := range 2 {
			s.Actors[i].Facing = facing[i]
		}
		s.Actors[b.Side].Animation = "attack"
		s.Actors[b.Side].Hold = 0
		s.Actors[1-b.Side].Animation = "exhausted"
		s.Actors[1-b.Side].Hold = 3
		s.Actors[1-b.Side].Gray = min(1, t/.6)
	case "escape", "catch", "straight-chase":
		runner, chaser := b.Side, 1-b.Side
		sign := -1.0
		if runner == 1 {
			sign = 1
		}
		target := sign * 330
		if b.Kind == "catch" || b.Kind == "straight-chase" {
			target = sign * 230
		}
		move(runner, target, min(1, p/.85))
		// Sidestep around the opponent before returning to the home formation.
		if b.Kind != "straight-chase" {
			s.Actors[runner].Lane = 80 * math.Sin(math.Pi*min(1, p/.85))
		}
		chaseTarget := sign * 100
		if b.Kind == "catch" || b.Kind == "straight-chase" {
			chaseTarget = target - sign*100
		}
		move(chaser, chaseTarget, (p-.22)/.63)
		for i := range 2 {
			s.Actors[i].Facing = "NW"
			if sign > 0 {
				s.Actors[i].Facing = "SE"
			}
		}
	case "escaped":
		s.Actors[b.Side].Facing = facing[b.Side]
		s.Actors[b.Side].Animation = "exhausted"
	}
}
