package gui

import (
	"reflect"
	"testing"

	"srpg/internal/core"
)

func TestSplitCounters(t *testing.T) {
	ev := func(kind, actor, target string) core.Event {
		return core.Event{Kind: kind, Actor: actor, Target: target}
	}
	t.Run("counter after the blow", func(t *testing.T) {
		events := []core.Event{
			ev("critical", "a", "b"), ev("damage", "a", "b"),
			ev("damage", "b", "a"), ev("retreat", "a", ""),
			ev("attack-hit", "a", "b"), ev("experience", "a", ""),
		}
		blow, counters := splitCounters("a", events)
		check(t, blow, []core.Event{events[0], events[1], events[4], events[5]})
		check(t, counters, []counter{{by: "b", events: []core.Event{events[2], events[3]}}})
	})
	t.Run("each target counters separately", func(t *testing.T) {
		events := []core.Event{
			ev("damage", "a", "b"), ev("miss", "b", "a"),
			ev("damage", "a", "c"), ev("critical", "c", "a"), ev("damage", "c", "a"),
		}
		blow, counters := splitCounters("a", events)
		check(t, blow, []core.Event{events[0], events[2]})
		check(t, counters, []counter{{by: "b", events: events[1:2]}, {by: "c", events: events[3:5]}})
	})
	t.Run("no counter", func(t *testing.T) {
		events := []core.Event{ev("damage", "a", "b"), ev("retreat", "b", "")}
		blow, counters := splitCounters("a", events)
		check(t, blow, events)
		check(t, counters, []counter{})
	})
}

func TestSplitDouble(t *testing.T) {
	ev := func(kind string) core.Event { return core.Event{Kind: kind, Actor: "a", Target: "b"} }
	t.Run("second blow", func(t *testing.T) {
		events := []core.Event{ev("damage"), ev("double"), ev("critical"), ev("damage")}
		check(t, splitDouble(events), [][]core.Event{events[:1], events[1:]})
	})
	t.Run("single blow", func(t *testing.T) {
		events := []core.Event{ev("miss")}
		check(t, splitDouble(events), [][]core.Event{events})
	})
}

func check(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
