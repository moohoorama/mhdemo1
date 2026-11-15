package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestGrantPutsThePieceOnTheNamedOfficer(t *testing.T) {
	load := func(t *testing.T, nodes ...content.Node) *Engine {
		d, err := content.Load("../../assets/data")
		require.NoError(t, err)
		d.Nodes = append(d.Nodes, nodes...)
		d.Campaign.Start = nodes[0].ID
		require.NoError(t, d.Validate())
		return New(d, 1)
	}
	grant := content.Node{ID: "test-grant", Kind: "grant", Item: "red_hare", Count: 1, Character: "jianyong", Event: "grant-test", Next: "c07-battle"}

	t.Run("an officer away wears it and his old piece goes to the warehouse", func(t *testing.T) {
		e := load(t,
			content.Node{ID: "test-join", Kind: "join", Character: "jianyong", Event: "join-test", Next: "test-say"},
			content.Node{ID: "test-say", Kind: "say", Speaker: "jianyong", Text: "잘 있게.", Next: "test-leave"},
			content.Node{ID: "test-leave", Kind: "leave", Character: "jianyong", Event: "leave-test", Next: "test-grant"},
			grant)
		e.char("jianyong").Equipment["accessory"] = "medicine_pouch"
		before := e.state.Warehouse["medicine_pouch"]
		apply(t, e, Command{Kind: "next"})

		require.Len(t, e.state.Away, 1)
		assert.Equal(t, "red_hare", e.state.Away[0].Equipment["accessory"])
		assert.Zero(t, e.state.Warehouse["red_hare"])
		assert.Equal(t, before+1, e.state.Warehouse["medicine_pouch"])

		o := e.welcome("jianyong", e.stageOrder())
		assert.Equal(t, "red_hare", o.Equipment["accessory"], "he comes back riding it")
	})

	t.Run("an officer in the party wears it at once", func(t *testing.T) {
		g := grant
		g.ID, g.Character = "test-grant2", "liubei"
		e := load(t, g)
		assert.Equal(t, "red_hare", e.char("liubei").Equipment["accessory"])
		assert.Zero(t, e.state.Warehouse["red_hare"])
		assert.True(t, e.state.Executed["grant-test"])
	})

	t.Run("an item that is not equipment cannot be put on anyone", func(t *testing.T) {
		d, err := content.Load("../../assets/data")
		require.NoError(t, err)
		d.Nodes = append(d.Nodes, content.Node{ID: "test-bad", Kind: "grant", Item: "insu_high", Count: 1, Character: "liubei", Event: "grant-bad"})
		assert.Error(t, d.Validate())
	})
}
