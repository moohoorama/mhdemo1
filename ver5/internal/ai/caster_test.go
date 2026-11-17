package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

// casterBattle puts the boss, a caster of the mage class with fire and mercy (which also open a cure, cleanse), at
// (x, y) beside a troop worn to hp percent of its strength, and opens the enemy turn.
func casterBattle(t *testing.T, prefer string, x, y, hp int, status string) *core.Engine {
	e := battleIn(t, func(d *content.Data) {
		boss := d.Characters["boss"]
		boss.Class = "mage"
		boss.Stats[1], boss.Stats[2] = 40, 95
		d.Characters["boss"] = boss
		mage := d.Classes["mage"]
		mage.Prefer = prefer
		d.Classes["mage"] = mage
		d.Skills["cleanse"] = content.Skill{Name: "Cleanse", Kind: content.KindHeal, Mode: "any", Shape: "single", Hit: 100,
			Effect: "cure", Cost: 20, Max: 3, Requires: []string{"mercy"}}
		s := &d.Stages[0]
		s.Enemies[0].X, s.Enemies[0].Y, s.Enemies[0].Policy = x, y, "assault"
		s.Enemies[0].Traits, s.Enemies[0].Equipment = &[]string{"fire", "mercy"}, &[]string{}
		s.Enemies[1].X, s.Enemies[1].Y = x+1, y
	})
	_, err := e.Apply(core.Command{Kind: "end"})
	require.NoError(t, err)
	snap := e.Snapshot()
	for i := range snap.Units {
		if u := &snap.Units[i]; u.ID == "boss" {
			u.MP = unit(e, "boss").Stats.MaxMP
		}
		if u := &snap.Units[i]; u.ID != "boss" && u.Faction == "enemy" && u.X == x+1 && u.Y == y {
			u.HP = unit(e, u.ID).Stats.MaxHP * hp / 100
			if status != "" {
				u.Status = map[string]core.Status{status: {Turns: 2}}
			}
		}
	}
	e, err = core.Restore(e.Data(), snap)
	require.NoError(t, err)
	return e
}

// turn lets the boss act until its turn is over and returns the skills it used.
func turn(t *testing.T, e *core.Engine) []string {
	var used []string
	for i := 0; i < 5 && !unit(e, "boss").Done; i++ {
		c, err := Choose(e, "boss")
		require.NoError(t, err)
		if c.Kind == "skill" {
			used = append(used, c.Skill)
		}
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	return used
}

func TestCasterClassPreferences(t *testing.T) {
	t.Run("a staff officer burns the hero in reach rather than heal a scratched troop", func(t *testing.T) {
		e := casterBattle(t, content.KindMagic, 4, 2, 60, "")
		assert.Equal(t, []string{"flame"}, turn(t, e))
	})

	t.Run("a staff officer walks to where its fire reaches rather than heal", func(t *testing.T) {
		e := casterBattle(t, content.KindMagic, 7, 2, 40, "")
		assert.Equal(t, []string{"flame"}, turn(t, e), "even a troop under half waits while a spell reaches someone")
	})

	t.Run("a healer mends an ally a quarter down", func(t *testing.T) {
		e := casterBattle(t, content.KindHeal, 9, 2, 60, "")
		assert.Equal(t, []string{"mend"}, turn(t, e))
	})

	t.Run("nobody heals an ally barely touched", func(t *testing.T) {
		e := casterBattle(t, content.KindHeal, 9, 2, 90, "")
		assert.NotContains(t, turn(t, e), "mend")
	})

	t.Run("a healer cures a confused ally", func(t *testing.T) {
		e := casterBattle(t, content.KindHeal, 9, 2, 100, "confusion")
		assert.Equal(t, []string{"cleanse"}, turn(t, e))
	})

	t.Run("no cure without an ailment", func(t *testing.T) {
		e := casterBattle(t, content.KindHeal, 9, 2, 100, "")
		assert.NotContains(t, turn(t, e), "cleanse")
	})
}

func TestEqualSpellsTakeTurns(t *testing.T) {
	// cast is the spell the boss opens with in a battle of the given seed, with a second fire spell as strong as the first
	cast := func(seed uint64) string {
		e := casterBattle(t, content.KindMagic, 4, 2, 60, "")
		d := e.Data()
		twin := d.Skills["flame"]
		twin.Name = "Twin"
		d.Skills["twin"] = twin
		snap := e.Snapshot()
		snap.Seed = seed
		e, err := core.Restore(d, snap)
		require.NoError(t, err)
		c, err := Choose(e, "boss")
		require.NoError(t, err)
		return c.Skill
	}

	t.Run("the same battle and round pick the same", func(t *testing.T) {
		for seed := uint64(1); seed <= 5; seed++ {
			assert.Equal(t, cast(seed), cast(seed))
		}
	})

	t.Run("other battles pick either", func(t *testing.T) {
		seen := map[string]bool{}
		for seed := uint64(1); seed <= 20; seed++ {
			seen[cast(seed)] = true
		}
		assert.Equal(t, map[string]bool{"flame": true, "twin": true}, seen)
	})
}
