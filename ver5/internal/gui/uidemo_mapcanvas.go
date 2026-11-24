package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

func init() { demos["mapcanvas"] = demoMapCanvas }

// demoMapCanvas writes the whole canvas of one scene map (-stage S_zhuo) as <out>/<id>.png, the way the field draws
// it with no people on it: the reference internal/maplight is compared with.
func demoMapCanvas(d *Demo) error {
	id := d.stage
	art := d.g.mapArt(id)
	if art == nil {
		return fmt.Errorf("no scene map %q (pass one with -stage)", id)
	}
	f := NewField(d.g.assets, art)
	laid := false
	d.replace = true
	d.overlay = func(dst *ebiten.Image) {
		if laid || d.err != nil {
			return
		}
		f.Draw(0, nil, nil)
		dir := d.out
		if dir == "" {
			dir = filepath.Join("reports", "uidemo", "mapcanvas")
		}
		if d.err = os.MkdirAll(dir, 0o755); d.err == nil {
			d.err = writeImagePNG(f.Canvas, filepath.Join(dir, id+".png"))
		}
		laid = true
	}
	return nil
}
