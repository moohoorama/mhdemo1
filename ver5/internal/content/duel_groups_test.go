package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuelGroups(t *testing.T) {
	t.Run("separate pairs", func(t *testing.T) {
		s := Stage{Duels: []Duel{{Ally: "zhangfei", Enemy: "dengmao"}, {Ally: "guanyu", Enemy: "chengyuanzhi"}}}
		assert.Equal(t, []int{0, 1}, s.DuelGroups())
	})
	t.Run("one enemy met by several", func(t *testing.T) {
		s := Stage{Duels: []Duel{{Ally: "zhangfei", Enemy: "lubu"}, {Ally: "mushun", Enemy: "lubu"}, {Ally: "guanyu", Enemy: "huaxiong"}}}
		assert.Equal(t, []int{0, 0, 1}, s.DuelGroups())
	})
	t.Run("joined by a support", func(t *testing.T) {
		s := Stage{Duels: []Duel{{Ally: "a", Enemy: "x"}, {Ally: "b", Enemy: "y"}, {Ally: "c", Enemy: "z", Supports: []string{"a", "b"}}}}
		assert.Equal(t, []int{0, 0, 0}, s.DuelGroups())
	})
}

func TestDrawDuelLeavesBothInPlace(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	for i, s := range d.Stages {
		if len(s.Duels) == 0 {
			continue
		}
		c := cloneMap(*d)
		c.Explores = d.Explores
		c.Stages[i].Duels = append([]Duel(nil), s.Duels...)
		c.Stages[i].Duels[0].Outcome, c.Stages[i].Duels[0].AllyResult, c.Stages[i].Duels[0].EnemyResult = "draw", "stay", "stay"
		require.NoError(t, c.Validate())
		c.Stages[i].Duels[0].EnemyResult = "retreat"
		assert.ErrorContains(t, c.Validate(), "draw")
		return
	}
	t.Fatal("no stage has a duel")
}
