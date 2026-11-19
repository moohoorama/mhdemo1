package content

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEveryStageNamesItsAITurnArmies(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	for _, st := range d.Stages {
		t.Run(st.ID, func(t *testing.T) {
			names := make(map[string]string)
			for _, army := range st.Armies {
				names[army.Faction] = army.Name
			}
			check := func(sp Spawn, enemy bool) {
				faction := d.Characters[sp.Character].Look.Faction
				if enemy && st.Faction != "" {
					faction = st.Faction
				}
				if sp.Faction != "" {
					faction = sp.Faction
				}
				assert.NotEmpty(t, strings.TrimSpace(names[faction]), "%s: %s의 진영 %q에 턴 표시 이름이 없습니다", st.Name, sp.Character, faction)
			}
			for _, sp := range st.Enemies {
				check(sp, true)
			}
			for _, sp := range st.Party {
				if sp.AI {
					check(sp, false)
				}
			}
			for _, event := range st.Events {
				for _, action := range event.Do {
					for _, sp := range action.Spawns {
						if sp.Side == "enemy" || sp.AI {
							check(sp, sp.Side == "enemy" && action.Kind == "spawn")
						}
					}
				}
			}
		})
	}
}
