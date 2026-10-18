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
	assert.Equal(t, Terrain{Factor: 1.1, Cost: 1}, d.Move(cav, cell("g--")), "grass")
	assert.Equal(t, Terrain{Factor: 0.9, Cost: 2}, d.Move(cav, cell("d--")), "wasteland")
	assert.Equal(t, Terrain{Factor: 1.1, Cost: 1}, d.Move(cav, cell("dv-")), "a village says its own cost over the wasteland's")
	assert.Equal(t, Terrain{Factor: 0.8, Cost: 2}, d.Move(cav, cell("gt-")), "a tree on grass")
	assert.Zero(t, d.Move(foot, cell("w--")).Cost, "water")
	assert.Positive(t, d.Move(foot, cell("wb-")).Cost, "a bridge over water can be crossed")
	assert.Zero(t, d.Move(foot, cell("c--")).Cost, "a wall")
}

func TestStatesOverlayTheLayersBelow(t *testing.T) {
	d := loadReal(t)
	cav := d.Classes["light_cavalry"].Terrain
	assert.Equal(t, Terrain{Factor: 0.8, Cost: 2}, d.Move(cav, cell("gte")), "embers: 80%, the tree's cost stays")
	assert.Equal(t, Terrain{Factor: 0.8, Cost: 1}, d.Move(cav, cell("g-e")), "embers on grass")
	assert.Zero(t, d.Move(cav, cell("g-F")).Cost, "flames cannot be entered")
	assert.Zero(t, d.Move(cav, cell("gvF")).Cost, "not even over a village")
	assert.Equal(t, d.Move(cav, cell("gt-")), d.Move(cav, cell("gts")), "soot changes nothing")
	assert.Equal(t, d.Props(cell("gv-")), d.Props(cell("gvs")), "soot changes nothing")
}

func TestRecoveryAndConditionsAcrossLayers(t *testing.T) {
	d := loadReal(t)
	assert.Equal(t, CellProps{RestHP: 10, RestMP: 2}, d.Props(cell("gv-")))
	assert.Equal(t, CellProps{RestHP: -20, RestMP: 2}, d.Props(cell("gve")), "embers burn away a village's healing, its spell recovery stays")
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
