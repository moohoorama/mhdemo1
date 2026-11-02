package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cell(s string) Cell {
	c, ok := ParseCell(s)
	if !ok {
		panic(s)
	}
	return c
}

func loadReal(t *testing.T) *Data {
	t.Helper()
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	return d
}

func TestTopmostLayerThatSaysAValueWins(t *testing.T) {
	d := loadReal(t)
	foot := d.Classes[d.Characters[d.Campaign.Roster[0]].Class].Terrain
	cav := d.Classes["light_cavalry"].Terrain
	assert.Equal(t, Terrain{Factor: 1.05, Cost: 1}, d.Move(cav, cell("g--"), false), "grass")
	assert.Equal(t, Terrain{Factor: 0.9, Cost: 2}, d.Move(cav, cell("d--"), false), "wasteland")
	assert.Equal(t, Terrain{Factor: 1.1, Cost: 1}, d.Move(cav, cell("dv-"), false), "a village says its own cost over the wasteland's")
	assert.Equal(t, Terrain{Factor: 0.8, Cost: 2}, d.Move(cav, cell("gt-"), false), "a tree on grass")
	assert.Zero(t, d.Move(foot, cell("w--"), false).Cost, "water")
	assert.Positive(t, d.Move(foot, cell("wb-"), false).Cost, "a bridge over water can be crossed")
	assert.Zero(t, d.Move(foot, cell("c--"), false).Cost, "a wall")
}

func TestStatesOverlayTheLayersBelow(t *testing.T) {
	d := loadReal(t)
	cav := d.Classes["light_cavalry"].Terrain
	assert.Equal(t, d.Move(cav, cell("gt-"), false), d.Move(cav, cell("gte"), false), "embers only burn: the tree's factor and cost stay")
	assert.InDelta(t, 0.95, d.Move(cav, cell("g-w"), false).Factor, 1e-9, "wet grass: ten points off the grass's 105%")
	assert.InDelta(t, 0.7, d.Move(cav, cell("gtw"), false).Factor, 1e-9, "and off the tree's 80%")
	assert.Equal(t, Terrain{Factor: 1.05, Cost: 3}, d.Move(cav, cell("g-r"), false), "rubble: three steps, the grass's factor")
	assert.Zero(t, d.Move(cav, cell("g-F"), false).Cost, "flames cannot be entered")
	assert.Zero(t, d.Move(cav, cell("gvF"), false).Cost, "not even over a village")
	assert.Equal(t, d.Move(cav, cell("gt-"), false), d.Move(cav, cell("gts"), false), "soot changes nothing")
	assert.Equal(t, d.Props(cell("gv-")), d.Props(cell("gvs")), "soot changes nothing")
}

func TestRecoveryAndConditionsAcrossLayers(t *testing.T) {
	d := loadReal(t)
	assert.Equal(t, CellProps{RestHP: 10, RestMP: 2, Village: true}, d.Props(cell("gv-")))
	assert.Equal(t, CellProps{RestHP: -10, RestMP: 2, Village: true}, d.Props(cell("gve")), "embers burn away a village's healing, its spell recovery stays")
	assert.Equal(t, CellProps{RestHP: 20, RestMP: 3, Castle: true}, d.Props(cell("ip-")))
	assert.True(t, d.Props(cell("gt-")).Ambush)
	assert.True(t, d.Props(cell("gtF")).Ambush, "an ambush is a condition: any layer may give it")
	assert.True(t, d.Props(cell("wb-")).Water, "a bridge lies over water")
	assert.False(t, d.Props(cell("g-F")).Water)
}

func TestWhatStandsOnWhat(t *testing.T) {
	d := loadReal(t)
	for s, ok := range map[string]bool{
		"g--": true, "gt-": true, "dt-": true, "wt-": false, "it-": false, "ct-": false, "ht-": false,
		"gv-": true, "dv-": true, "iv-": true, "wv-": false, "cv-": false,
		"gk-": true, "ik-": true, "go-": true, "io-": true, "do-": false,
		"ip-": true, "gp-": false, "cp-": false,
		"wb-": true, "gb-": false,
		"g-e": true, "g-F": true, "g-s": true, "g-x": false, "x--": false, "gx-": false,
		"m--": true, "mt-": false, "mz-": false, "gz-": true, "dz-": true, "iz-": true, "wz-": false, "cz-": false,
		"wbd": true, "wbx": true, "gbd": false, "g-d": false, "gvx": false, "gtd": false,
	} {
		assert.Equal(t, ok, d.ValidCell(cell(s)) == "", s)
	}
}

