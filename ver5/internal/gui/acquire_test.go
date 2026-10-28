package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestAcquireGains(t *testing.T) {
	before := &core.Observation{State: core.State{
		Inventory:  map[string]int{"tonic": 1},
		Warehouse:  map[string]int{},
		Characters: []core.Character{{Equipment: map[string]string{"weapon": "sword"}}},
	}}
	events := []core.Event{
		{Kind: "grant", Target: "spear", Amount: 1},
		{Kind: "damage", Target: "tonic", Amount: 3},
		{Kind: "grant", Target: "tonic", Amount: 2},
		{Kind: "grant", Target: "tonic", Amount: 1},
		{Kind: "grant", Target: "sword", Amount: 1},
		{Kind: "grant", Target: content.Money, Amount: 50},
	}
	assert.Equal(t, []gain{{"spear", 1, true}, {"tonic", 3, false}, {"sword", 1, false}, {content.Money, 50, false}}, acquireGains(before, events))
	assert.Empty(t, acquireGains(before, []core.Event{{Kind: "join", Target: "spear", Amount: 1}}))
}

// Closing takes off the trait window, then the piece's, and only then the cards.
func TestAcquireClosesOneLayerAtATime(t *testing.T) {
	m := &modal{kind: "acquire", gains: []gain{{"spear", 1, true}}, piece: "spear", chip: equipChip{"trait", "keen"}}
	g := &Game{modals: []*modal{m, {kind: "levelup"}}}
	g.closeModal()
	assert.Empty(t, m.chip.id)
	assert.Equal(t, "spear", m.piece)
	g.closeModal()
	assert.Empty(t, m.piece)
	assert.Len(t, g.modals, 2)
	g.closeModal()
	assert.Equal(t, "levelup", g.modals[0].kind)
}

// Sun Jian's sword handed over in the story opens the acquire window with it as a first.
func TestStoryGrantOpensAcquire(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s := core.New(d, 1).Snapshot()
	s.Node = "c05-gift-s"
	g := &Game{S: &session.Session{Data: d}}
	g.S.Engine, err = core.Restore(d, s)
	require.NoError(t, err)
	r := g.send(session.Request{Op: "command", Command: core.Command{Kind: "next"}})
	require.True(t, r.OK, r.Error)
	require.NotEmpty(t, g.modals)
	m := g.modals[len(g.modals)-1]
	assert.Equal(t, "acquire", m.kind)
	assert.Equal(t, []gain{{"gujeong_blade", 1, true}}, m.gains)
}
