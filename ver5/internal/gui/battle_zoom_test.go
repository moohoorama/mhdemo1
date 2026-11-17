package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A battle zooms out to 1 wherever the screen fits on the map's canvas (the battle area and the dark outside round
// it), not only where it fits on the battle area; zoomed out, the camera keeps the view on the canvas.
func TestBattleZoomOutToCanvas(t *testing.T) {
	d0, err := NewDemo("../..", DemoOptions{Name: "turn"})
	require.NoError(t, err)
	for _, st := range d0.g.S.Data.Stages {
		t.Run(st.ID, func(t *testing.T) {
			d, err := NewDemo("../..", DemoOptions{Name: "turn", Stage: st.ID})
			require.NoError(t, err)
			g := d.g
			g.S.Settings.UIScale = 100
			g.fit(1280, 850)
			vp, size := g.battleViewport(), g.field.Size()
			if vp.Dx() <= size.X && vp.Dy() <= size.Y {
				assert.Equal(t, 1.0, g.zoomSteps()[0], "the desktop screen fits the canvas at 1")
			}
			z := g.zoomSteps()[0]
			g.setZoomImmediate(z)
			for _, p := range [][2]float64{{-1e5, -1e5}, {1e5, -1e5}, {-1e5, 1e5}, {1e5, 1e5}} {
				x, y := g.clampCam(p[0], p[1])
				hw, hh := float64(vp.Dx())/z/2, float64(vp.Dy())/z/2
				assert.GreaterOrEqual(t, x-hw, -1e-6, "left edge on the canvas")
				assert.LessOrEqual(t, x+hw, float64(size.X)+1e-6, "right edge on the canvas")
				assert.GreaterOrEqual(t, y-hh, -1e-6, "top edge on the canvas")
				assert.LessOrEqual(t, y+hh, float64(size.Y)+1e-6, "bottom edge on the canvas")
			}
		})
	}
}
