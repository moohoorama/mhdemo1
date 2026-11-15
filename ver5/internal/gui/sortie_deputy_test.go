package gui

import (
	"fmt"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	screens := []struct {
		size  image.Point
		scale int
	}{{image.Pt(2400, 1080), 130}, {image.Pt(2400, 1080), 207}, {image.Pt(1280, 850), 100}}
	for _, sc := range screens {
		for tab := range sortieTabs {
			t.Run(fmt.Sprintf("buttons/%dx%d@%d/%s", sc.size.X, sc.size.Y, sc.scale, sortieTabs[tab]), func(t *testing.T) {
				g.party.sortieTab = tab
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
