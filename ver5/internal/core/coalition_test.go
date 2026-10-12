package core

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"testing"
)

func hulaoBattle(t *testing.T) *Engine {
	t.Helper()
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "preparation" && n.Stage == "B03" {
			d.Campaign.Start = n.ID
			break
		}
	}
	e := New(d, 17)
	apply(t, e, Command{Kind: "start"})
	finishDeployment(t, e)
	return e
}
func summonCoalition(t *testing.T, e *Engine) Result {
	t.Helper()
	for _, id := range keys(e.unitIDs) {
		u := e.unit(id)
		if u.Faction == "enemy" && id != "lubu" {
			u.HP = 0
		}
	}
	return apply(t, e, Command{Kind: "wait", Actor: "liubei"})
}
func TestCoalitionArrivalAndControl(t *testing.T) {
	e := hulaoBattle(t)
	boss := e.unit("lubu")
	assert.True(t, boss.Guard)
	moves := e.moves(boss)
	assert.Empty(t, moves)
	e.settle()
	assert.Nil(t, e.unit("fangyue"))
	// An occupied entry moves the reinforcement to a nearby free cell.
	e.unit("guanyu").X, e.unit("guanyu").Y = 25, 7
	r := summonCoalition(t, e)
	aiCount, speechCount := 0, 0
	for _, u := range e.Observe().UnitViews {
		if u.AI {
			aiCount++
			assert.Equal(t, "ally", u.Faction)
			assert.Equal(t, 1, u.Level)
		}
	}
	assert.Equal(t, 9, aiCount)
	for _, v := range r.Events {
		if v.Kind == "speech" {
			speechCount++
		}
		if v.Kind == "unit-arrival" {
			require.NotNil(t, v.Unit)
		}
	}
	assert.Equal(t, 3, speechCount)
	assert.NotEqual(t, content.Point{X: 25, Y: 7}, content.Point{X: e.unit("fangyue").X, Y: e.unit("fangyue").Y})
	require.Empty(t, e.LegalActions("fangyue").Commands)
	assertRejected(t, e, Command{Kind: "wait", Actor: "fangyue"})
	apply(t, e, Command{Kind: "end"})
	assert.Equal(t, "support", e.state.Turn)
	require.NotEmpty(t, e.LegalActions("fangyue").Commands)
	require.Empty(t, e.LegalActions("guanyu").Commands)
	restored, err := Restore(e.data, e.Snapshot())
	require.NoError(t, err)
	assert.Equal(t, encoded(e.Snapshot()), encoded(restored.Snapshot()))
	apply(t, e, Command{Kind: "end"})
	assert.Equal(t, "enemy", e.state.Turn)
	apply(t, e, Command{Kind: "end"})
	assert.Equal(t, "ally", e.state.Turn)
	assert.Equal(t, 2, e.state.Round)
	e.events = nil
	e.settle()
	assert.Empty(t, e.events)
	assert.Equal(t, 20, e.unit("lubu").Status["defense"].Value)
}
func TestCoalitionDuelsAndSimultaneousResponse(t *testing.T) {
	for _, id := range []string{"fangyue", "mushun", "wuanguo"} {
		t.Run(id, func(t *testing.T) {
			e := hulaoBattle(t)
			summonCoalition(t, e)
			apply(t, e, Command{Kind: "end"})
			u := e.unit(id)
			boss := e.unit("lubu")
			u.X, u.Y = boss.X-1, boss.Y
			r := apply(t, e, Command{Kind: "attack", Actor: id, Target: "lubu"})
			assert.Zero(t, e.unit(id).HP)
			assert.Equal(t, id != "wuanguo", e.char(id).Dead)
			assert.Positive(t, e.unit("lubu").HP)
			duels, talks := 0, []Event{}
			for _, v := range r.Events {
				if v.Kind == "duel" {
					duels++
					assert.Equal(t, "defeat", v.Outcome)
				}
				if v.Kind == "speech" {
					talks = append(talks, v)
				}
			}
			assert.Equal(t, 1, duels)
			require.Len(t, talks, 2)
			require.Len(t, talks[1].Speech, 2)
			assert.Equal(t, "zhangfei", talks[1].Speech[0].Actor)
			assert.Equal(t, "guanyu", talks[1].Speech[1].Actor)
			e.events = nil
			e.battleEvents()
			assert.Empty(t, e.events)
			_, err := Restore(e.data, e.Snapshot())
			require.NoError(t, err)
		})
	}
}
func TestEquipmentTraitsWorkForEitherFaction(t *testing.T) {
	for _, side := range []string{"ally", "enemy"} {
		t.Run(side, func(t *testing.T) {
			e := hulaoBattle(t)
			for _, id := range keys(e.unitIDs) {
				e.unit(id).HP = 0
			}
			attacker := e.unit("lubu")
			if side == "ally" {
				attacker = e.unit("guanyu")
				e.char(attacker.Character).Equipment = map[string]string{"weapon": "fangtian_halberd", "armor": "diamond_armor", "accessory": "red_hare"}
			}
			attacker.HP = e.stats(attacker).MaxHP
			attacker.Faction = side
			attacker.X, attacker.Y = 10, 3
			foeSide := "ally"
			if side == "ally" {
				foeSide = "enemy"
			}
			sp := content.Spawn{Character: "troop_footman", Level: 1}
			fallen := e.spawn(sp.Character, sp.Character, foeSide, 11, 3, sp)
			fallen.HP = 0
			other := e.spawn(sp.Character, sp.Character, foeSide, 12, 3, sp)
			other.HP = 1
			friendly := e.spawn(sp.Character, sp.Character, side, 11, 4, sp)
			farther := e.spawn(sp.Character, sp.Character, foeSide, 13, 3, sp)
			hp := friendly.HP
			farHP := farther.HP
			e.data.Rules.HitFloor, e.data.Rules.HitCeil = 100, 100
			e.events = nil
			e.retreatSweep(attacker, fallen, false)
			assert.Zero(t, other.HP)
			assert.Equal(t, hp, friendly.HP)
			assert.Equal(t, farHP, farther.HP)
			sweeps := 0
			for _, v := range e.events {
				if v.Kind == "retreat-sweep" {
					sweeps++
				}
			}
			assert.Equal(t, 1, sweeps)
			assert.Equal(t, e.unitClass(attacker).Move+2, e.movement(attacker))
			// A stronger attacker still provokes a counter without a counter stance.
			e.char(attacker.Character).Level = 1
			enemyDef := e.data.Characters[e.char(farther.Character).Def]
			enemyDef.Stats[1] = 101
			e.data.Characters[e.char(farther.Character).Def] = enemyDef
			farther.X, farther.Y = 10, 4
			e.events = nil
			e.counterBack(farther, attacker)
			counter := false
			for _, v := range e.events {
				counter = counter || v.Actor == attacker.ID && (v.Kind == "damage" || v.Kind == "miss")
			}
			assert.True(t, counter)
			attacker.Status = map[string]Status{}
			e.state.Turn = "ally"
			if side == "ally" {
				e.state.Turn = "enemy"
			}
			e.endFaction()
			assert.Equal(t, 20, attacker.Status["defense"].Value)
		})
	}
}
