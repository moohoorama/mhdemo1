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

func TestPickerListsTheWornPieceOnceAndOnlyWhatFits(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	start := core.New(d, 1)
	for i := 0; i < 200 && start.Observe().Dialogue.Kind != "battle"; i++ { // past the opening, to the walk where pieces change
		start.Apply(core.Command{Kind: "next"})
	}
	snap := start.Snapshot()
	require.NotEmpty(t, snap.Characters)
	snap.Characters[0].Equipment = map[string]string{"armor": "leather_armor"}
	snap.Warehouse = map[string]int{"leather_armor": 2, "tangni_armor": 1, "cotton_clothes": 1, "iron_sword": 1}
	e, err := core.Restore(d, snap)
	require.NoError(t, err)
	g := &Game{S: &session.Session{Data: d, Engine: e}, itemEffects: map[string]itemEffect{}}
	o := e.Observe()
	of := g.officer(o, snap.Characters[0].ID)
	ids := []string{}
	for _, r := range g.pickable(o, of, "armor") {
		ids = append(ids, r.id)
	}
	assert.Equal(t, pickNothing, ids[0])
	assert.Equal(t, 1, countOf(ids, "leather_armor"))
	assert.Contains(t, ids, "tangni_armor")
	assert.NotContains(t, ids, "cotton_clothes", "a robe the first officer's class cannot wear is left out")
	assert.NotContains(t, ids, "iron_sword")
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

func TestBonusGrades(t *testing.T) {
	for v, want := range map[float64]string{1: "D", 2: "C", 3: "B", 4: "A", 5: "S", 1.5: "+1.5", 6: "+6"} {
		assert.Equal(t, want, bonusGrade(v), v)
	}
	b := [6]float64{3, 0, 1}
	assert.Equal(t, "공격 B · 정신 D", bonusText(b, nil))
	assert.Equal(t, "공격 B (+39) · 정신 D (+13)", bonusText(b, []statDelta{{"공격", 39}, {"정신", 13}}))
	assert.Equal(t, "없음", bonusText([6]float64{}, nil))
}
