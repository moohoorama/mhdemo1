package duelview

import "testing"

func TestBubbleTopNeverCoversFighter(t *testing.T) {
	const h = 90.0
	for _, c := range []struct{ head, foot float64 }{{300, 380}, {120, 200}, {40, 120}, {-20, 60}, {700, 780}} {
		top := bubbleTop(c.head, c.foot, h)
		if top < 12 || top+h > 610 {
			t.Fatalf("bubble %v leaves the screen", c)
		}
		if top < c.foot && top+h > c.head && c.head-h >= 12 {
			t.Fatalf("bubble covers the fighter: %v top=%g", c, top)
		}
		if c.head-h < 12 && top != c.foot+18 {
			t.Fatalf("a bubble that does not fit above must go below the feet: %v top=%g", c, top)
		}
	}
}
