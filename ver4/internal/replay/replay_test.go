package replay

import (
	"reflect"
	"testing"

	"srpg/internal/content"
	"srpg/internal/core"
)

func act(actor string, at content.Point, events ...core.Event) *Action {
	return &Action{Actor: actor, Before: map[string]content.Point{actor: at, "t1": {X: 9, Y: 9}, "t2": {X: 9, Y: 0}},
		Steps: []Step{{Events: events}}}
}
func walk(actor string, cells ...content.Point) core.Event {
	return core.Event{Kind: "move", Actor: actor, Path: cells}
}

func TestLink(t *testing.T) {
	t.Run("independent actions start together", func(t *testing.T) {
		a := act("a", content.Point{X: 0, Y: 0}, walk("a", content.Point{X: 0, Y: 0}, content.Point{X: 0, Y: 1}))
		b := act("b", content.Point{X: 5, Y: 0}, walk("b", content.Point{X: 5, Y: 0}, content.Point{X: 5, Y: 1}))
		Link([]*Action{a, b})
		if len(a.After) != 0 || len(b.After) != 0 {
			t.Fatalf("after: %v %v", a.After, b.After)
		}
	})
	t.Run("same target waits for the last claimant only", func(t *testing.T) {
		hit := core.Event{Kind: "damage", Target: "t1", Amount: 3}
		a := act("a", content.Point{X: 0, Y: 0}, hit)
		b := act("b", content.Point{X: 2, Y: 0}, hit)
		c := act("c", content.Point{X: 4, Y: 0}, hit)
		Link([]*Action{a, b, c})
		if !reflect.DeepEqual(b.After, []int{0}) || !reflect.DeepEqual(c.After, []int{1}) {
			t.Fatalf("after: %v %v", b.After, c.After)
		}
	})
	t.Run("crossing paths are ordered", func(t *testing.T) {
		a := act("a", content.Point{X: 0, Y: 1}, walk("a", content.Point{X: 0, Y: 1}, content.Point{X: 1, Y: 1}, content.Point{X: 2, Y: 1}))
		b := act("b", content.Point{X: 1, Y: 0}, walk("b", content.Point{X: 1, Y: 0}, content.Point{X: 1, Y: 1}, content.Point{X: 1, Y: 2}))
		Link([]*Action{a, b})
		if !reflect.DeepEqual(b.After, []int{0}) {
			t.Fatalf("after: %v", b.After)
		}
	})
	t.Run("a unit hit earlier moves after the blow", func(t *testing.T) {
		a := act("a", content.Point{X: 0, Y: 0}, core.Event{Kind: "damage", Actor: "a", Target: "b"})
		a.Before["b"] = content.Point{X: 1, Y: 0}
		b := act("b", content.Point{X: 1, Y: 0}, walk("b", content.Point{X: 1, Y: 0}, content.Point{X: 2, Y: 0}))
		Link([]*Action{a, b})
		if !reflect.DeepEqual(b.After, []int{0}) {
			t.Fatalf("after: %v", b.After)
		}
	})
	t.Run("barrier waits for all and holds back the rest", func(t *testing.T) {
		a := act("a", content.Point{X: 0, Y: 0})
		b := act("b", content.Point{X: 5, Y: 5})
		end := &Action{Barrier: true}
		c := act("c", content.Point{X: 7, Y: 7})
		Link([]*Action{a, b, end, c})
		if !reflect.DeepEqual(end.After, []int{0, 1}) || !reflect.DeepEqual(c.After, []int{2}) {
			t.Fatalf("after: %v %v", end.After, c.After)
		}
	})
}
