package autoplay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"srpg/internal/content"
	"srpg/internal/core"
)

func TestValuePlayerTakesAWinEvenWhenTheNetworkScoresItPoorly(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	require.NoError(t, err)
	d.Campaign.Start = "prep1"
	d.Rules.HitFloor, d.Rules.HitCeil = 100, 100
	d.Stages[0].Victory = []content.Goal{{If: []string{"defeated:boss"}}}
	d.Stages[0].Enemies[0].X, d.Stages[0].Enemies[0].Y = 2, 2
	e := core.New(d, 1)
	_, err = e.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	for _, c := range []core.Command{
		{Kind: "place", Actor: "hero", X: 1, Y: 2},
		{Kind: "place", Actor: "archer1", X: 1, Y: 3},
		{Kind: "deployment_finish"},
	} {
		_, err := e.Apply(c)
		require.NoError(t, err)
	}
	snap := e.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].ID == "boss" {
			snap.Units[i].HP = 1
		}
	}
	e, err = core.Restore(d, snap)
	require.NoError(t, err)
	net := NewMLP([]int{StateLen, 1}, 1)
	net.InMean, net.InStd = make([]float64, StateLen), make([]float64, StateLen)
	net.OutMean, net.OutStd = []float64{0}, []float64{1}
	for i := range StateLen {
		net.InStd[i], net.W[0][i] = 1, 0
	}
	net.W[0][15] = -100 // the win input never appeared in the nonterminal training states
	p := &ValuePlayer{D: d, G: DefaultGenome(), Net: net, K: 10000}
	c := p.Act(e, "hero")
	_, err = e.Apply(c)
	require.NoError(t, err)
	assert.Equal(t, "victory", e.Observe().Result)
}
