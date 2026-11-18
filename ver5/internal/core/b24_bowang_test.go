package core_test

import (
	"math"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// bowang plays the real B24 from its battle node with a scripted 조운: he walks up to the Wei column, baits
// 하후돈 and runs for 신야, while the other officers wait and the Wei side is driven by the game's AI.
type bowang struct {
	t         *testing.T
	e         *core.Engine
	burnt     int     // column units the fire event hit
	size      int     // column units alive when it fired
	fireRound int     // round the fire was lit, 0 while unlit
	share     float64 // burnt / size
	confused  int     // column units confused right after the fire
	fleeing   int     // column units on the flee policy right after the fire
	ford      float64 // the ford guard's strength right after the fire, 1 = untouched
	fordHP    int
	zhaoMin   int // the lowest strength 조운 reached, as a share of his start
	zhaoHP    int
}

func newBowang(t *testing.T, seed uint64) *bowang {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := core.NewAt(d, seed, "c24-battle", "")
	require.NoError(t, err)
	b := &bowang{t: t, e: e}
	for {
		c, ok := ai.Learning(e)
		if !ok {
			break
		}
		b.apply(c)
	}
	b.apply(core.Command{Kind: "start"})
	plan, err := e.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		b.apply(c)
	}
	require.Equal(t, "battle", e.Observe().Phase)
	for _, u := range e.Observe().UnitViews {
		if u.Group == "ford" {
			b.fordHP += u.HP
		}
		if u.ID == "zhaoyun" {
			b.zhaoHP = u.HP
		}
	}
	b.zhaoMin = b.zhaoHP
	return b
}

func (b *bowang) unit(id string) *core.UnitView {
	us := b.e.Observe().UnitViews
	if i := slices.IndexFunc(us, func(u core.UnitView) bool { return u.ID == id }); i >= 0 {
		return &us[i]
	}
	return nil
}

// column is the marching Wei body: vanguard, center and rear guard, not the supply carts.
func column(u core.UnitView) bool {
	return u.HP > 0 && (u.Group == "van" || u.Group == "main" || u.Group == "rear") && !strings.HasPrefix(u.Character, "loot_cart")
}

func (b *bowang) apply(c core.Command) {
	b.t.Helper()
	r, err := b.e.Apply(c)
	require.NoError(b.t, err, "%+v", c)
	b.watch(r)
}

// watch counts the column units the fire event hits (a damage event without an actor).
func (b *bowang) watch(r core.Result) {
	if z := b.unit("zhaoyun"); z != nil && z.HP < b.zhaoMin {
		b.zhaoMin = z.HP
	}
	if b.fireRound > 0 {
		return
	}
	for _, ev := range r.Events {
		if ev.Kind == "damage" && ev.Actor == "" {
			if u := b.unit(ev.Target); u != nil && column(*u) {
				b.burnt++
			}
		}
	}
	o := b.e.Observe()
	if o.Executed["battle-B24-fire"] {
		b.fireRound = o.Round
		var hp int
		for _, u := range o.UnitViews {
			if column(u) {
				b.size++
				if _, ok := u.Status["confusion"]; ok {
					b.confused++
				}
				if u.Policy == "flee" {
					b.fleeing++
				}
			}
			if u.Group == "ford" {
				hp += u.HP
			}
		}
		b.ford = float64(hp) / float64(b.fordHP)
		b.share = float64(b.burnt) / float64(b.size)
	}
}

func (b *bowang) executed(event string) bool { return b.e.Observe().Executed["battle-B24-"+event] }

// walk moves the unit up to steps cells (Manhattan) toward (x, y), or as far as it can.
func (b *bowang) walk(id string, x, y, steps int) {
	u := b.unit(id)
	if u == nil || u.HP <= 0 {
		return
	}
	dist := func(px, py int) int { return int(math.Abs(float64(px-x)) + math.Abs(float64(py-y))) }
	best, bestD := core.Command{}, dist(u.X, u.Y)
	for _, c := range b.e.LegalActions(id).Commands {
		if c.Kind == "move" && int(math.Abs(float64(c.X-u.X))+math.Abs(float64(c.Y-u.Y))) <= steps && dist(c.X, c.Y) < bestD {
			best, bestD = c, dist(c.X, c.Y)
		}
	}
	if best.Kind == "move" {
		b.apply(best)
	}
}

