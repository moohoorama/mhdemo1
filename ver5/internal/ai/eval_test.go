package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestStrikeCacheMatchesEngineWithinOneEvaluation(t *testing.T) {
	e := battleIn(t, func(d *content.Data) {})
	check := func(t *testing.T, e *core.Engine) {
		x := &evaluator{e: e}
		units := e.Observe().UnitViews
		for repeat := range 3 {
			for _, a := range units {
				for _, b := range units {
					assert.Equal(t, e.Strike(a.ID, b.ID), x.strike(a.ID, b.ID), "repeat %d, %s -> %s", repeat, a.ID, b.ID)
				}
			}
			assert.Zero(t, x.strike("missing", "hero"))
		}
	}
	t.Run("repeated pairs", func(t *testing.T) { check(t, e) })
	t.Run("changed state starts a fresh evaluation", func(t *testing.T) {
		snap := e.Snapshot()
		for i := range snap.Characters {
			if snap.Characters[i].ID == "hero" {
				snap.Characters[i].Level += 10
			}
		}
		other, err := core.Restore(e.Data(), snap)
		require.NoError(t, err)
		assert.NotEqual(t, e.Strike("hero", "boss"), other.Strike("hero", "boss"))
		check(t, other)
	})
}

// TestPlayerFollowsAChangedGoal: at the Bian River the player's side first fights Xu Rong's rear guard,
// and once Lu Bu comes and the goal turns to the east end, Liu Bei gets there.
func TestPlayerFollowsAChangedGoal(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "battle" && n.Stage == "B07" {
			d.Campaign.Start = n.ID
		}
	}
	for id, c := range d.Characters {
		if c.Playable {
			c.Level = d.Rules.StageLevel(7)
			d.Characters[id] = c
		}
	}
	e := core.New(d, 1)
	_, err = e.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	struck := false
	for i := 0; i < 5000 && e.Observe().Phase != "result"; i++ {
		c, err := Next(e)
		require.NoError(t, err)
		struck = struck || c.Kind == "attack" && e.Observe().Round < 3
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	assert.True(t, struck, "the rear guard is fought first")
	assert.Equal(t, "victory", e.Observe().Result)
	assert.LessOrEqual(t, e.Observe().Round, 8)
}

// An ally healer ends a fighting ally's confusion before it bolsters anyone.
func TestPlayerCuresAConfusedAlly(t *testing.T) {
	e := battleIn(t, func(d *content.Data) {
		d.Skills["cure"] = content.Skill{Name: "Cure", Kind: content.KindHeal, Mode: content.ModeAny, Shape: content.ShapeSingle, Hit: 100, Effect: "cure", Cost: 10, Max: 3, Requires: []string{"mercy"}}
		d.Stages[0].Enemies[0].X, d.Stages[0].Enemies[0].Y = 3, 3
	})
	snap := e.Snapshot()
	for i := range snap.Characters {
		if snap.Characters[i].ID == "hero" {
			snap.Characters[i].Learned = append(snap.Characters[i].Learned, "mercy")
		}
	}
	for i := range snap.Units {
		switch snap.Units[i].ID {
		case "hero":
			snap.Units[i].MP = 40
		case "archer1":
			snap.Units[i].Status = map[string]core.Status{"confusion": {Turns: 2}}
		}
	}
	e, err := core.Restore(e.Data(), snap)
	require.NoError(t, err)
	require.Contains(t, unit(e, "hero").Skills, "cure")
	require.Contains(t, unit(e, "hero").Skills, "bolster")

	t.Run("cures the confused ally", func(t *testing.T) {
		c, err := Choose(e, "hero")
		require.NoError(t, err)
		assert.Equal(t, "cure", c.Skill)
		assert.Equal(t, "archer1", c.Target)
	})
	t.Run("nothing to cure without a status", func(t *testing.T) {
		snap := e.Snapshot()
		for i := range snap.Units {
			if snap.Units[i].ID == "archer1" {
				snap.Units[i].Status = map[string]core.Status{}
			}
		}
		e, err := core.Restore(e.Data(), snap)
		require.NoError(t, err)
		c, err := Choose(e, "hero")
		require.NoError(t, err)
		assert.NotEqual(t, "cure", c.Skill)
	})
}
