package content

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Armies on one field must read apart: no two of their colours (troops or officers of a faction) lie close, and the
// pairs the user asked to be told about (Sun officers orange against Yuan troops orange, Yuan officers yellow against
// Turban troops yellow) are listed wherever a stage fields both.
func TestFactionColorsStandApartInEveryStage(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	shade := func(f, role string) [3]float64 {
		cols := d.Factions[f].Colors
		if role == "officer" && len(d.Factions[f].OfficerColors) > 0 {
			cols = d.Factions[f].OfficerColors
		}
		var rgb [3]float64
		for i := range rgb {
			v, err := strconv.ParseUint(cols[1][1+2*i:3+2*i], 16, 8)
			require.NoError(t, err)
			rgb[i] = float64(v)
		}
		return rgb
	}
	type army struct{ faction, role string }
	risky := [][2]army{{{"wu", "officer"}, {"yuan", "troop"}}, {{"yuan", "officer"}, {"turban", "troop"}}}
	for _, st := range d.Stages {
		fielded := map[army]bool{{"shu", "officer"}: true}
		note := func(sp Spawn) {
			c := d.Characters[sp.Character]
			f := sp.Faction
			if f == "" {
				f = st.Faction
			}
			if f == "" || st.Faction == "" && sp.Faction == "" {
				f = c.Look.Faction
			}
			role := "officer"
			if c.Template && sp.Rank != "officer" && sp.Rank != "commander" && sp.Rank != "boss" {
				role = "troop"
			}
			if _, ok := d.Factions[f]; ok {
				fielded[army{f, role}] = true
			}
		}
		for _, sp := range append(append([]Spawn{}, st.Enemies...), st.Party...) {
			note(sp)
		}
		for _, e := range st.Events {
			for _, a := range e.Do {
				for _, sp := range a.Spawns {
					note(sp)
				}
			}
		}
		for _, p := range risky {
			if fielded[p[0]] && fielded[p[1]] {
				t.Logf("%s fields %v and %v together", st.ID, p[0], p[1])
			}
		}
		for a := range fielded {
			for b := range fielded {
				if a.faction >= b.faction || a.faction == b.faction {
					continue
				}
				x, y := shade(a.faction, a.role), shade(b.faction, b.role)
				dist := math.Sqrt((x[0]-y[0])*(x[0]-y[0]) + (x[1]-y[1])*(x[1]-y[1]) + (x[2]-y[2])*(x[2]-y[2]))
				assert.Greater(t, dist, 60.0, fmt.Sprintf("%s: %v and %v", st.ID, a, b))
			}
		}
	}
	for id, f := range d.Factions {
		if len(f.OfficerColors) > 0 {
			assert.NotEqual(t, f.Colors[1], f.OfficerColors[1], id)
		}
	}
}
