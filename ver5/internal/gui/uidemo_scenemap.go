package gui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
)

func init() { demos["scenemap"] = demoSceneMap }

// demoSceneMap shows one scenario scene map (-stage S_village, S_road, ...) on its own, fitted to the screen,
// as the scene camera sees it: at twice the canvas size around the middle of its playing area.
func demoSceneMap(d *Demo) error {
	id := d.stage
	art := d.g.assets.Maps[id]
	if art == nil {
		return fmt.Errorf("no scene map %q (pass one with -stage)", id)
	}
	f := NewField(d.g.assets, art)
	d.overlay = func(dst *ebiten.Image) {
		f.Draw(d.g.clockMS, nil, nil)
		dst.Fill(color.Black)
		// the scene's camera: twice the canvas size, on the middle of the playing area
		w := dst.Bounds()
		cx, cy := f.Center(float64(art.W)/2-.5, float64(art.H)/2-.5)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterPixelated}
		op.GeoM.Translate(-cx, -cy)
		op.GeoM.Scale(2, 2)
		op.GeoM.Translate(float64(w.Dx())/2, float64(w.Dy())/2)
		dst.DrawImage(f.Canvas, op)
	}
	return nil
}
