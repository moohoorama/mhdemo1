package gui

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

// TestMeasureFieldCapture renders fields at a fixed camera, zoom and clock (SRPG_CLOCK, ms) into PNGs in
// SRPG_CAPTURE_OUT, so two builds can be compared pixel by pixel (docs/todo/vram.md). SRPG_MAPS picks the maps.
func TestMeasureFieldCapture(t *testing.T) {
	out := os.Getenv("SRPG_CAPTURE_OUT")
	if os.Getenv("SRPG_RENDER_TEST") != "1" || out == "" {
		t.Skip("set SRPG_RENDER_TEST=1 and SRPG_CAPTURE_OUT")
	}
	require.NoError(t, os.MkdirAll(out, 0755))
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	const vw, vh = 1280, 850
	clock := 1234.0
	if v, err := strconv.ParseFloat(os.Getenv("SRPG_CLOCK"), 64); err == nil {
		clock = v
	}
	p := &onceRenderTest{run: func() error {
		for _, id := range measureMaps("B16", "B02", "B09", "B12", "B19", "S_pingyuan", "S_long_field", "S_baimen", "S_pond_deck", "S_hall") {
			art := g.mapArt(id)
			if art == nil {
				return fmt.Errorf("no map %s", id)
			}
			f := NewField(g.assets, art)
			night := id == "B02" || id == "B19"
			corner := f.mapOutline()[0]
			mx, my := f.Center(float64(f.Map.W-1)/2, float64(f.Map.H-1)/2)
			spots := map[string][2]float64{"mid": {mx, my}, "edge": {corner[0], corner[1] + 120}}
			for _, z := range []float64{1, 1.5, 2, 4} {
				for name, at := range spots {
					hw, hh := vw/2/z, vh/2/z
					f.prepareRender(z, at[0]-hw, at[1]-hh, vw, vh)
					f.DrawBounds = image.Rect(int(at[0]-hw)-64, int(at[1]-hh)-96, int(at[0]+hw)+64, int(at[1]+hh)+96)
					f.setNight(night, nil)
					f.Draw(clock, nil, nil)
					b := f.Canvas.Bounds()
					img := image.NewRGBA(b)
					f.Canvas.ReadPixels(img.Pix)
					w, err := os.Create(filepath.Join(out, fmt.Sprintf("%s_z%v_%s.png", id, z, name)))
					if err != nil {
						return err
					}
					if err := png.Encode(w, img); err != nil {
						return err
					}
					w.Close()
				}
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

// TestMeasureUICapture renders the dialogue box, a window and a shou-pattern window at 1, 2 and 2.625 device pixels
// per UI px into SRPG_CAPTURE_OUT. Run it apart from the other measures: each runs the game loop once.
func TestMeasureUICapture(t *testing.T) {
	out := os.Getenv("SRPG_CAPTURE_OUT")
	if os.Getenv("SRPG_RENDER_TEST") != "1" || out == "" {
		t.Skip("set SRPG_RENDER_TEST=1 and SRPG_CAPTURE_OUT")
	}
	require.NoError(t, os.MkdirAll(out, 0755))
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	p := &onceRenderTest{run: func() error {
		for _, px := range []float64{1, 2, 2.625} {
			setUnits(1280*px, 850*px, px, 1)
			dst := ebiten.NewImage(int(1280*px), int(850*px))
			dst.Fill(color.NRGBA{R: 40, G: 40, B: 40, A: 255})
			g.drawBoxBody(dst, 30, 30, 900, 220, [3]float32{.9, .95, 1})
			g.window(dst, image.Rect(30, 300, 600, 820))
			g.titledPattern(dst, image.Rect(650, 300, 1250, 820), "")
			img := image.NewRGBA(dst.Bounds())
			dst.ReadPixels(img.Pix)
			w, err := os.Create(filepath.Join(out, fmt.Sprintf("UI_px%v.png", px)))
			if err != nil {
				return err
			}
			if err := png.Encode(w, img); err != nil {
				return err
			}
			w.Close()
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

// TestMeasureFrameTime prints the median and 90th percentile time of drawing a field, the GPU waited for. SRPG_MAPS
// picks the maps.
func TestMeasureFrameTime(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1")
	}
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	const vw, vh, frames = 1280, 850, 60
	sync := func(im *ebiten.Image) { im.At(0, 0) } // waits for the GPU
	measure := func(label string, draw func(i int), target func() *ebiten.Image) {
		for i := range 10 {
			draw(i)
			sync(target())
		}
		ts := make([]float64, frames)
		for i := range frames {
			t0 := time.Now()
			draw(i)
			sync(target())
			ts[i] = float64(time.Since(t0).Microseconds()) / 1000
		}
		sort.Float64s(ts)
		fmt.Printf("BENCH %-44s median %6.2f ms   p90 %6.2f ms\n", label, ts[frames/2], ts[frames*9/10])
	}
	p := &onceRenderTest{run: func() error {
		for _, c := range measureZoomed(measureMaps("B02", "S_gongsun_camp", "S_long_field"), 1, 2) {
			f := NewField(g.assets, g.mapArt(c.id))
			mx, my := f.Center(float64(f.Map.W-1)/2, float64(f.Map.H-1)/2)
			hw, hh := vw/2/c.z, vh/2/c.z
			prep := func() {
				f.prepareRender(c.z, mx-hw, my-hh, vw, vh)
				f.DrawBounds = image.Rect(int(mx-hw)-64, int(my-hh)-96, int(mx+hw)+64, int(my+hh)+96)
			}
			prep()
			canvas := func() *ebiten.Image { return f.Canvas }
			measure(fmt.Sprintf("%s z%v whole field (now)", c.id, c.z), func(i int) { f.Draw(float64(i)*16.7, nil, nil) }, canvas)
			f.dispose()
		}
		return nil
	}}
	result := make(chan error, 1)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	require.NoError(t, <-result)
	require.NoError(t, p.err)
}

// TestMeasureGPUMemory prints the GPU image memory, as ebiten counts it, once the assets are loaded and with each
// map open at two screen sizes. SRPG_MAPS picks the maps.
func TestMeasureGPUMemory(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1")
	}
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	g := d.g
	mb := func() float64 {
		var info ebiten.DebugInfo
		ebiten.ReadDebugInfo(&info)
		return float64(info.TotalGPUImageMemoryUsageInBytes) / 1048576
	}
	p := &onceRenderTest{run: func() error {
		fmt.Printf("MEM assets loaded            %7.1f MB\n", mb())
		for _, vp := range [][2]int{{1280, 850}, {2560, 1700}} {
			for _, id := range measureMaps("B02", "B16", "S_long_field", "S_cliff_coalition", "S_pond_deck", "S_baimen", "S_coalition_camp") {
				f := NewField(g.assets, g.mapArt(id))
				mx, my := f.Center(float64(f.Map.W-1)/2, float64(f.Map.H-1)/2)
				z := float64(vp[0]) / 1280
				hw, hh := float64(vp[0])/2/z, float64(vp[1])/2/z
				f.prepareRender(z, mx-hw, my-hh, vp[0], vp[1])
				f.DrawBounds = image.Rect(int(mx-hw)-64, int(my-hh)-96, int(mx+hw)+64, int(my+hh)+96)
				f.setNight(id == "B02", nil)
				f.Draw(1234, nil, nil)
				f.Canvas.At(0, 0)
				fmt.Printf("MEM %4dx%-4d %-18s %7.1f MB\n", vp[0], vp[1], id, mb())
				f.dispose()
				f.Canvas.At(0, 0)
			}
		}
		fmt.Printf("MEM after all fields closed  %7.1f MB\n", mb())
		return nil
	}}
	result := make(chan error, 1)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	require.NoError(t, <-result)
	require.NoError(t, p.err)
}

// measureMaps is the maps in SRPG_MAPS (comma separated), else def.
func measureMaps(def ...string) []string {
	if v := os.Getenv("SRPG_MAPS"); v != "" {
		return strings.Split(v, ",")
	}
	return def
}

// measureZoomed pairs each map with each zoom.
func measureZoomed(ids []string, zooms ...float64) []struct {
	id string
	z  float64
} {
	var out []struct {
		id string
		z  float64
	}
	for _, id := range ids {
		for _, z := range zooms {
			out = append(out, struct {
				id string
				z  float64
			}{id, z})
		}
	}
	return out
}

// TestMeasureSurge prints the GPU image memory the flood (assets/graphics/effects/flood/suigong) takes: loaded, and
// at its most while every frame plays in turn, with the time to make each frame's pictures. SRPG_SHEET names a
// sheet PNG (the format before the frames were packed) to be loaded the way that was, for the old figure.
func TestMeasureSurge(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1")
	}
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	mb := func() float64 {
		var info ebiten.DebugInfo
		ebiten.ReadDebugInfo(&info)
		return float64(info.TotalGPUImageMemoryUsageInBytes) / 1048576
	}
	p := &onceRenderTest{run: func() error {
		base := mb()
		fmt.Printf("SURGE base %7.1f MB\n", base)
		if path := os.Getenv("SRPG_SHEET"); path != "" {
			pix, err := loadRGBA(path)
			if err != nil {
				return err
			}
			sheet := ebiten.NewImageFromImage(pix)
			dst := ebiten.NewImage(64, 64)
			dst.DrawImage(sheet.SubImage(image.Rect(0, 0, 64, 64)).(*ebiten.Image), nil)
			dst.At(0, 0)
			fmt.Printf("SURGE sheet %dx%d loaded: +%.1f MB\n", pix.Bounds().Dx(), pix.Bounds().Dy(), mb()-base)
			return nil
		}
		start := time.Now()
		s, err := d.g.assets.Surge("suigong")
		if err != nil {
			return err
		}
		fmt.Printf("SURGE loaded in %v: +%.1f MB\n", time.Since(start).Round(time.Millisecond), mb()-base)
		f := d.g.field
		f.Surge = s
		var peak float64
		var times []float64
		dst := ebiten.NewImage(1280, 850)
		defer dst.Dispose()
		warm := ebiten.NewImage(8, 8)
		dst.DrawImage(warm, nil)
		dst.At(0, 0)
		base = mb()
		for k := range s.ms {
			t0 := time.Now()
			ms := 0.0
			for i := 0; i < k; i++ {
				ms += float64(s.ms[i])
			}
			f.SurgeT = (ms + 1) / 1000
			f.surgeItems()
			for _, b := range s.bands[k] {
				if b.img != nil {
					dst.DrawImage(b.img, nil)
				}
			}
			el := time.Since(t0)
			if el > 50*time.Millisecond {
				fmt.Printf("SURGE slow frame %d: %v with %d bands\n", k, el.Round(time.Millisecond), len(s.bands[k]))
			}
			times = append(times, float64(el.Microseconds())/1000)
			if k%20 == 0 {
				dst.At(0, 0) // makes the GPU finish, so the memory below counts what was drawn; not timed
			}
			if m := mb() - base; m > peak {
				peak = m
			}
		}
		first := times[0]
		times = times[1:]
		sort.Float64s(times)
		fmt.Printf("SURGE %d frames: peak +%.1f MB; making and drawing a frame's pictures median %.2f ms, p90 %.2f ms, max %.2f ms (the first frame %.1f ms)\n",
			len(times)+1, peak, times[len(times)/2], times[len(times)*9/10], times[len(times)-1], first)
		return nil
	}}
	result := make(chan error, 1)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	require.NoError(t, <-result)
	require.NoError(t, p.err)
}
