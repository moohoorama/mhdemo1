package core_test

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// bowang plays the real B24 from its battle node: the opening duel, the chase, a scripted 조운 who walks into the lure
// area, and the game's AI for the Wei side.
type bowang struct {
	t       *testing.T
	e       *core.Engine
	cells   int // terrain events of the fire (cells lit)
	burnt   int // column units the fire event hit
	hpAfter map[string]int
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

// watch counts the fire's lit cells and the column units it hurts (a damage event without an actor).
func (b *bowang) watch(r core.Result) {
	if b.executed("fire") && b.hpAfter != nil {
		return
	}
	for _, ev := range r.Events {
		switch {
		case ev.Kind == "terrain" && ev.Target == "F":
			b.cells++
		case ev.Kind == "damage" && ev.Actor == "":
			if u := b.unit(ev.Target); u != nil && column(*u) {
				b.burnt++
			}
		}
	}
	if b.executed("fire") {
		b.hpAfter = map[string]int{}
		for _, u := range b.e.Observe().UnitViews {
			if column(u) && hasStatus(u, "confusion") {
				b.hpAfter[u.ID] = u.HP
			}
		}
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

// run lets 조운 walk into the lure area right after the duel and the chase.
func (b *bowang) run(rounds int) {
	for b.e.Observe().Round <= rounds && b.e.Observe().Phase == "battle" && !b.executed("fire") {
		b.walk("zhaoyun", 8, 32, 9)
		b.endTurn()
	}
}

func TestB24FireTrap(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	st := d.Stages[slices.IndexFunc(d.Stages, func(s content.Stage) bool { return s.ID == "B24" })]
	require.Equal(t, []string{"guanyu", "zhangfei"}, st.Reserve)
	require.Equal(t, []string{"alldefeated:enemy"}, st.Victory[0].If)

	t.Run("the duel is a draw and the chase starts at once", func(t *testing.T) {
		b := newBowang(t, 1)
		assert.True(t, b.executed("opening-duel"))
		assert.True(t, b.executed("chase"))
		assert.False(t, b.executed("fire"))
		assert.Nil(t, b.unit("guanyu"), "the ambush is not on the map")
		assert.Nil(t, b.unit("zhangfei"))
		z, x := b.unit("zhaoyun"), b.unit("xiahoudun")
		assert.Positive(t, z.HP)
		assert.Positive(t, x.HP)
		assert.Equal(t, "assault", x.Policy)
		assert.Contains(t, x.Traits, "gale", "절영 gives 하후돈 the speed")
	})

	t.Run("the fire is lit at once when zhaoyun reaches the lure area", func(t *testing.T) {
		for seed := uint64(1); seed <= 3; seed++ {
			b := newBowang(t, seed)
			b.run(8)
			require.True(t, b.executed("fire"), "seed %d", seed)
			assert.Greater(t, b.cells, 15, "a line of woods burns")
			assert.Positive(t, b.burnt, "the column on the road is hurt")
			assert.Positive(t, len(b.hpAfter), "the hurt are confused")
			g, z := b.unit("guanyu"), b.unit("zhangfei")
			require.NotNil(t, g, "the ambush comes out")
			require.NotNil(t, z)
			assert.False(t, g.AI)
		}
	})
}

func hasStatus(u core.UnitView, name string) bool { _, ok := u.Status[name]; return ok }
