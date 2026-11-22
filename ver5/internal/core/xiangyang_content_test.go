package core

import (
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/require"
	"srpg/internal/content"
)

func TestStageXiangyangCampBuffs(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)

	tests := []struct {
		name  string
		camps []content.Point
	}{
		{"서쪽부터", []content.Point{{X: 6, Y: 9}, {X: 6, Y: 17}, {X: 14, Y: 23}, {X: 21, Y: 23}}},
		{"남쪽부터", []content.Point{{X: 21, Y: 23}, {X: 14, Y: 23}, {X: 6, Y: 17}, {X: 6, Y: 9}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := NewAt(d, 1, "c27v-battle", "easy")
			require.NoError(t, err)
			for _, id := range e.Observe().Learning {
				applyOK(t, e, Command{Kind: "learn_close", Actor: id})
			}
			applyOK(t, e, Command{Kind: "start"})
			plan, err := e.DeploymentPlan()
			require.NoError(t, err)
			for _, command := range plan {
				applyOK(t, e, command)
			}

			cao := e.unit("caocao")
			targets := []*Unit{cao, e.unit("caoren"), e.unit("caimao")}
			baseline := make([]map[string]Status, len(targets))
			for n, unit := range targets {
				baseline[n] = maps.Clone(unit.Status)
			}
			for i, p := range tt.camps {
				if guard := e.occupied(p.X, p.Y); guard != nil {
					guard.X, guard.Y = cao.X, cao.Y
				}
				cao.X, cao.Y = p.X, p.Y
				e.battleEvents()
				count := [...]int{1, 2, 3, 5}[i]
				for stage := 1; stage <= len(tt.camps); stage++ {
					key := fmt.Sprintf("battle-B27V-cao-camp-bonus-%d", stage)
					require.Equal(t, stage <= i+1, e.state.Executed[key], key)
				}
				for n, unit := range targets {
					for j, stat := range []string{"morale", "agility", "defense", "attack", "mind"} {
						want := baseline[n][stat].Value
						if unit.ID != "caimao" && j < count {
							want = max(want, 20)
						}
						require.Equal(t, want, unit.Status[stat].Value, "%s, 진채 %d곳, %s", unit.ID, i+1, stat)
					}
				}
			}

			endTurn(t, e)
			require.Equal(t, "battle", e.Observe().Phase)
		})
	}
}
