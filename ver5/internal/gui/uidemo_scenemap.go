package gui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
)

func init() { demos["scenemap"] = demoSceneMap }

// demoSceneMap shows one scenario scene map (-stage S_road, S_zhuo, ...) on its own, fitted to the screen,
// as the scene camera sees it: at twice the canvas size around the middle of its playing area, or around game cell
// "u,v" given with -node; -node fit shows the whole map.
func demoSceneMap(d *Demo) error {
	id := d.stage
	art := d.g.mapArt(id)
	if art == nil {
		return fmt.Errorf("no scene map %q (pass one with -stage)", id)
	}
	f := NewField(d.g.assets, art)
	d.overlay = func(dst *ebiten.Image) {
		f.Draw(d.g.clockMS, nil, nil)
		dst.Fill(color.Black)
		// the scene's camera: twice the canvas size, on the middle of the playing area
		w := dst.Bounds()
		u, v := float64(art.W)/2-.5, float64(art.H)/2-.5
		fmt.Sscanf(d.node, "%g,%g", &u, &v)
		cx, cy := f.Center(u, v)
		k := 2.0
		if d.node == "fit" {
			size := f.Size()
			cx, cy = float64(size.X)/2, float64(size.Y)/2
			k = min(float64(w.Dx())/float64(size.X), float64(w.Dy())/float64(size.Y))
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterPixelated}
		op.GeoM.Translate(-cx, -cy)
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(float64(w.Dx())/2, float64(w.Dy())/2)
		dst.DrawImage(f.Canvas, op)
	}
	return nil
}
