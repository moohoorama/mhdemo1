package gui

import "testing"

func TestBarTrailStaysWithinTarget(t *testing.T) {
	var b barState
	b.update(1, 0)
	b.update(.5, .05) // a hit leaves the trail lingering above the fill
	b.update(.8, .05) // healed before the trail shrank
	for range 600 {
		b.update(.8, 1.0/60)
	}
	if b.trail > .8+1e-9 || b.fill != .8 {
		t.Fatalf("trail %v fill %v, want both settled at .8", b.trail, b.fill)
	}
}
