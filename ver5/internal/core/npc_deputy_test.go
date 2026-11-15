package core

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srpg/internal/content"
)

// caoSpawn is 조조's spawn in the B18 event that brings him in.
func caoSpawn(t *testing.T, d *content.Data) (int, content.Spawn) {
	t.Helper()
	i := d.StageIndex("B18")
	require.GreaterOrEqual(t, i, 0)
	for _, ev := range d.Stages[i].Events {
		for _, a := range ev.Do {
			for _, sp := range a.Spawns {
				if sp.Character == "caocao" {
					return i, sp
				}
			}
		}
	}
	t.Fatal("no 조조 spawn in B18")
	return 0, content.Spawn{}
}

func TestCaoCaoAmbitionAndDeputy(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	stage, sp := caoSpawn(t, d)
	require.Equal(t, "xunyou", sp.Deputy)
	require.NoError(t, d.ValidateSpawn(sp))

	e := fromState(d, State{Difficulty: "normal", Stage: stage, Phase: "battle"})
	u := e.spawn(sp.Character, sp.Character, "enemy", sp.X, sp.Y, sp)
	c := e.char(u.Character)

	t.Run("패기 buffs every stat each turn", func(t *testing.T) {
		assert.Contains(t, innate(c), "ambition", "level %d plan", c.Level)
		for id := range content.StatStatuses {
			assert.Equal(t, 20, u.Status[id].Value, id)
		}
	})
	t.Run("순유 rides as deputy and opens attack spells", func(t *testing.T) {
		q := e.char(c.Deputy)
		require.NotNil(t, q)
		assert.True(t, q.Temp)
		assert.Nil(t, e.unit(q.ID), "a deputy has no unit of its own")
		assert.Equal(t, q.Level, c.Level)
		assert.Contains(t, d.ClassLine("strategist"), q.Class)
		skills := e.skills(c)
		for _, s := range []string{"scorch", "torrent", "rockfall"} {
			assert.True(t, slices.Contains(skills, s), "%s in %v", s, skills)
		}
		foe := e.spawn("troop_footman", "troop_footman", "ally", sp.X+1, sp.Y, content.Spawn{})
		u.MP = 100
		_, err := e.validateSkill(u, Command{Kind: "skill", Actor: u.ID, Target: foe.ID, Skill: "scorch"}, d.Skills["scorch"])
		assert.NoError(t, err)
	})
}

func TestNPCDeputySaveRestores(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	_, sp := caoSpawn(t, d)
	sp.X, sp.Y, sp.Side = 12, 2, ""
	d.Stages[0].Enemies = append(d.Stages[0].Enemies, sp)
	e := New(d, 1)
	for n := 0; e.Observe().Phase != "battle"; n++ {
		require.Less(t, n, 100, "battle not reached")
		if e.Deploying() {
			finishDeployment(t, e)
			continue
		}
		if e.Walking() {
			for _, c := range e.ExploreNext() {
				_, err = e.Apply(c)
				require.NoError(t, err)
			}
			continue
		}
		c := Command{Kind: "next"}
		switch o := e.Observe(); o.Dialogue.Kind {
		case "choice":
			for k := range o.Dialogue.Choices {
				c = Command{Kind: "choose", Option: k}
				break
			}
		case "battle":
			c.Kind = "start"
		}
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	require.Equal(t, "xunyou", e.char("caocao").Deputy)
	restored, err := Restore(d, e.Snapshot())
	require.NoError(t, err)
	assert.Equal(t, "xunyou", restored.char("caocao").Deputy)
	assert.Equal(t, e.stats(e.unit("caocao")), restored.stats(restored.unit("caocao")))
}
