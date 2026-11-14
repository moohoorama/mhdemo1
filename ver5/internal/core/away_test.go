package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestAwayOfficerComesBack(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	R := d.Rules

	leave := func(level int) *Engine {
		e := New(d, 1)
		o := e.addCharacter("zhangfei", "zhangfei")
		e.joinAt(o, level)
		o.Learned = []string{"assist"}
		o.Equipment["weapon"] = "qinglong_blade"
		e.dismiss("zhangfei", 19)
		require.Nil(t, e.char("zhangfei"))
		return e
	}

	t.Run("with the levels of the battles missed and what was learned", func(t *testing.T) {
		e := leave(R.JoinLevel(19, 0) + 5)
		o := e.welcome("zhangfei", 20)
		assert.Equal(t, R.JoinLevel(19, 0)+5+R.StageLevel(20)-R.StageLevel(19), o.Level)
		assert.Equal(t, []string{"assist"}, o.Learned)
		assert.Zero(t, o.Left)
		assert.Empty(t, e.state.Away)
	})

	t.Run("comes back wearing what they wore", func(t *testing.T) {
		e := leave(30)
		assert.Zero(t, e.state.Warehouse["qinglong_blade"])
		assert.Equal(t, "qinglong_blade", e.welcome("zhangfei", 20).Equipment["weapon"])
	})

	t.Run("a join brings them back quietly", func(t *testing.T) {
		e := leave(30)
		e.once(content.Node{Kind: "join", Character: "zhangfei", Event: "join-zhangfei-back"})
		require.NotNil(t, e.char("zhangfei"))
		assert.Equal(t, "return", e.events[len(e.events)-1].Kind)
	})

	t.Run("never below a newcomer", func(t *testing.T) {
		e := leave(5)
		assert.Equal(t, e.joinLevel(20), e.welcome("zhangfei", 20).Level)
	})

	t.Run("keeps the class they left in", func(t *testing.T) {
		e := leave(20)
		class := e.state.Away[0].Class
		o := e.welcome("zhangfei", 20)
		assert.GreaterOrEqual(t, o.Level, R.PromoteLevels[1])
		assert.Equal(t, class, o.Class)
	})

	t.Run("a save keeps who is away", func(t *testing.T) {
		e := leave(30)
		r, err := Restore(d, e.Snapshot())
		require.NoError(t, err)
		require.Len(t, r.state.Away, 1)
		assert.Equal(t, 19, r.state.Away[0].Left)
	})
}

func TestAwayInSaveIsChecked(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e := New(d, 1)
	s := e.Snapshot()
	s.Away = []Character{copyOf(s.Characters[0])}
	s.Away[0].Left = 3
	_, err = Restore(d, s)
	assert.EqualError(t, err, "CorruptSave", "an officer cannot be in the party and away at once")
}
