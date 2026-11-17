package ai

import (
	"testing"

	"github.com/stretchr/testify/require"

	"srpg/internal/content"
	"srpg/internal/core"
)

// BenchmarkRankCampaignBattle/liubei-8  1000  481300099 ns/op  59156559 B/op  149303 allocs/op # baseline
// BenchmarkRankCampaignBattle/liubei-8  1000  166781461 ns/op  29680415 B/op   76231 allocs/op # cache strikes within Rank
// BenchmarkRankCampaignBattle/guanyu-8  1000  611308636 ns/op  83084579 B/op  207719 allocs/op # baseline
// BenchmarkRankCampaignBattle/guanyu-8  1000  127823100 ns/op  31664081 B/op   82406 allocs/op # cache strikes within Rank
func BenchmarkRankCampaignBattle(b *testing.B) {
	d, err := content.Load("../../assets/data")
	require.NoError(b, err)
	e, err := core.NewAt(d, 1, "c16-battle", "hard")
	require.NoError(b, err)
	for len(e.Observe().Learning) > 0 {
		_, err := e.Apply(core.Command{Kind: "learn_close", Actor: e.Observe().Learning[0]})
		require.NoError(b, err)
	}
	_, err = e.Apply(core.Command{Kind: "start"})
	require.NoError(b, err)
	commands, err := e.DeploymentPlan()
	require.NoError(b, err)
	for _, c := range commands {
		_, err := e.Apply(c)
		require.NoError(b, err)
	}
	for _, actor := range []string{"liubei", "guanyu"} {
		b.Run(actor, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				Rank(e, actor, Player, true)
			}
		})
	}
}
