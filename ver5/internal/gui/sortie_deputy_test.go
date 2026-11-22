package gui

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/core"
	"srpg/internal/session"
)

// TestSortieDeputy draws the sortie preparation of c20-battle with a benched officer as the first officer's deputy (the sortie-deputy demo) on
// the phone at 130% and 207% (914×411) and on the desktop, and checks the candidates' table and that no two buttons
// share area or leave the screen on any tab.
func TestSortieDeputy(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "sortie-deputy", Node: "c20-battle"})
	require.NoError(t, err)
	g := d.g
	d.step(1)
	draw := func(sc image.Point, scale int) {
		g.S.Settings.UIScale = scale
		g.fit(float64(sc.X), float64(sc.Y))
		g.advance(1.0 / 60)
		dst := ebiten.NewImage(sc.X, sc.Y)
		defer dst.Deallocate()
		g.Draw(dst)
	}
	draw(image.Pt(2400, 1080), 130)

	t.Run("the deputy now on top, the rest by weight, those who cannot serve last", func(t *testing.T) {
		o := g.S.View()
		lead := g.officer(o, g.selected)
		require.NotNil(t, lead)
		require.NotEmpty(t, lead.Deputy)
		assert.Equal(t, tableSort{"weight", 2}, g.party.deputies.order)
		rows := g.deputyRows(o)
		sortRows(rows, g.deputyColumns(), g.party.deputies.order, deputyRank)
		require.NotEmpty(t, rows)
		assert.True(t, rows[0].current)
		for i := 2; i < len(rows); i++ {
			if rows[i].reason == "" && rows[i-1].reason == "" {
				assert.GreaterOrEqual(t, rows[i-1].weight, rows[i].weight, "%s before %s", rows[i-1].name, rows[i].name)
			}
			if rows[i-1].reason != "" {
				assert.NotEmpty(t, rows[i].reason, "%s can serve but comes after one who cannot", rows[i].name)
			}
		}
		for _, r := range rows {
			if r.reason == "" {
				assert.Greater(t, r.weight, 0.0, r.name)
			}
		}
	})

	t.Run("a trait list longer than its lines ends in +n", func(t *testing.T) {
		ids := make([]string, 0, 12)
		for _, id := range sorted(g.S.Data.Traits) {
			if len(ids) < 12 {
				ids = append(ids, id)
			}
		}
		chips, used := g.layoutSmallChips(ids, 0, 300, 2, false, true)
		assert.LessOrEqual(t, used, 2)
		last := chips[len(chips)-1]
		assert.Equal(t, fmt.Sprintf("+%d", len(ids)-(len(chips)-1)), last.label)
		all, usedAll := g.layoutSmallChips(ids, 0, 300, 2, true, true)
		assert.Greater(t, usedAll, 2)
		assert.Equal(t, "접기", all[len(all)-1].label)
		assert.Len(t, all, len(ids)+1)
	})

	t.Run("a deputy cannot be added to the sortie", func(t *testing.T) {
		lead := g.selected
		d.step(120)
		d.step(124)
		require.NotEmpty(t, g.servedBy(g.S.View(), g.selected))
		draw(image.Pt(1280, 850), 100)
		var start bool
		for _, b := range g.buttons {
			assert.NotEqual(t, "◀", b.label)
			assert.NotEqual(t, "▶", b.label)
			start = start || b.label == "출진 ›"
		}
		assert.True(t, start)
		g.selected = lead
	})

	t.Run("clicking the deputy portrait selects and reveals the deputy", func(t *testing.T) {
		lead := g.selected
		deputy := g.officer(g.S.View(), lead).Deputy
		require.NotEmpty(t, deputy)
		g.party.sortieView, g.party.officerList.top = 0, 0
		draw(image.Pt(2400, 1080), 207)
		g.pressLabel(g.name(deputy))
		require.Equal(t, deputy, g.selected)
		assert.Zero(t, g.party.sortieView)
		require.True(t, g.party.officerList.revealSelected)
		draw(image.Pt(2400, 1080), 207)
		assert.False(t, g.party.officerList.revealSelected)
		assert.Positive(t, g.party.officerList.top)
		g.selected = lead
	})

	t.Run("the wheel scrolls candidates beneath the pinned deputy", func(t *testing.T) {
		lead := g.selected
		deputy := g.officer(g.S.View(), lead).Deputy
		require.NotEmpty(t, deputy)
		g.party.sortieView = 5
		g.deputyFresh()
		draw(image.Pt(2400, 1080), 207)
		area := slices.IndexFunc(g.wheelAreas, func(a wheelArea) bool { return a.l == &g.party.deputies })
		require.NotEqual(t, -1, area)
		bounds := g.wheelAreas[area].r
		g.wheelAreas[area].l.scroll(-20)
		draw(image.Pt(2400, 1080), 207)
		require.Positive(t, g.party.deputies.top)

		g.deputyPick = lead
		cursorPos = func() (int, int) { return bounds.Min.X + 15, bounds.Min.Y + 32 + touchH/2 }
		defer func() { cursorPos = func() (int, int) { return -1000, -1000 } }()
		g.leftClick()
		require.NotNil(t, g.tableDrag)
		pick := g.tableDrag.pick
		g.tableDrag = nil
		require.NotNil(t, pick)
		pick()
		assert.Equal(t, deputy, g.deputyPick)
	})

	screens := []struct {
		size  image.Point
		scale int
	}{{image.Pt(2400, 1080), 130}, {image.Pt(2400, 1080), 207}, {image.Pt(1280, 850), 100}}
	for _, sc := range screens {
		for tab := range sortieTabs {
			t.Run(fmt.Sprintf("buttons/%dx%d@%d/%s", sc.size.X, sc.size.Y, sc.scale, sortieTabs[tab]), func(t *testing.T) {
				g.party.sortieView = []int{2, 4, 3, 5}[tab]
				draw(sc.size, sc.scale)
				screen := image.Rect(0, 0, uiW, uiH)
				for i, a := range g.buttons {
					if a.rect.Empty() {
						continue
					}
					assert.True(t, a.rect.In(screen), "%q leaves the screen: %v", a.label, a.rect)
					for _, b := range g.buttons[i+1:] {
						if a.label == "" || b.label == "" { // a row's own press lies under what it holds
							continue
						}
						if in := a.rect.Intersect(b.rect); in.Dx() > 2 && in.Dy() > 2 {
							t.Errorf("%q %v overlaps %q %v", a.label, a.rect, b.label, b.rect)
						}
					}
				}
			})
		}
	}
}