func (b *bowang) attackIf(id, target string) {
	for _, c := range b.e.LegalActions(id).Commands {
		if c.Kind == "attack" && c.Target == target {
			b.apply(c)
			return
		}
	}
}

// endTurn ends the player's turn and lets the AI play the others until it is the player's again.
func (b *bowang) endTurn() {
	b.apply(core.Command{Kind: "end"})
	for i := 0; i < 400 && b.e.Observe().Phase == "battle" && b.e.Observe().Turn != "ally"; i++ {
		r, err := ai.Step(b.e)
		require.NoError(b.t, err)
		b.watch(r)
	}
}

// run plays rounds: 조운 walks up to the column and baits it, then falls back stepsBack cells a round toward 신야.
func (b *bowang) run(stepsBack, rounds int) {
	for b.e.Observe().Round <= rounds && b.e.Observe().Phase == "battle" && b.fireRound == 0 {
		if b.executed("lure") {
			b.walk("zhaoyun", 8, 40, stepsBack)
		} else {
			b.walk("zhaoyun", 8, 25, 9)
			b.attackIf("zhaoyun", "xiahoudun")
		}
		b.endTurn()
	}
}

func TestB24FireTrap(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	st := d.Stages[slices.IndexFunc(d.Stages, func(s content.Stage) bool { return s.ID == "B24" })]
	require.Equal(t, []string{"guanyu", "zhangfei"}, st.Reserve)

	t.Run("nothing burns and the column stands still until zhaoyun baits it", func(t *testing.T) {
		b := newBowang(t, 1)
		for i := 0; i < 3; i++ {
			b.endTurn()
		}
		assert.False(t, b.executed("lure"))
		assert.Zero(t, b.fireRound)
		assert.Nil(t, b.unit("guanyu"), "the ambush is not on the map")
		assert.Nil(t, b.unit("zhangfei"))
		xhd := b.unit("xiahoudun")
		assert.Equal(t, [2]int{8, 16}, [2]int{xhd.X, xhd.Y})
		assert.Equal(t, "hold", xhd.Policy)
		assert.Contains(t, xhd.Traits, "gale", "절영 gives 하후돈 the speed")
	})

	// The base case: 조운 baits the column and simply runs. Baiting it better, so that more of it stands in the
	// fire, is the player's play; the numbers are logged, not asserted.
	for _, tc := range []struct {
		name string
		back int
	}{{"zhaoyun just runs for 신야", 99}, {"zhaoyun falls back three cells a round", 3}} {
		t.Run(tc.name, func(t *testing.T) {
			var shares []float64
			for seed := uint64(1); seed <= 5; seed++ {
				b := newBowang(t, seed)
				b.run(tc.back, 14)
				require.NotZero(t, b.fireRound, "seed %d: the fire is lit (lure %v)", seed, b.executed("lure"))
				shares = append(shares, b.share)
				t.Logf("seed %d: round %d, %d of %d column units in the fire, 조운 down to %d of %d, safety net %v",
					seed, b.fireRound, b.burnt, b.size, b.zhaoMin, b.zhaoHP, !b.executed("column-in-trap"))

				g, z := b.unit("guanyu"), b.unit("zhangfei")
				require.NotNil(t, g, "the ambush comes out")
				require.NotNil(t, z)
				assert.False(t, g.AI)
				assert.NotEmpty(t, b.e.LegalActions("guanyu").Commands, "the player moves them")
				assert.True(t, g.Y >= 15 && g.Y <= 22, "관우 cuts the middle of the fire stretch (y 9..28): %d", g.Y)
				assert.Less(t, z.Y, g.Y, "장비 comes in behind it, on the rear guard")
				assert.Positive(t, b.burnt, "the fire catches the column")
				assert.Equal(t, 1.0, b.ford, "the ford guard is untouched")
				assert.Equal(t, b.size, b.fleeing, "the whole column is ordered back")
			}
			sort.Float64s(shares)
			t.Logf("%s: share in the fire %v, median %.2f", tc.name, shares, shares[len(shares)/2])
		})
	}

	t.Run("a column nobody baited advances by itself in the end", func(t *testing.T) {
		b := newBowang(t, 1)
		for b.e.Observe().Round < 11 && b.e.Observe().Phase == "battle" {
			b.endTurn()
		}
		assert.Equal(t, "assault", b.unit("xiahoudun").Policy)
	})
}
