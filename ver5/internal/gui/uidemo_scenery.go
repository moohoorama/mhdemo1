package gui

import (
	"os"
	"strconv"
	"strings"
)

func init() { demos["scenery"] = demoScenery }

const (
	sceneryFirstShot = 10
	scenerySpacing   = 30
)

// demoScenery plays a camera tour over the battlefield to compare map art. The env var SCENERY_TOUR
// lists the shots as "u,v,zoom;u,v,zoom;..." (cell coordinates, fractions allowed); shot k is framed at
// frame sceneryFirstShot + k*scenerySpacing, so tools/mapshots/shots.py can save exactly those frames.
func demoScenery(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	g.modals = nil
	type shot struct{ u, v, zoom float64 }
	var tour []shot
	for _, s := range strings.Split(os.Getenv("SCENERY_TOUR"), ";") {
		p := strings.Split(s, ",")
		if len(p) != 3 {
			continue
		}
		u, _ := strconv.ParseFloat(p[0], 64)
		v, _ := strconv.ParseFloat(p[1], 64)
		z, _ := strconv.ParseFloat(p[2], 64)
		tour = append(tour, shot{u, v, z})
	}
	d.step = func(f int) {
		k := (f - sceneryFirstShot) / scenerySpacing
		if f < sceneryFirstShot || (f-sceneryFirstShot)%scenerySpacing != 0 || k >= len(tour) {
			return
		}
		g.camX, g.camY = g.field.Center(tour[k].u, tour[k].v)
		g.setZoomImmediate(tour[k].zoom)
	}
	return nil
}
