package gui

import (
	"math"
	"testing"
)

func TestTimelineBeatsKeepEachMomentAndTheTotal(t *testing.T) {
	var order []string
	mark := func(s string) func() { return func() { order = append(order, s) } }
	var durs []float64
	var fns []func()
	add := func(d float64, f func()) { durs, fns = append(durs, d), append(fns, f) }
	addTimeline(add, []timedFn{{1.0, mark("c")}, {0, mark("a")}, {0, mark("b")}, {2.5, mark("d")}}, 3)
	// play the beats the way the player does: run a beat's function, then wait its length
	at, times := 0.0, map[string]float64{}
	for i := range fns {
		order = nil
		fns[i]()
		if len(order) > 0 {
			times[order[0]] = at
		}
		at += durs[i]
	}
	want := map[string]float64{"a": 0, "b": 0, "c": 1, "d": 2.5}
	for k, v := range want {
		if math.Abs(times[k]-v) > 1e-9 {
			t.Fatalf("%s runs at %v, want %v (%v)", k, times[k], v, times)
		}
	}
	if math.Abs(at-3) > 1e-9 {
		t.Fatalf("the stretch lasts %v, want 3", at)
	}
}