func TestSortieTransition(t *testing.T) {
	var transition sortieTransition
	check := func(now float64, active bool, want float64) {
		t.Helper()
		assert.InDelta(t, want, transition.update(now, active), 1e-9)
	}
	check(1, false, 0)
	check(2, true, 0)
	check(2.05, true, 0.5)
	check(2.1, true, 1)
	check(3, true, 1)
	check(4, false, 1)
	check(4.05, false, 0.5)
	check(4.1, false, 0)
	check(5, true, 0)
	check(5.05, false, 0.5)
	check(5.075, true, 0.25)
	check(5.1, true, 0.5)
}

func TestStoryManagementDeputy(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "story-farewell"})
	require.NoError(t, err)
	g := d.g
	defer g.Close()
	g.fit(1280, 850)
	dst := ebiten.NewImage(1280, 850)
	defer dst.Deallocate()

	result := g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}})
	require.True(t, result.OK, result.Error)
	require.Equal(t, "c13-wait-zhaoyun", g.S.View().Node)
	result = g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn", Actor: "liubei", Trait: "deputy_appoint"}})
	require.True(t, result.OK, result.Error)
	g.selected = "liubei"
	g.Draw(dst)
	g.pressLabel("관리")
	require.Equal(t, "party", g.overlay())
	g.Draw(dst)
	g.pressLabel("부관")
	require.Equal(t, 3, g.party.tab)
	g.Draw(dst)
	g.deputyPick = "jianyong"
	g.Draw(dst)
	g.pressLabel("부관 임명")
	require.Equal(t, "jianyong", g.officer(g.S.View(), "liubei").Deputy)
	g.Draw(dst)
	g.pressLabel("부관 해임")
	assert.Empty(t, g.officer(g.S.View(), "liubei").Deputy)

	require.NotContains(t, g.S.View().Deployment, "jianyong")
	result = g.request(session.Request{Op: "command", Command: core.Command{Kind: "learn", Actor: "jianyong", Trait: "deputy_appoint"}})
	require.True(t, result.OK, result.Error)
	g.selected = "jianyong"
	g.Draw(dst)
	g.deputyPick = "guanyu"
	g.Draw(dst)
	g.pressLabel("부관 임명")
	assert.Equal(t, "guanyu", g.officer(g.S.View(), "jianyong").Deputy)
}
