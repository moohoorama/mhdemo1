package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

// realRangedData copies the shipped classes and 근접사격 into the mini test bundle.
func realRangedData(t *testing.T, e *Engine) {
	t.Helper()
	real, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, id := range []string{"horse_archer", "horse_crossbow", "archer", "crossbow", "catapult", "light_infantry", "pikeman", "light_cavalry"} {
		e.data.Classes[id] = real.Classes[id]
	}
	e.data.Traits["close_shot"] = real.Traits["close_shot"]
}

func bandOf(e *Engine, id string) (int, int) { return e.AttackRange(id) }

func TestCloseShotDataAndPools(t *testing.T) {
	real, err := content.Load("../../assets/data")
	require.NoError(t, err)
	tr := real.Traits["close_shot"]
	assert.Equal(t, 800, tr.Cost)
	assert.Equal(t, 1.0, tr.Effects[content.FxCloseShot])
	assert.Contains(t, real.Classes["horse_archer"].Pool, "close_shot")
	assert.Contains(t, real.Classes["horse_crossbow"].Pool, "close_shot")
	for _, id := range []string{"archer", "crossbow", "repeating_crossbow", "catapult", "light_infantry", "pikeman"} {
		assert.NotContains(t, real.Classes[id].Pool, "close_shot", id)
	}
	for _, id := range []string{"horse_archer", "horse_crossbow"} {
		cl := real.Classes[id]
		assert.Equal(t, 2, cl.MinRange, id)
		assert.Equal(t, 3, cl.Range, id)
	}
	assert.Equal(t, 1, real.Classes["light_cavalry"].Range)
}

func TestCloseShotBands(t *testing.T) {
	cases := []struct {
		class  string
		trait  string // who holds close_shot: "", "leader", "deputy", "both"
		lo, hi int
	}{
		{"horse_archer", "", 2, 3},
		{"horse_archer", "leader", 1, 3},
		{"horse_archer", "deputy", 1, 3},
		{"horse_archer", "both", 1, 3},
		{"horse_crossbow", "", 2, 3},
		{"crossbow", "", 2, 3},
		{"crossbow", "deputy", 1, 3},
		{"catapult", "", 2, 3},
		{"catapult", "deputy", 1, 3},
		{"archer", "leader", 1, 3},
		{"light_infantry", "", 1, 1},
		{"light_infantry", "deputy", 1, 1},
		{"pikeman", "both", 1, 1},
		{"light_cavalry", "leader", 1, 1},
	}
	for _, c := range cases {
		t.Run(c.class+"/"+c.trait, func(t *testing.T) {
			e, _, _ := arena(t)
			realRangedData(t, e)
			e.char("hero").Class = c.class
			e.char("hero").Equipment = map[string]string{}
			e.char("hero").Deputy = "archer1"
			switch c.trait {
			case "leader":
				with(e, "hero", "close_shot")
			case "deputy":
				with(e, "archer1", "close_shot")
			case "both":
				with(e, "hero", "close_shot")
				with(e, "archer1", "close_shot")
			}
			lo, hi := bandOf(e, "hero")
			assert.Equal(t, [2]int{c.lo, c.hi}, [2]int{lo, hi})
		})
	}
}

func TestCloseShotCountedOnceWhenLeaderAndDeputyBothHaveIt(t *testing.T) {
	e, hero, _ := arena(t)
	realRangedData(t, e)
	e.char("hero").Deputy = "archer1"
	with(e, "hero", "close_shot")
	with(e, "archer1", "close_shot")
	assert.Equal(t, 1.0, e.fx(hero, content.FxCloseShot))
}

func TestCloseShotDeputyLossRemovesIt(t *testing.T) {
	setup := func(t *testing.T) *Engine {
		e, _, _ := arena(t)
		realRangedData(t, e)
		e.char("hero").Class = "catapult"
		e.char("hero").Equipment = map[string]string{}
		e.char("hero").Deputy = "archer1"
		with(e, "archer1", "close_shot")
		lo, _ := bandOf(e, "hero")
		require.Equal(t, 1, lo)
		return e
	}
	t.Run("deputy dies", func(t *testing.T) {
		e := setup(t)
		e.char("archer1").Dead = true
		lo, hi := bandOf(e, "hero")
		assert.Equal(t, [2]int{2, 3}, [2]int{lo, hi})
	})
	t.Run("deputy dismissed", func(t *testing.T) {
		e := setup(t)
		e.char("hero").Deputy = ""
		lo, hi := bandOf(e, "hero")
		assert.Equal(t, [2]int{2, 3}, [2]int{lo, hi})
	})
}

