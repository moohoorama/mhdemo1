package gui

import (
	"image/color"
	"maps"
	"slices"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestQuickAttack(t *testing.T) {
	g, actor, move := moveGame(t)
	enemy := placeQuickAttackEnemy(t, g, actor, move)
	g.mapClick(actor.X, actor.Y)
	target := g.unitView(g.S.View(), enemy)
	_, ok := g.ord.attackFrom[enemy]
	require.True(t, ok, "enemy is reachable with a basic attack")
	assert.Contains(t, g.orderMarks(g.S.View()), Highlight{target.X, target.Y, color.RGBA{240, 70, 50, 150}})
	g.mapClick(target.X, target.Y)
	require.Equal(t, "target", g.ord.stage)
	assert.Equal(t, "attack", g.ord.pick.Kind)
	assert.NotEmpty(t, g.options(g.ord.pick))
	assert.True(t, g.cancel())
	assert.Equal(t, "menu", g.ord.stage)
	assert.True(t, g.cancel())
	assert.Equal(t, "move", g.ord.stage)
	assert.Equal(t, actor.X, g.unitView(g.view(), actor.ID).X)
	assert.Equal(t, actor.Y, g.unitView(g.view(), actor.ID).Y)
}

func TestQuickAttackMatchesOriginal(t *testing.T) {
	g, actor, move := moveGame(t)
	enemy := placeQuickAttackEnemy(t, g, actor, move)
	base := g.S.Engine
	from, ok := g.quickAttackPositionsOrg(actor)[enemy]
	require.True(t, ok)

	for _, tc := range []struct {
		name   string
		edit   func(*core.Unit, *core.Unit, content.Point)
		ranged bool
	}{
		{name: "move and attack"},
		{name: "already moved", edit: func(actor, _ *core.Unit, _ content.Point) {
			actor.Moved = true
		}},
		{name: "disarmed", edit: func(actor, _ *core.Unit, _ content.Point) {
			actor.Status = map[string]core.Status{"disarm": {Turns: 1}}
		}},
		{name: "rooted", edit: func(actor, _ *core.Unit, _ content.Point) {
			actor.Status = map[string]core.Status{"root": {Turns: 1}}
		}},
		{name: "target leaves after movement", edit: func(_, enemy *core.Unit, _ content.Point) {
			enemy.Policy = "flee"
			enemy.Goal = &content.Point{X: enemy.X, Y: enemy.Y}
			enemy.GoalArea = 0
		}},
		{name: "actor reaches its exit", edit: func(actor, _ *core.Unit, from content.Point) {
			actor.Policy = "flee"
			actor.Goal = &from
			actor.GoalArea = 0
		}},
		{name: "ranged attack", ranged: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := base.Snapshot()
			if tc.edit != nil {
				var attacker, target *core.Unit
				for i := range state.Units {
					u := &state.Units[i]
					switch u.ID {
					case actor.ID:
						attacker = u
					case enemy:
						target = u
					}
				}
				require.NotNil(t, attacker)
				require.NotNil(t, target)
				tc.edit(attacker, target, from)
			}
			if tc.ranged {
				original := g.S.Data.Classes[actor.Class]
				t.Cleanup(func() { g.S.Data.Classes[actor.Class] = original })
				cl := original
				cl.Attack, cl.MinRange, cl.Range = content.AttackRanged, 2, 3
				g.S.Data.Classes[actor.Class] = cl
			}
			var err error
			g.S.Engine, err = core.Restore(g.S.Data, state)
			require.NoError(t, err)
			before := g.S.Engine.Snapshot()
			var moves []content.Point
			for _, c := range g.S.Engine.LegalActions(actor.ID).Commands {
				if c.Kind == "move" {
					moves = append(moves, content.Point{X: c.X, Y: c.Y})
				}
			}
			current := *g.unitView(g.S.View(), actor.ID)
			g.begin(current)
			assert.True(t, slices.Equal(moves, g.ord.moves), "move cells: want %v, got %v", moves, g.ord.moves)
			want := g.quickAttackPositionsOrg(current)
			assert.True(t, maps.Equal(want, g.ord.attackFrom), "attack origins: want %v, got %v", want, g.ord.attackFrom)
			assert.Equal(t, before, g.S.Engine.Snapshot())
		})
	}
}

func (g *Game) quickAttackPositionsOrg(u core.UnitView) map[string]content.Point {
	result := map[string]content.Point{}
	e := g.S.Engine
	candidates := []content.Point{{X: u.X, Y: u.Y}}
	for _, c := range e.LegalActions(u.ID).Commands {
		if c.Kind == "move" {
			candidates = append(candidates, content.Point{X: c.X, Y: c.Y})
		}
	}
	costs := e.MoveCosts(u.ID)
	sort.SliceStable(candidates, func(i, j int) bool {
		return costs[candidates[i]] < costs[candidates[j]]
	})
	for _, p := range candidates {
		var sb *core.Engine
		if p.X == u.X && p.Y == u.Y {
			sb = e
		} else {
			var err error
			sb, err = core.Restore(g.S.Data, e.Snapshot())
			if err != nil {
				continue
			}
			if _, err = sb.Apply(core.Command{Kind: "move", Actor: u.ID, X: p.X, Y: p.Y}); err != nil {
				continue
			}
		}
		for _, c := range sb.LegalActions(u.ID).Commands {
			if c.Kind == "attack" {
				if _, ok := result[c.Target]; !ok {
					result[c.Target] = p
				}
			}
		}
	}
	return result
}

// BenchmarkQuickAttack/original-8    1000    97444215 ns/op    155986445 B/op    228518 allocs/op
// BenchmarkQuickAttack/current-8     1000      259351 ns/op        56712 B/op       272 allocs/op
// Apple M2, Go 1.27.3, 40 move cells. Apply checks save restoration in test binaries.
func BenchmarkQuickAttack(b *testing.B) {
	g, actor, move := moveGame(b)
	placeQuickAttackEnemy(b, g, actor, move)
	moves := 0
	for _, c := range g.S.Engine.LegalActions(actor.ID).Commands {
		if c.Kind == "move" {
			moves++
		}
	}
	for _, tc := range []struct {
		name string
		run  func()
	}{
		{"original", func() { g.quickAttackPositionsOrg(actor) }},
		{"current", func() { g.begin(actor) }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				tc.run()
			}
			b.ReportMetric(float64(moves), "move-cells")
		})
	}
}

func placeQuickAttackEnemy(tb testing.TB, g *Game, actor core.UnitView, move core.Command) string {
	state := g.S.Engine.Snapshot()
	var enemy string
	for i := range state.Units {
		u := &state.Units[i]
		if u.Faction != actor.Faction && u.HP > 0 {
			enemy = u.ID
			u.X, u.Y = move.X, move.Y
			break
		}
	}
	require.NotEmpty(tb, enemy)
	var err error
	g.S.Engine, err = core.Restore(g.S.Data, state)
	require.NoError(tb, err)
	return enemy
}
