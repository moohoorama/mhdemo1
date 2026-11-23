package gui

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserLists(t *testing.T) {
	oldCursor, oldPointer := cursorPos, pointer
	defer func() { cursorPos, pointer = oldCursor, oldPointer }()
	for _, name := range []string{"map-list", "duel-list"} {
		t.Run(name, func(t *testing.T) {
			d, err := NewDemo("../..", DemoOptions{Name: name})
			require.NoError(t, err)
			g := d.g
			defer g.Close()
			var list *tableList
			total, rowH := len(g.S.Data.Stages), listRowH
			if g.editor != nil {
				list = &g.editor.maps
			} else {
				list, total, rowH = &g.gallery.list, 0, 80
				for _, st := range g.S.Data.Stages {
					total += len(st.Duels)
				}
			}
			for _, sc := range []struct{ w, h, scale int }{{1280, 850, 100}, {2400, 1080, 130}, {2400, 1080, 160}, {2048, 1536, 130}, {800, 480, 160}} {
				t.Run(fmt.Sprintf("%dx%d@%d", sc.w, sc.h, sc.scale), func(t *testing.T) {
					g.S.Settings.UIScale = sc.scale
					g.fit(float64(sc.w), float64(sc.h))
					assert.False(t, g.fixedLayout())
					draw := func() { dst := ebiten.NewImage(sc.w, sc.h); defer dst.Deallocate(); g.Draw(dst) }
					list.top = 10000
					draw()
					require.Len(t, g.wheelAreas, 1)
					area := g.wheelAreas[0]
					assert.Same(t, list, area.l)
					fit := area.r.Dy() / rowH
					assert.Equal(t, max(0, total-fit), list.top, "last entries must be reachable")
					for i, a := range g.buttons {
						assert.True(t, a.rect.In(image.Rect(0, 0, uiW, uiH)), "%q outside screen: %v", a.label, a.rect)
						for _, b := range g.buttons[i+1:] {
							assert.True(t, a.rect.Intersect(b.rect).Empty(), "buttons overlap: %v %v", a.rect, b.rect)
						}
					}
					var row button
					for _, b := range g.buttons {
						if b.label == "" && b.rect.Dx() > 100 && b.rect.In(area.r) {
							row = b
						}
					}
					require.NotNil(t, row.click)
					x, y := row.rect.Min.X+20, row.rect.Min.Y+rowH/2
					cursorPos = func() (int, int) { return x, y }
					before := list.top
					pointer.held, pointer.justPressed = true, true
					g.pressButton(slices.IndexFunc(g.buttons, func(b button) bool { return b.rect == row.rect }))
					pointer.justPressed = false
					y -= rowH
					if g.editor != nil {
						require.NoError(t, g.updateEditor())
					} else {
						require.NoError(t, g.updateDuelGallery())
					}
					assert.Equal(t, before+1, list.top, "independent screens must process row drags")
					pointer.held = false
					if g.editor != nil {
						require.NoError(t, g.updateEditor())
						assert.Nil(t, g.editor.doc)
					} else {
						require.NoError(t, g.updateDuelGallery())
						assert.Nil(t, g.gallery.view)
					}
					list.top = 0
					draw()
					assert.Zero(t, list.top)
				})
			}
			// A release on the final row opens its entry, and the return keeps its position.
			g.S.Settings.UIScale = 100
			g.fit(1280, 850)
			list.top = 10000
			dst := ebiten.NewImage(1280, 850)
			defer dst.Deallocate()
			g.Draw(dst)
			saved := list.top
			var row button
			for _, b := range g.buttons {
				if b.label == "" && b.rect.Dx() > 100 {
					row = b
				}
			}
			require.NotNil(t, row.click)
			cursorPos = func() (int, int) { return row.rect.Min.X + 20, row.rect.Min.Y + rowH/2 }
			pointer.held, pointer.justPressed = true, true
			g.pressButton(slices.IndexFunc(g.buttons, func(b button) bool { return b.rect == row.rect }))
			pointer.held, pointer.justPressed = false, false
			if g.editor != nil {
				require.NoError(t, g.updateEditor())
				require.NotNil(t, g.editor.doc)
				assert.Equal(t, g.S.Data.Stages[len(g.S.Data.Stages)-1].ID, g.editor.doc.Stage.ID)
				g.editor.doc = nil
				assert.Equal(t, saved, g.editor.maps.top)
			} else {
				require.NoError(t, g.updateDuelGallery())
				require.NotNil(t, g.gallery.view)
				last := ""
				for _, st := range g.S.Data.Stages {
					for _, d := range st.Duels {
						last = d.ID
					}
				}
				assert.Equal(t, last, g.gallery.id)
				g.gallery.view = nil
				assert.Equal(t, saved, g.gallery.list.top)
			}
		})
	}
}

func TestEditorChoiceLists(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "editor-choices"})
	require.NoError(t, err)
	g := d.g
	defer g.Close()
	g.fit(1280, 850)
	dst := ebiten.NewImage(1280, 850)
	defer dst.Deallocate()
	for _, kind := range []string{"officers", "classes", "traits", "equipment", "scenario"} {
		t.Run(kind, func(t *testing.T) {
			g.editor.popup = kind
			for family := range g.S.Data.MapEditor.Families {
				g.editor.family = family
				g.editor.choices.top = 10000
				g.Draw(dst)
				require.Len(t, g.wheelAreas, 1)
				assert.Same(t, &g.editor.choices, g.wheelAreas[0].l)
				for i, a := range g.buttons {
					assert.True(t, a.rect.In(image.Rect(160, 65, 1120, 785)), "%q leaves the editor panel: %v", a.label, a.rect)
					for _, b := range g.buttons[i+1:] {
						assert.True(t, a.rect.Intersect(b.rect).Empty(), "buttons overlap: %q %v %q %v", a.label, a.rect, b.label, b.rect)
					}
				}
			}
		})
	}
}
