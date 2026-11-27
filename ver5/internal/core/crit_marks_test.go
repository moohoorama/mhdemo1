package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// sureSpellCrits makes every spell land and every blow critical.
func sureSpellCrits(e *Engine) {
	e.data.Rules.Crit = content.Chance{Base: 100, Max: 100}
	e.data.Rules.SpellFloor, e.data.Rules.SpellCeil = 100, 100
}

func TestSpellsCritByMoraleLikeBlows(t *testing.T) {
	e, _, boss := arena(t)
	sage(e, 4, 6)
	u, flame := e.unit("sage"), e.data.Skills["flame"]
	want := chance(e.data.Rules.Crit, e.ratio(e.stats(u).Morale, e.stats(boss).Morale))
	require.Positive(t, want)
	assert.Equal(t, want, e.critChance(u, boss, flame))
	p, err := e.Preview(Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	require.NoError(t, err)
	assert.Equal(t, want, p.Crit, "the preview shows a spell's critical chance")
}

func TestACriticalSpellLeavesItsMarkForThreeRounds(t *testing.T) {
	e, _, boss := arena(t)
	sage(e, 4, 6)
	sureSpellCrits(e)
	x, y := boss.X, boss.Y
	r := applyOK(t, e, Command{Kind: "skill", Actor: "sage", Target: "boss", Skill: "flame"})
	assert.Equal(t, byte('e'), e.Cell(x, y).State, "a critical fire spell leaves embers")
	marked := false
	for _, ev := range r.Events {
		marked = marked || ev.Kind == "mark" && ev.Target == "e" && len(ev.Path) == 1 && ev.Path[0] == content.Point{X: x, Y: y}
	}
	assert.True(t, marked, "the replay hears of it")
	round := func() {
		applyOK(t, e, Command{Kind: "end"})
		applyOK(t, e, Command{Kind: "end"})
	}
	round()
	round()
	assert.Equal(t, byte('e'), e.Cell(x, y).State, "two rounds gone")
	round()
	assert.Equal(t, byte('s'), e.Cell(x, y).State, "the embers burn out into soot")
}

func TestMarksChangeTheGroundTheyLieOn(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for family := range d.Terrain {
		for _, ground := range []byte{'g', 'd', 'l', 'u'} {
			bare := d.Move(family, content.Cell{Ground: ground, Structure: content.NoLayer, State: content.NoLayer}, false)
			wet := d.Move(family, content.Cell{Ground: ground, Structure: content.NoLayer, State: 'w'}, false)
			rubble := d.Move(family, content.Cell{Ground: ground, Structure: content.NoLayer, State: 'r'}, false)
			if family == "cavalry" { // horses bog down in the mud; men on foot wade through
				assert.InDelta(t, bare.Factor-.1, wet.Factor, 1e-9, family+": wet ground takes ten points off whatever the ground gives")
				assert.Equal(t, 3, wet.Cost, family+": wet ground costs three steps")
			} else {
				assert.Equal(t, bare.Factor, wet.Factor, family)
				assert.Equal(t, bare.Cost, wet.Cost, family)
			}
			assert.Equal(t, bare.Factor, rubble.Factor, family)
			assert.Equal(t, 3, rubble.Cost, family+": rubble costs three steps")
		}
		forest := content.Cell{Ground: 'g', Structure: 't', State: content.NoLayer}
		under := d.Move(family, forest, false)
		forest.State = 'w'
		if family == "cavalry" {
			assert.InDelta(t, under.Factor-.1, d.Move(family, forest, false).Factor, 1e-9, family+": also under trees")
		}
	}
	for _, ground := range []byte{'g', 'd', 'i', 'l', 'u'} {
		for _, mark := range []byte{'e', 'w', 'r'} {
			assert.Empty(t, d.ValidCell(content.Cell{Ground: ground, Structure: content.NoLayer, State: mark}), "a mark lies on every open ground")
		}
	}
	assert.Equal(t, map[string]string{"fire": "e", "water": "w", "earth": "r"}, d.Rules.CritMarks)
	assert.Equal(t, 3, d.Rules.CritMarkTurns)
}

func TestMarksLieOnlyOnOpenGroundAndLeaveEventStatesAlone(t *testing.T) {
	e := battle(t)
	stage := e.data.Stages[0]
	e.markCell(4, 3, "water")
	assert.Equal(t, byte(content.NoLayer), e.Cell(4, 3).State, "no mark on water")
	e.setState(stage, 6, 6, "F", 0)
	e.markCell(6, 6, "water")
	assert.Equal(t, byte('F'), e.Cell(6, 6).State, "a fire an event lit stays")
	e.setState(stage, 7, 6, "s", 0)
	e.markCell(7, 6, "earth")
	assert.Equal(t, byte('r'), e.Cell(7, 6).State, "soot is covered")
	e.markCell(7, 6, "water")
	assert.Equal(t, byte('w'), e.Cell(7, 6).State, "a mark covers another")
	e.markCell(8, 6, "wind")
	assert.Equal(t, byte(content.NoLayer), e.Cell(8, 6).State, "an element without a mark leaves none")
}

func TestAMarkingArrowLeavesEmbersWhereItHits(t *testing.T) {
	e, a, boss := arena(t)
	s := e.data.Skills["arrow_s"]
	s.Element, s.Mark = "fire", true
	x, y := boss.X, boss.Y
	t.Run("a miss leaves nothing", func(t *testing.T) {
		e.data.Rules.HitFloor, e.data.Rules.HitCeil = 0, 0
		e.attackOn(a, boss, []*Unit{boss}, s, map[string]float64{boss.ID: 1}, ext{}, false, false, false)
		assert.NotEqual(t, byte('e'), e.Cell(x, y).State)
	})
	t.Run("a hit marks without a critical", func(t *testing.T) {
		e.data.Rules.Crit = content.Chance{}
		e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
		boss.HP = 999
		e.attackOn(a, boss, []*Unit{boss}, s, map[string]float64{boss.ID: 1}, ext{}, false, false, false)
		assert.Equal(t, byte('e'), e.Cell(x, y).State)
	})
}