func TestEveryRealMapCellIsValidAndDrawsTheOldGlyph(t *testing.T) {
	d := loadReal(t)
	require.NotEmpty(t, d.Maps)
	glyphs := map[string]byte{"g--": '.', "d--": 'd', "w--": '~', "gt-": 'f', "dv-": 'v', "dk-": 'k', "c--": 'c', "i--": 'i', "wb-": 'b', "ip-": 'g', "go-": 'o', "h--": 'h'}
	for id, m := range d.Maps {
		art := m.Graphics(d)
		for y := 0; y < m.Play.H; y++ {
			for x := 0; x < m.Play.W; x++ {
				c := m.Cell(m.Play.X+x, m.Play.Y+y)
				assert.Empty(t, d.ValidCell(c), id)
				if want, ok := glyphs[c.String()]; ok {
					assert.Equal(t, want, art.Tiles[y][x], "%s (%d,%d)", id, x, y)
				}
			}
		}
	}
}

func TestGatesFencesAndBrokenBridgesBlockTheWay(t *testing.T) {
	d := loadReal(t)
	foot := d.Classes[d.Characters[d.Campaign.Roster[0]].Class].Terrain
	cav := d.Classes["light_cavalry"].Terrain
	for _, g := range []string{foot, cav} {
		assert.Zero(t, d.Move(g, cell("m--"), false).Cost, "a gate wall")
		assert.Zero(t, d.Move(g, cell("gz-"), false).Cost, "a fence")
		assert.Zero(t, d.Move(g, cell("wbx"), false).Cost, "a broken bridge")
		assert.Positive(t, d.Move(g, cell("wb-"), false).Cost, "a whole bridge")
		assert.Positive(t, d.Move(g, cell("wbd"), false).Cost, "a damaged bridge can still be crossed")
	}
	assert.Equal(t, 2, d.Move(foot, cell("wbd"), false).Cost, "...slowly")
	assert.True(t, d.Props(cell("wbx")).Water, "a broken bridge lies over water")
}

func TestBoatsOpenTheWater(t *testing.T) {
	d := loadReal(t)
	t.Run("without boats water stops everyone", func(t *testing.T) {
		for group := range d.Terrain {
			assert.Zero(t, d.Move(group, cell("w--"), false).Cost, group)
		}
	})
	t.Run("with boats foot soldiers are at home, horses slow, machines stay ashore", func(t *testing.T) {
		assert.Equal(t, Terrain{Factor: 1.1, Cost: 1}, d.Move("infantry", cell("w--"), true))
		assert.Equal(t, Terrain{Factor: 1.1, Cost: 1}, d.Move("bandit", cell("w--"), true))
		assert.Equal(t, Terrain{Factor: 1, Cost: 1}, d.Move("archer", cell("w--"), true))
		assert.Equal(t, Terrain{Factor: 1, Cost: 1}, d.Move("scholar", cell("w--"), true))
		assert.Equal(t, Terrain{Factor: 0.9, Cost: 2}, d.Move("cavalry", cell("w--"), true))
		assert.Zero(t, d.Move("machine", cell("w--"), true).Cost)
	})
	t.Run("bridges and their states still lie over the boat rule", func(t *testing.T) {
		assert.Equal(t, d.Move("cavalry", cell("wb-"), false), d.Move("cavalry", cell("wb-"), true))
		assert.Zero(t, d.Move("infantry", cell("wbx"), true).Cost, "a broken bridge still blocks the way")
	})
	t.Run("land is unchanged", func(t *testing.T) {
		assert.Equal(t, d.Move("cavalry", cell("g--"), false), d.Move("cavalry", cell("g--"), true))
	})
}
