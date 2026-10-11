package duel

import (
	"math"
	"testing"
)

func at(s Story, kind string, local float64) State {
	for i, b := range s.Beats {
		if b.Kind == kind {
			return s.Sample(s.Start(i) + local)
		}
	}
	panic("missing beat: " + kind)
}

func TestFacingAndCrossing(t *testing.T) {
	s := New(0)
	t.Run("backstep maintains eye contact", func(t *testing.T) {
		r := at(s, "backstep", .8)
		if r.Actors[0].Facing != "SE" || r.Actors[1].Facing != "NW" || r.Actors[0].Position >= -60 || r.Actors[1].Position <= 60 {
			t.Fatalf("backstep: %+v", r)
		}
	})
	t.Run("flash at shared contact and hit stop", func(t *testing.T) {
		for _, time := range []float64{.601, .70} {
			r := at(s, "crossing-charge", time)
			if r.Flash <= 0 || r.Actors[0].Position != 0 || r.Actors[1].Position != 0 {
				t.Fatalf("contact: %+v", r)
			}
		}
	})
	t.Run("inertia preserves facing until turn", func(t *testing.T) {
		r := at(s, "crossing-charge", 1.4)
		if r.Actors[0].Position <= 0 || r.Actors[1].Position >= 0 || r.Actors[0].Facing != "SE" || r.Actors[1].Facing != "NW" {
			t.Fatalf("cross: %+v", r)
		}
		r = at(s, "turn", .3)
		if r.Actors[0].Facing != "NW" || r.Actors[1].Facing != "SE" {
			t.Fatalf("turn: %+v", r)
		}
	})
}

func TestImpactTiming(t *testing.T) {
	s := New(0)
	for _, kind := range []string{"hit", "block", "critical"} {
		t.Run(kind, func(t *testing.T) {
			before := at(s, kind, .55)
			after := at(s, kind, .57)
			if before.Impact != 0 || after.Impact <= 0 {
				t.Fatalf("impact timing %s: %g / %g", kind, before.Impact, after.Impact)
			}
		})
	}
	t.Run("dodge has no impact and retreats while facing attacker", func(t *testing.T) {
		r := at(s, "dodge", .8)
		if r.Impact != 0 || r.Actors[0].Animation != "walk" || r.Actors[0].Position >= -60 || r.Actors[0].Facing != "SE" {
			t.Fatalf("dodge: %+v", r)
		}
	})
}

func TestEndings(t *testing.T) {
	for ending, name := range Endings {
		t.Run(name, func(t *testing.T) {
			s := New(ending)
			r := s.Sample(s.Duration() + 10)
			if !r.Done || r.Stage != name {
				t.Fatalf("result: %+v", r)
			}
			if ending == 0 || ending == 1 || ending == 4 || ending == 6 {
				winner := 0
				if ending == 1 || ending == 4 {
					winner = 1
				}
				if r.Actors[winner].Animation != "attack" || r.Actors[winner].Hold != 0 || r.Actors[1-winner].Animation != "exhausted" || r.Actors[1-winner].Hold != 3 {
					t.Fatalf("result poses: %+v", r)
				}
			}
			if ending >= 3 {
				runner := 0
				if ending >= 5 {
					runner = 1
				}
				if ending == 3 || ending == 5 {
					if math.Abs(r.Actors[runner].Position) != 330 {
						t.Fatalf("escape: %+v", r)
					}
				} else {
					caught := at(s, "backhit", .57)
					if caught.Actors[0].Facing != caught.Actors[1].Facing || !caught.Critical {
						t.Fatalf("back strike: %+v", caught)
					}
				}
				if runner == 0 && r.Actors[runner].Position >= 0 || runner == 1 && r.Actors[runner].Position <= 0 {
					t.Fatalf("wrong home: %+v", r)
				}
			}
			// Sampling is stable when seeking backwards, with no accumulated motion.
			a := s.Sample(5)
			s.Sample(20)
			if a != s.Sample(5) {
				t.Fatal("sampling depends on playback history")
			}
		})
	}
}

func TestNewDuelMotionsOnEitherSide(t *testing.T) {
	for side := range 2 {
		t.Run([]string{"ally", "enemy"}[side], func(t *testing.T) {
			story := Story{Beats: []Beat{{Kind: "approach", Duration: 1}, {Kind: "critical-block", Side: side, Duration: 1.3}, {Kind: "withdraw", Side: side, Duration: 1}, {Kind: "straight-chase", Side: side, Duration: 2.3}, {Kind: "backhit", Side: 1 - side, Duration: 1.3}, {Kind: "win", Side: 1 - side, Duration: 2}}}
			blocked := at(story, "critical-block", .57)
			if blocked.Actors[1-side].Animation != "block" || blocked.ShakeSide != side || !blocked.Critical {
				t.Fatalf("blocked: %+v", blocked)
			}
			retreat := at(story, "withdraw", .9)
			if retreat.Actors[1-side].Animation != "idle" || retreat.Actors[side].Animation != "walk" || retreat.Actors[side].Facing != []string{"SE", "NW"}[side] {
				t.Fatalf("withdraw: %+v", retreat)
			}
			for _, time := range []float64{.1, .6, 1.5, 2.2} {
				chase := at(story, "straight-chase", time)
				facing := []string{"NW", "SE"}[side]
				for _, a := range chase.Actors[:2] {
					if a.Lane != 0 || a.Facing != facing {
						t.Fatalf("chase: %+v", chase)
					}
				}
			}
		})
	}
}
