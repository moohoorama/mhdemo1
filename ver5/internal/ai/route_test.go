package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

// riverTiles is the mini map with a river down column x, open only at the rows in fords.
func riverTiles(x int, fords ...int) []string {
	rows := make([]string, 8)
	for y := range rows {
		cells := make([]string, 12)
		for i := range cells {
			cells[i] = "g--"
		}
		open := false
		for _, f := range fords {
			open = open || f == y
		}
		if !open {
			cells[x] = "w--"
		}
		rows[y] = strings.Join(cells, "")
	}
	return rows
}

// moveUnit puts a unit at (x, y) and returns the restored engine.
func moveUnit(t *testing.T, e *core.Engine, id string, x, y int) *core.Engine {
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].ID == id {
			snap.Units[i].X, snap.Units[i].Y = x, y
		}
	}
	e, err := core.Restore(e.Data(), snap)
	require.NoError(t, err)
	return e
}

func TestPlanFollowsTheWholeWalk(t *testing.T) {
	t.Run("farthest is the last cell of the way a move reaches", func(t *testing.T) {
		way := []content.Point{{X: 1}, {X: 2}, {X: 3}, {X: 4}}
		options := []core.Command{{Kind: "move", X: 1}, {Kind: "move", X: 3}, {Kind: "move", X: 3, Y: 1}, {Kind: "wait"}}
		p, ok := farthest(way, options)
		assert.True(t, ok)
		assert.Equal(t, content.Point{X: 3}, p)
		_, ok = farthest(way, []core.Command{{Kind: "wait"}})
		assert.False(t, ok, "no move reaches the way")
	})

	t.Run("a march goes around a pass an opponent holds", func(t *testing.T) {
		goal := content.Point{X: 1, Y: 0}
		e := battle(t, func(s *content.Stage) {
			s.Tiles = riverTiles(6, 0, 7)
			s.Enemies[0].X, s.Enemies[0].Y = 9, 1
			s.Enemies[0].Policy, s.Enemies[0].Goal = "march", &goal
		})
		e = moveUnit(t, e, "hero", 6, 0) // the hero stands in the near ford
		for i := 0; i < 2; i++ {
			enemyTurn(t, e)
		}
		b := unit(e, "boss")
		assert.Greater(t, b.Y, 3, "it walks the long way by the far ford instead of waiting at the held one")
		for i := 0; i < 4; i++ {
			enemyTurn(t, e)
		}
		b = unit(e, "boss")
		assert.Less(t, b.X, 6, "and gets across")
	})
}

// A lord whose victory goal is to reach an area walks there, even with a buff to cast and opponents
// a few cells off across a river that neither side can cross.
func TestAWalkerDoesNotStallBuffing(t *testing.T) {
	e := battleIn(t, func(d *content.Data) {
		s := &d.Stages[0]
		s.Tiles = riverTiles(2)
		s.Enemies[0].X, s.Enemies[0].Y = 5, 2
		s.Areas = append(s.Areas, content.Area{ID: 9, Rect: content.MapRect{X: 0, Y: 7, W: 2, H: 1}, Name: "south"})
		s.Victory = []content.Goal{{If: []string{"at:hero:9"}, Text: "reach"}}
		s.TurnLimit = 30 // time enough: the goal pulls only lightly
	})
	snap := e.Snapshot()
	for i := range snap.Characters {
		if snap.Characters[i].ID == "hero" {
			snap.Characters[i].Learned = append(snap.Characters[i].Learned, "mercy") // 보강·팔진: buffs
		}
	}
	for i := range snap.Units {
		if snap.Units[i].ID == "hero" {
			snap.Units[i].MP = 40
		}
	}
	e, err := core.Restore(e.Data(), snap)
	require.NoError(t, err)
	require.Contains(t, unit(e, "hero").Skills, "bolster")
	cast := false
	for i := 0; i < 200 && e.Observe().Phase == "battle"; i++ {
		c, err := Next(e)
		require.NoError(t, err)
		cast = cast || c.Kind == "skill" && c.Actor == "hero"
		if u := unit(e, c.Actor); u != nil && u.Faction == "ally" {
			t.Logf("r%d %s (%d,%d) %+v", e.Observe().Round, c.Actor, u.X, u.Y, c)
		}
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	assert.False(t, cast, "nobody can be struck across the river, so a buff is wasted")
	assert.Equal(t, "victory", e.Observe().Result)
	assert.LessOrEqual(t, e.Observe().Round, 3)
}

// A caster who prefers schemes (곽도) puts its status spells before its fire while it can pay for one.
func TestASchemerCastsStatusSpellsFirst(t *testing.T) {
	cast := func(prefer string) []string {
		e := battleIn(t, func(d *content.Data) {
			boss := d.Characters["boss"]
			boss.Class, boss.Prefer = "mage", prefer
			boss.Stats[2] = 95
			d.Characters["boss"] = boss
			s := &d.Stages[0]
			s.Enemies[0].X, s.Enemies[0].Y, s.Enemies[0].Policy = 4, 2, "assault"
			s.Enemies[0].Traits, s.Enemies[0].Equipment = &[]string{"fire"}, &[]string{}
		})
		var kinds []string
		for round := 0; round < 2; round++ {
			_, err := e.Apply(core.Command{Kind: "end"})
			require.NoError(t, err)
			for i := 0; e.Observe().Turn == "enemy" && i < 100; i++ {
				c, err := Next(e)
				require.NoError(t, err)
				if c.Kind == "skill" && c.Actor == "boss" {
					kinds = append(kinds, e.Data().Skills[c.Skill].Kind)
				}
				_, err = e.Apply(c)
				require.NoError(t, err)
			}
		}
		return kinds
	}
	kinds := cast(content.KindStatus)
	require.NotEmpty(t, kinds)
	assert.NotContains(t, kinds, content.KindMagic)
}
