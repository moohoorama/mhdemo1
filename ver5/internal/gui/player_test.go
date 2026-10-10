package gui

import (
	"reflect"
	"testing"

	"srpg/internal/core"
	"srpg/internal/sprite"
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

func TestRestAnimationShowsActionAvailability(t *testing.T) {
	art := &sprite.Art{Animations: map[string]sprite.Animation{
		"idle":      {MS: []int{100, 100}, Loop: true},
		"exhausted": {MS: []int{100, 100}, Loop: true},
		"attack":    {MS: []int{100, 100}},
	}}
	for _, tc := range []struct {
		name     string
		anim     string
		inactive bool
		actor    bool
		want     int
	}{
		{"ready", "idle", false, false, 1},
		{"completed", "idle", true, false, 0},
		{"completed exhausted", "exhausted", true, false, 0},
		{"attack still plays", "attack", true, false, 1},
		{"scene actor still plays", "idle", true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := &unitVis{art: art, anim: tc.anim, inactive: tc.inactive, actor: tc.actor, dying: -1, shownHP: 100, maxHP: 100}
			u.update(.15)
			check(t, u.frame(), tc.want)
			if tc.anim == "idle" && tc.inactive && !tc.actor {
				u.inactive = false
				u.update(.15)
				check(t, u.frame(), 1)
			}
		})
	}
}
