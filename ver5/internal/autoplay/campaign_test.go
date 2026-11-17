package autoplay

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestPreparationCommandsReproduceTheSelectedPreparation(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := core.NewAt(d, 12, "c12-battle", "hard")
	require.NoError(t, err)
	for len(e.Observe().Learning) > 0 {
		_, err = e.Apply(core.Command{Kind: "learn_close", Actor: e.Observe().Learning[0]})
		require.NoError(t, err)
	}
	before := e.Snapshot()
	copy, err := core.Restore(d, before)
	require.NoError(t, err)
	p := CampaignPlayer(d)
	p.P, p.B, p.Par = 2, 1, 2
	commands := p.PrepareCommands(copy, 1)
	require.NotEmpty(t, commands)
	for _, c := range commands {
		_, err = e.Apply(c)
		require.NoError(t, err)
	}
	assert.Equal(t, copy.Snapshot(), e.Snapshot())
	assert.Equal(t, before.Money, e.Observe().Money, "preparation does not grant money")
	assert.LessOrEqual(t, e.Observe().Inventory["insu_mid"], before.Inventory["insu_mid"])
}

func TestCancelledSearchDoesNotPrepareTheGame(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := core.NewAt(d, 12, "c01-battle", "hard")
	require.NoError(t, err)
	before := e.Snapshot()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := CampaignPlayer(d)
	p.Context = ctx
	assert.Empty(t, p.PrepareCommands(e, 1))
	assert.Equal(t, before, e.Snapshot())
}
