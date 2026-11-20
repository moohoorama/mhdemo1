package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// give adds a trait with the effects to the bundle and teaches it to character id.
func give(e *Engine, id, trait string, fx map[string]float64) {
	e.data.Traits[trait] = content.Trait{Name: trait, Kind: "talent", Effects: fx}
	with(e, id, trait)
}

func TestOfficerTraits(t *testing.T) {
	t.Run("패기 is a spell that buffs every stat of the caster", func(t *testing.T) {
		e, hero, _ := arena(t)
		e.state.Learning = nil
		e.data.Traits["ambition"] = content.Trait{Name: "패기", Kind: "strategy", Cost: 1}
		e.data.Skills["ambition"] = content.Skill{Name: "패기", Kind: content.KindBuff, Mode: content.ModeSelf, Shape: content.ShapeSingle,
			Cost: 1, Hit: 100, Duration: 3, Effect: "attack+defense+mind+agility+morale", Requires: []string{"ambition"}}
		with(e, "hero", "ambition")
		hero.MP = 500
		assert.Empty(t, e.startStatus(hero), "nothing as it takes the field")
		apply(t, e, Command{Kind: "skill", Actor: "hero", Target: "hero", Skill: "ambition"})
		for id := range content.StatStatuses {
			assert.Positive(t, e.unit("hero").Status[id].Value, id)
		}
	})

	t.Run("안하무인 halves physical blows from a foe ten 무력 below", func(t *testing.T) {
		e, hero, boss := arena(t)
		mine := e.def(e.char(hero.Character)).Stats[1]
		set := func(n int) {
			d := e.def(e.char(boss.Character))
			d.Stats[1] = n
			e.data.Characters[e.char(boss.Character).Def] = d
		}
		set(mine - 9)
		nine := e.damage(boss, hero, e.basic())
		set(mine - 10)
		ten := e.damage(boss, hero, e.basic())
		give(e, "hero", "arrogance", map[string]float64{content.FxScorn: .5})
		assert.InDelta(t, ten/2, e.damage(boss, hero, e.basic()), 1e-9, "ten below: halved")
		set(mine - 9)
		assert.Equal(t, nine, e.damage(boss, hero, e.basic()), "nine below: as before")
	})

	t.Run("화덕성군 takes no harm from fire spells", func(t *testing.T) {
		e, _, boss := arena(t)
		sage(e, 4, 6)
		e.data.Traits["fire_star"] = content.Trait{Name: "fire_star", Kind: "talent", Effects: map[string]float64{"immune:fire": 1}}
		e.char(boss.Character).TraitsOverride = &[]string{"fire_star"}
		e.state.Learning = nil
		hp := boss.HP
		r := apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
		assert.Equal(t, hp, e.unit("boss").HP)
		for _, ev := range r.Events {
			assert.NotEqual(t, "mark", ev.Kind, "no embers under it")
			assert.NotEqual(t, "critical", ev.Kind)
		}
	})

	t.Run("수영 stands firmer in water and takes half from water spells", func(t *testing.T) {
		e, hero, boss := arena(t)
		sage(e, 4, 6)
		e.markCell(hero.X, hero.Y, "water")
		require.True(t, e.CellProps(hero.X, hero.Y).Wet, "a wet patch")
		before := e.terrain(hero, hero.X, hero.Y).Factor
		torrent := e.data.Skills["flame"]
		torrent.Element = "water"
		hit := e.damage(e.unit("sage"), hero, torrent)
		give(e, "hero", "swim", map[string]float64{content.FxWetFactor: .2, content.FxWard + "water": .5})
		assert.InDelta(t, before+.2, e.terrain(hero, hero.X, hero.Y).Factor, 1e-9)
		assert.InDelta(t, hit/2, e.damage(e.unit("sage"), hero, torrent), 1e-9)
		dry := e.terrain(boss, boss.X, boss.Y).Factor
		give(e, e.char(boss.Character).ID, "swim", map[string]float64{content.FxWetFactor: .2})
		assert.Equal(t, dry, e.terrain(boss, boss.X, boss.Y).Factor, "dry ground: as before")
	})

	t.Run("산악지형보조 adds only on woods and mountains", func(t *testing.T) {
		e, hero, _ := arena(t)
		stage := &e.data.Stages[e.state.Stage]
		woods := content.Point{X: hero.X, Y: hero.Y + 1}
		row := []byte(stage.Tiles[woods.Y])
		row[woods.X*content.CellWidth], row[woods.X*content.CellWidth+1] = 'g', 't' // grass under a forest
		stage.Tiles[woods.Y] = string(row)
		require.True(t, e.CellProps(woods.X, woods.Y).Ambush)
		require.False(t, e.CellProps(hero.X, hero.Y).Ambush)
		plain, forest := e.terrain(hero, hero.X, hero.Y).Factor, e.terrain(hero, woods.X, woods.Y).Factor
		give(e, "hero", "mountain_footing", map[string]float64{content.FxAmbushFactor: .05})
		assert.Equal(t, plain, e.terrain(hero, hero.X, hero.Y).Factor, "open ground: as before")
		assert.InDelta(t, forest+.05, e.terrain(hero, woods.X, woods.Y).Factor, 1e-9)
	})

	t.Run("위진화하 drops every stat round a felled foe", func(t *testing.T) {
		e, _, boss := arena(t)
		give(e, "hero", "awe", map[string]float64{content.FxCrush: 1})
		e.state.Learning = nil
		foe := e.unit("troop#1")
		foe.HP, foe.X, foe.Y = e.stats(foe).MaxHP, boss.X+1, boss.Y
		boss.HP = 1
		for i := 0; e.unit("boss").HP > 0 && i < 20; i++ {
			resetActions(e.unit("hero"))
			e.state.Learning = nil
			apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
		}
		require.Zero(t, e.unit("boss").HP, "the boss falls")
		for id := range content.StatStatuses {
			assert.Equal(t, -20, e.unit("troop#1").Status[id].Value, id)
		}
	})

	t.Run("지형정비 clears the marks round an ally as a spell", func(t *testing.T) {
		e, hero, _ := arena(t)
		sage(e, 4, 6)
		e.state.Learning = nil
		e.data.Traits["terrain_mend"] = content.Trait{Name: "지형정비", Kind: "strategy", Cost: 1}
		e.data.Skills["terrain_mend"] = content.Skill{Name: "지형정비", Kind: content.KindBuff, Mode: content.ModeAny, Shape: content.ShapeSquare,
			Cost: 20, Hit: 100, Effect: "clear_marks", Max: 3, Requires: []string{"terrain_mend"}}
		with(e, "sage", "terrain_mend")
		e.markCell(hero.X+1, hero.Y, "fire")
		require.NotEmpty(t, e.state.TileStates)
		r := apply(t, e, Command{Kind: "skill", Actor: "sage", Target: "hero", Skill: "terrain_mend"})
		assert.Empty(t, e.state.TileStates, "the embers are gone")
		gained := false
		for _, ev := range r.Events {
			gained = gained || ev.Kind == "experience" && ev.Actor == "sage"
		}
		assert.True(t, gained, "clearing teaches")
	})

	t.Run("지형정비 clears the marks of the 3x3", func(t *testing.T) {
		e, hero, _ := arena(t)
		e.markCell(hero.X, hero.Y, "fire")
		e.markCell(hero.X+1, hero.Y+1, "water")
		e.markCell(hero.X+3, hero.Y, "earth")
		marked := len(e.state.TileStates)
		require.Positive(t, marked)
		assert.True(t, e.clearMarks(hero.X, hero.Y))
		for _, s := range e.state.TileStates {
			assert.False(t, abs(s.X-hero.X) <= 1 && abs(s.Y-hero.Y) <= 1, "%+v left in the 3x3", s)
		}
		assert.False(t, e.clearMarks(hero.X, hero.Y), "nothing left to clear")
	})
}

