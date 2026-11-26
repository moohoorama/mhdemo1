package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestPassAilmentsToEnemy(t *testing.T) {
	t.Run("moves harmful statuses and leaves buffs", func(t *testing.T) {
		e, hero, boss := arena(t)
		boss.X, boss.Y = hero.X+1, hero.Y
		hero.Status["poison"] = Status{Turns: 3, Source: boss.ID}
		hero.Status["attack"] = Status{Turns: 2, Value: -20}
		hero.Status["defense"] = Status{Turns: 2, Value: 20}
		e.passAilments(hero)
		assert.NotContains(t, hero.Status, "poison")
		assert.NotContains(t, hero.Status, "attack")
		assert.Contains(t, hero.Status, "defense")
		assert.Equal(t, Status{Turns: 3, Source: boss.ID}, boss.Status["poison"])
		assert.Equal(t, -20, boss.Status["attack"].Value)
	})
	t.Run("immunity keeps poison on source", func(t *testing.T) {
		e, hero, boss := arena(t)
		boss.X, boss.Y = hero.X+1, hero.Y
		with(e, "boss", "calm")
		hero.Status["poison"] = Status{Turns: 3}
		e.passAilments(hero)
		assert.Contains(t, hero.Status, "poison")
		assert.NotContains(t, boss.Status, "poison")
	})
}

func TestPoisonOnHostileSpells(t *testing.T) {
	for _, name := range []string{"flame", "weaken"} {
		t.Run(name, func(t *testing.T) {
			e, _, boss := arena(t)
			sage(e, 4, 6)
			with(e, "sage", "poison_t")
			caster := e.unit("sage")
			spell := e.data.Skills[name]
			spell.Hit, spell.Sure = 100, true
			require.Contains(t, []string{content.KindMagic, content.KindStatus}, spell.Kind)
			e.castSpell(caster, boss, spell, Command{Kind: "skill", Actor: caster.ID, Target: boss.ID, Skill: name}, false)
			assert.Positive(t, boss.Status["poison"].Turns)
			assert.Equal(t, caster.ID, boss.Status["poison"].Source)
		})
	}
}
