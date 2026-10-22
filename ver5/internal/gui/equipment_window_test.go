package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestEveryEquipmentAndTraitIconIsOnTheUISheet(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	ui, err := LoadShots("../../assets/graphics", "ui/ui")
	require.NoError(t, err)
	for id, eq := range d.Equipment {
		assert.Contains(t, ui, "item_"+eq.Icon, id)
	}
	for slot, icon := range slotIcons {
		assert.Contains(t, ui, "item_"+icon, slot)
	}
	for id, tr := range d.Traits {
		assert.Contains(t, ui, "trait_"+tr.Kind, id)
		assert.Contains(t, toneColors, tr.Tone, id)
	}
}

func TestPickerListsTheWornPieceOnce(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	g := &Game{S: &session.Session{Data: d, Engine: core.New(d, 1)}, itemEffects: map[string]itemEffect{}}
	o := g.S.Engine.Observe()
	require.NotEmpty(t, o.Characters)
	of := &o.Characters[0]
	of.Equipment = map[string]string{"armor": "leather_armor"}
	o.Warehouse = map[string]int{"leather_armor": 2, "plain_clothes": 1, "iron_sword": 1}
	rows := g.pickable(o, of, "armor")
	assert.Equal(t, pickNothing, rows[0])
	assert.Equal(t, 1, countOf(rows, "leather_armor"))
	assert.Contains(t, rows, "plain_clothes")
	assert.NotContains(t, rows, "iron_sword")
}

func countOf(xs []string, x string) int {
	n := 0
	for _, v := range xs {
		if v == x {
			n++
		}
	}
	return n
}