func TestCloseShotReachesFromAndRealResolution(t *testing.T) {
	setup := func(t *testing.T, class string, withTrait bool) (*Engine, Command) {
		e, _, boss := arena(t)
		realRangedData(t, e)
		e.char("hero").Class = class
		e.char("hero").Equipment = map[string]string{}
		e.char("hero").Deputy = "archer1"
		if withTrait {
			with(e, "archer1", "close_shot")
		}
		boss.HP = 9999
		return e, Command{Kind: "attack", Actor: "hero", Target: "boss"}
	}
	for _, class := range []string{"horse_archer", "catapult", "crossbow"} {
		t.Run(class, func(t *testing.T) {
			e, c := setup(t, class, false)
			x, y := e.unit("boss").X, e.unit("boss").Y // hero is one cell away
			assert.False(t, e.CanStrikeFrom("hero", content.Point{X: x - 1, Y: y}, x, y), "adjacent without the trait")
			assertRejected(t, e, c)
			for d, want := range map[int]bool{2: true, 3: true, 4: false} {
				assert.Equal(t, want, e.CanStrikeFrom("hero", content.Point{X: x - d, Y: y}, x, y), "distance %d", d)
			}

			e, c = setup(t, class, true)
			assert.True(t, e.CanStrikeFrom("hero", content.Point{X: x - 1, Y: y}, x, y))
			assert.False(t, e.CanStrikeFrom("hero", content.Point{X: x - 4, Y: y}, x, y), "max range unchanged")
			hero, boss := e.unit("hero"), e.unit("boss")
			assert.True(t, e.rangedAttack(hero, boss, e.basic()), "adjacent shot stays ranged")
			hp := boss.HP
			r := apply(t, e, c)
			assert.Less(t, e.unit("boss").HP, hp)
			assert.NotEmpty(t, r.Events)
		})
	}
	t.Run("light infantry never shoots", func(t *testing.T) {
		e, _ := setup(t, "light_infantry", true)
		x, y := e.unit("boss").X, e.unit("boss").Y
		assert.False(t, e.CanStrikeFrom("hero", content.Point{X: x - 2, Y: y}, x, y))
		assert.False(t, e.rangedAttack(e.unit("hero"), e.unit("boss"), e.basic()))
	})
}

func TestCloseShotCounterRulesStayRanged(t *testing.T) {
	e, _, boss := arena(t)
	realRangedData(t, e)
	e.char("hero").Class = "horse_archer"
	e.char("hero").Equipment = map[string]string{}
	e.char("hero").Deputy = "archer1"
	with(e, "archer1", "close_shot")
	boss.HP = 9999
	e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
	r := apply(t, e, Command{Kind: "attack", Actor: "hero", Target: "boss"})
	assert.True(t, answered(r, "boss", "hero"), "the adjacent boss strikes back as usual")
	assert.False(t, e.rangedAttack(e.unit("boss"), e.unit("hero"), e.basic()))
}

func TestCloseShotAILegalActions(t *testing.T) {
	setup := func(t *testing.T, trait bool) *Engine {
		e, _, _ := arena(t)
		realRangedData(t, e)
		e.char("hero").Class = "catapult"
		e.char("hero").Equipment = map[string]string{}
		e.char("hero").Deputy = "archer1"
		if trait {
			with(e, "archer1", "close_shot")
		}
		return e
	}
	has := func(e *Engine, from content.Point) bool {
		for _, c := range e.LegalActions("hero").Commands {
			if c.Kind == "attack" && c.Target == "boss" {
				return true
			}
		}
		return false
	}
	assert.False(t, has(setup(t, false), content.Point{}), "adjacent attack is not legal without the trait")
	assert.True(t, has(setup(t, true), content.Point{}), "adjacent attack is legal with the deputy's trait")
}
