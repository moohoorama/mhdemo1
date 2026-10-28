package gui

import (
	"fmt"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// TestNoOverlappingButtons draws the battle UI headlessly at phone, desktop and tablet shapes and
// every UI size, and fails when two buttons share area or one leaves the screen. Buttons are
// recorded while drawing, so the check reads the layout and never looks at pixels.
func TestNoOverlappingButtons(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "hudpanel"})
	require.NoError(t, err)
	g := d.g
	ally := d.firstAlly()
	screens := []image.Point{{1280, 850}, {1920, 1080}, {2400, 1080}, {1024, 768}, {800, 480}}
	scenes := map[string]func(){
		"minimap":      func() { g.hudBase = "minimap" },
		"unit-none":    func() { g.hudBase = "unit" },
		"unit":         func() { g.selected = ally },
		"units":        func() { g.hudBase = "units" },
		"log":          func() { g.hudBase = "log" },
		"misc":         func() { g.hudBase = "misc" },
		"menu":         func() { g.overlays = []string{"menu"} },
		"settings":     func() { g.overlays = []string{"settings"} },
		"objective":    func() { g.modals = []*modal{{kind: "objective", manual: true}} },
		"roster-basic": func() { g.overlays = []string{"roster"} },
		"roster-stats": func() { g.overlays, g.rosterList.kind = []string{"roster"}, 1 },
		"roster-bio":   func() { g.overlays, g.rosterList.kind = []string{"roster"}, 2 },
		"roster-equip": func() { g.overlays, g.rosterList.kind = []string{"roster"}, 3 },
	}
	for name, setup := range scenes {
		for _, scale := range []int{70, 100, 130, 160, 200} {
			for _, sc := range screens {
				t.Run(fmt.Sprintf("%s/%dx%d@%d", name, sc.X, sc.Y, scale), func(t *testing.T) {
					g.hudBase, g.selected, g.overlays, g.modals, g.rosterList = "", "", nil, nil, unitList{filter: "all"}
					g.windows = nil
					g.S.Settings.UIScale = scale
					setup()
					g.fit(float64(sc.X), float64(sc.Y))
					g.advance(1.0 / 60)
					dst := ebiten.NewImage(sc.X, sc.Y)
					defer dst.Deallocate()
					g.Draw(dst)
					screen := image.Rect(0, 0, uiW, uiH)
					for i, a := range g.buttons {
						if a.rect.Empty() {
							continue
						}
						if !a.rect.In(screen) {
							t.Errorf("%q leaves the screen: %v", a.label, a.rect)
						}
						for _, b := range g.buttons[i+1:] {
							if in := a.rect.Intersect(b.rect); in.Dx() > 2 && in.Dy() > 2 {
								t.Errorf("%q %v overlaps %q %v", a.label, a.rect, b.label, b.rect)
							}
						}
					}
				})
			}
		}
	}
}
