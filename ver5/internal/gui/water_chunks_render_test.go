package gui

import (
	"fmt"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/scenemap"
)

// A scene map with water bakes its ground once and lays water chunks over it at each frame: the ground then matches
// the flat layer baked whole at that frame, pixel for pixel.
func TestRenderedWaterChunks(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for the native renderer")
	}
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	a := d.g.assets
	p := &onceRenderTest{run: func() error {
		for _, m := range []string{"S_gongsun_camp", "S_pingyuan"} {
			f := NewField(a, a.Maps[m])
			plan, err := scenemap.Build(f.Map.SceneMap, a.SceneRules, scenemap.Options{Decor: true})
			if err != nil {
				return err
			}
			if !assert.NotEmpty(t, f.waterChunks, m) {
				continue
			}
			ox, oy := f.OriginX*K, f.OriginY*K
			water := a.Animations["water"].Frames
			for _, k := range []int{3, 5, 0} {
				want := newLayer(f.worldSize.X, f.worldSize.Y)
				for _, dr := range plan.Flat {
					if dr.Name == "water" {
						dr.Tile = water[k]
					}
					f.drawPlan(want, dr, ox, oy, nil)
				}
				f.playWater(k)
				w, g := make([]byte, 4*f.worldSize.X*f.worldSize.Y), make([]byte, 4*f.worldSize.X*f.worldSize.Y)
				want.ReadPixels(w)
				f.ground[0].ReadPixels(g)
				assert.Equal(t, w, g, fmt.Sprintf("%s at water frame %d", m, k))
				want.Deallocate()
			}
			f.dispose()
		}
		return nil
	}}
	result := make(chan error, 1)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	require.NoError(t, <-result)
	require.NoError(t, p.err)
}

// onceRenderTest runs a check once inside the game loop, where pictures can be read back.
type onceRenderTest struct {
	run  func() error
	err  error
	done bool
}

func (p *onceRenderTest) Layout(int, int) (int, int) { return 64, 64 }
func (p *onceRenderTest) Update() error {
	if p.done {
		return ebiten.Termination
	}
	return nil
}
func (p *onceRenderTest) Draw(*ebiten.Image) {
	if !p.done {
		p.err, p.done = p.run(), true
	}
}
