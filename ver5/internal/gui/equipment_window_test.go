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
	g := &Game{S: &session.Session{Data: d, Engine: e}}
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

func TestEquipmentAdviceTracksSharedStock(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	e, err := core.NewAt(d, 1, "c14-battle", "")
	require.NoError(t, err)
	st := e.Snapshot()
	st.Learning = nil
	st.Warehouse = map[string]int{"fire_manual": 1}
	for _, c := range st.Characters {
		if c.ID == "liubei" || c.ID == "guanyu" {
			delete(c.Equipment, "accessory")
		}
	}
	e, err = core.Restore(d, st)
	require.NoError(t, err)
	g := &Game{S: &session.Session{Data: d, Engine: e}}
	reason := func(actor string) string {
		return g.equipmentAdvice(g.S.View(), actor)["accessory"]["fire_manual"]
	}
	assert.Contains(t, reason("liubei"), "사용 가능")
	assert.NotEmpty(t, reason("guanyu"))

	_, err = e.Apply(core.Command{Kind: "equip", Actor: "liubei", Item: "fire_manual"})
	require.NoError(t, err)
	assert.Empty(t, reason("liubei"))
	assert.Empty(t, reason("guanyu"))

	// A different save may come back with the same revision.
	st.Revision = e.Observe().Revision
	g.S.Engine, err = core.Restore(d, st)
	require.NoError(t, err)
	assert.NotEmpty(t, reason("liubei"))
	g.S.Settings.NoGuide = true
	assert.Empty(t, reason("liubei"))
}

func TestShopAdviceNamesWhoGainsAndHonoursNoGuide(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	start := core.New(d, 1)
	for i := 0; i < 200 && start.Observe().Dialogue.Kind != "battle"; i++ {
		start.Apply(core.Command{Kind: "next"})
	}
	snap := start.Snapshot()
	require.NotEmpty(t, snap.Characters)
	snap.Characters[0].Equipment = map[string]string{}
	snap.Warehouse = map[string]int{}
	e, err := core.Restore(d, snap)
	require.NoError(t, err)
	s := &session.Session{Data: d, Engine: e}
	g := &Game{S: s}
	o := e.Observe()
	item := ""
	for _, id := range sorted(d.Equipment) {
		if ups := e.EquipmentUpgradesFor(id); len(ups) > 0 {
			item = id
			break
		}
	}
	require.NotEmpty(t, item, "some piece strengthens an unarmed officer")
	t.Run("lists the officers a purchase would improve", func(t *testing.T) {
		ups := g.shopAdvice(o, item)
		require.NotEmpty(t, ups)
		assert.NotEmpty(t, g.equipmentReason(ups[0]))
		assert.Equal(t, item, g.shopGain.item)
	})
	t.Run("the no-guide setting hides it", func(t *testing.T) {
		g.S.Settings.NoGuide = true
		assert.Empty(t, g.shopAdvice(o, item))
	})
}