// 조조 gains each planned trait at its specified level.
func TestTraitAtLevels(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	cao := d.Characters["caocao"]
	at := func(level int) []string { return d.PlannedTraits("caocao", cao.Class, level, nil) }
	assert.Contains(t, at(1), "versatile")
	for trait, level := range map[string]int{"civil_martial": 10, "ambition": 15, "feint": 20, "hegemony": 30} {
		t.Run(trait, func(t *testing.T) {
			assert.NotContains(t, at(level-1), trait)
			assert.Contains(t, at(level), trait)
		})
	}
}

// 쉬움 has the player's side deal 120% and take 80%; the others leave blows as they are.
func TestDifficultyDamage(t *testing.T) {
	e, hero, boss := arena(t)
	dealt, taken := e.damage(hero, boss, e.basic()), e.damage(boss, hero, e.basic())
	e.data.Rules.Difficulties = append(e.data.Rules.Difficulties, content.Difficulty{ID: "casual", Name: "쉬움", XPPct: 150, DealtPct: 120, TakenPct: 80})
	e.state.Difficulty = "casual"
	assert.InDelta(t, dealt*1.2, e.damage(hero, boss, e.basic()), 1e-9)
	assert.InDelta(t, taken*.8, e.damage(boss, hero, e.basic()), 1e-9)
}
