package gui

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestSmithGiftsConfirmSound(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s := core.New(d, 1).Snapshot()
	s.Node = "c02-k2"
	g := &Game{S: &session.Session{Data: d}}
	g.S.Engine, err = core.Restore(d, s)
	require.NoError(t, err)

	trace, err := os.CreateTemp(t.TempDir(), "sfx")
	require.NoError(t, err)
	previous := sfxTrace
	sfxTrace = trace
	t.Cleanup(func() { sfxTrace = previous; trace.Close() })

	for _, gift := range []struct{ item, next string }{
		{"qinglong_blade", "c02-k4"},
		{"serpent_spear", "c02-k6"},
	} {
		t.Run(gift.item, func(t *testing.T) {
			r := g.send(session.Request{Op: "command", Command: core.Command{Kind: "next"}})
			require.True(t, r.OK, r.Error)
			require.Len(t, g.modals, 1)
			m := g.modals[0]
			assert.Equal(t, []gain{{id: gift.item, n: 1, first: true}}, m.gains)
			assert.Equal(t, 1, g.S.View().Warehouse[gift.item])
			assert.Equal(t, gift.next, g.S.View().Node)

			m.piece, m.chip = gift.item, equipChip{"trait", "keen"}
			g.closeModal()
			assert.Empty(t, m.chip.id)
			assert.Equal(t, gift.item, m.piece)
			g.closeModal()
			assert.Empty(t, m.piece)
			require.Len(t, g.modals, 1)
			g.closeModal()
			assert.Empty(t, g.modals)
		})
	}
	g.closeModal()
	b, err := os.ReadFile(trace.Name())
	require.NoError(t, err)
	assert.Equal(t, "0 close\n0 close\n0 ok\n0 close\n0 close\n0 ok\n", string(b))
}

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
	assert.Equal(t, []gain{{"spear", 1, true, false}, {"tonic", 3, false, false}, {"sword", 1, false, false}, {content.Money, 50, false, false}}, acquireGains(before, events))
	joined := append(events, core.Event{Kind: "join", Target: "zhaoyun"})
	assert.Equal(t, gain{id: "zhaoyun", n: 1, officer: true}, acquireGains(before, joined)[0], "the officer who joined comes before the pieces")
}

// Closing takes off the trait window, then the piece's, and only then the cards.
func TestAcquireClosesOneLayerAtATime(t *testing.T) {
	m := &modal{kind: "acquire", gains: []gain{{id: "spear", n: 1, first: true}}, piece: "spear", chip: equipChip{"trait", "keen"}}
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
	assert.Equal(t, []gain{{id: "gujeong_blade", n: 1, first: true}}, m.gains)
}

// 조운 joining in the story opens the acquire window with his card.
func TestStoryJoinOpensAcquire(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s := core.New(d, 1).Snapshot()
	s.Node = "c10-19"
	g := &Game{S: &session.Session{Data: d}}
	g.S.Engine, err = core.Restore(d, s)
	require.NoError(t, err)
	r := g.send(session.Request{Op: "command", Command: core.Command{Kind: "next"}})
	require.True(t, r.OK, r.Error)
	require.NotEmpty(t, g.modals)
	m := g.modals[len(g.modals)-1]
	assert.Equal(t, "acquire", m.kind)
	assert.Equal(t, gain{id: "zhaoyun", n: 1, officer: true}, m.gains[0])
	assert.True(t, g.joined("zhaoyun"))
	assert.False(t, g.joined("gujeong_blade"))
}
