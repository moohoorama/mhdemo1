package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/scenes"
)

func init() { demos["scenespots"] = demoSceneSpots }

// spotRender is one scene of a script file (plan/script/chNN.json "Renders") to draw: its shots (the cast as it
// stands once the beats of every node up to After have played; the whole scene when After is empty) and the named
// places, each a few cells, the script's directions refer to as (A), (B) ….
type spotRender struct {
	Scene string
	Shots []struct{ Name, After string }
	Spots []struct {
		Label, Name string
		Cells       [][2]float64
	}
}

var spotColors = []color.NRGBA{{230, 80, 60, 255}, {60, 150, 230, 255}, {70, 190, 90, 255}, {230, 180, 40, 255},
	{180, 90, 220, 255}, {40, 200, 200, 255}}

// demoSceneSpots draws the whole map of each of a script's scenes with its cast and the script's places marked, one
// PNG per shot next to the script (-stage plan/script/ch10.json → plan/script/ch10.<scene>.<shot>.png), for writing
// directions against. It draws the field canvas itself, without the camera, the screen background or the effects.
func demoSceneSpots(d *Demo) error {
	path := d.stage
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var sc struct{ Renders []spotRender }
	if err := json.Unmarshal(raw, &sc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	type job struct {
		r          spotRender
		def        *scenes.Scene
		name, upTo string
	}
	var jobs []job
	for _, r := range sc.Renders {
		def := d.g.scenes.Scenes[r.Scene]
		if def == nil || d.g.mapArt(def.Map) == nil {
			return fmt.Errorf("%s: no scene %q with a known map", path, r.Scene)
		}
		if len(r.Shots) == 0 {
			jobs = append(jobs, job{r, def, "stage", ""})
		}
		for _, sh := range r.Shots {
			jobs = append(jobs, job{r, def, sh.Name, sh.After})
		}
	}
	g := d.g
	base := strings.TrimSuffix(path, ".json")
	next, laid := 0, false
	d.replace = true
	d.overlay = func(dst *ebiten.Image) {
		dst.Fill(color.Black)
		if next >= len(jobs) || d.err != nil {
			return
		}
		j := jobs[next]
		if !laid { // the first frame of a shot lays out the cast; the next draws it
			g.openScene(j.def)
			g.scene.fade = 0
			for _, n := range j.def.Nodes {
				g.settleNode(n)
				if n == j.upTo {
					break
				}
			}
			laid = true
			return
		}
		img := g.spotPicture(j.r)
		if err := writeImagePNG(img, fmt.Sprintf("%s.%s.%s.png", base, j.r.Scene, j.name)); err != nil {
			d.err = err
			return
		}
		op := &ebiten.DrawImageOptions{}
		k := math.Min(float64(dst.Bounds().Dx())/float64(img.Bounds().Dx()), float64(dst.Bounds().Dy())/float64(img.Bounds().Dy()))
		op.GeoM.Scale(k, k)
		dst.DrawImage(img, op)
		next, laid = next+1, false
	}
	return nil
}

func (g *Game) spotPicture(r spotRender) *ebiten.Image {
	s, f := g.scene, g.field
	actors := make([]*unitVis, 0, len(s.actors))
	for _, v := range s.actors {
		actors = append(actors, v)
	}
	f.setGoals(nil)
	f.Draw(g.clockMS, actors, nil)
	b := f.Canvas.Bounds()
	img := ebiten.NewImage(b.Dx(), b.Dy())
	img.Fill(color.NRGBA{24, 22, 20, 255})
	img.DrawImage(f.Canvas, nil)
	for _, p := range s.props {
		pr, ok := g.assets.Props[p.art()]
		u, v, shown := p.at(s)
		if !ok || !shown {
			continue
		}
		x, y := f.Center(u, v)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(math.Round(x-pr.PivotX), math.Round(y-pr.PivotY))
		img.DrawImage(pr.Image, op)
	}

	// crop to the places and the cast on the stage, with a margin, and enlarge it
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	grow := func(u, v float64) {
		x, y := f.Center(u, v)
		x0, y0, x1, y1 = math.Min(x0, x), math.Min(y0, y), math.Max(x1, x), math.Max(y1, y)
	}
	for _, sp := range r.Spots {
		for _, c := range sp.Cells {
			grow(c[0], c[1])
		}
	}
	for _, v := range s.actors {
		if !v.gone {
			grow(v.u, v.v)
		}
	}
	const margin, zoom = 48 * K, 2
	crop := image.Rect(int(x0-margin), int(y0-margin-24*K), int(x1+margin), int(y1+margin)).Intersect(b)
	full := img
	img = ebiten.NewImage(crop.Dx()*zoom, crop.Dy()*zoom)
	zop := &ebiten.DrawImageOptions{}
	zop.GeoM.Scale(zoom, zoom)
	img.DrawImage(full.SubImage(crop).(*ebiten.Image), zop)
	at := func(u, v float64) (float64, float64) {
		x, y := f.Center(u, v)
		return (x - float64(crop.Min.X)) * zoom, (y - float64(crop.Min.Y)) * zoom
	}
	const Z = K * zoom

	label := g.serif(34, 800)
	small := g.serif(22, 700)
	for i, sp := range r.Spots {
		c := spotColors[i%len(spotColors)]
		fill := c
		fill.A = 38
		var sx, sy, top float64 = 0, 0, math.Inf(1)
		for _, cell := range sp.Cells {
			x, y := at(cell[0], cell[1])
			var p vector.Path
			p.MoveTo(float32(x), float32(y-8*Z))
			p.LineTo(float32(x+16*Z), float32(y))
			p.LineTo(float32(x), float32(y+8*Z))
			p.LineTo(float32(x-16*Z), float32(y))
			p.Close()
			vector.FillPath(img, &p, nil, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(fill)})
			vector.StrokePath(img, &p, &vector.StrokeOptions{Width: 2}, &vector.DrawPathOptions{AntiAlias: true, ColorScale: colorScale(c)})
			sx, sy, top = sx+x, sy+y, math.Min(top, y)
		}
		n := float64(len(sp.Cells))
		if n == 0 {
			continue
		}
		cx, cy := sx/n, math.Min(sy/n, top+4*Z)
		vector.FillCircle(img, float32(cx), float32(cy), 24, c, true)
		vector.StrokeCircle(img, float32(cx), float32(cy), 24, 3, color.White, true)
		drawCentered(img, sp.Label, label, cx, cy, color.White)
	}

	names := make([]string, 0, len(s.actors))
	for key, v := range s.actors {
		if !v.gone && !scenes.Extra(key) {
			names = append(names, key)
		}
	}
	sort.Strings(names)
	for _, key := range names {
		v := s.actors[key]
		x, y := at(v.u, v.v)
		y += 10*Z - v.lift*zoom
		w, h := text.Measure(key, small, 0)
		vector.FillRect(img, float32(x-w/2-4), float32(y-h/2-1), float32(w+8), float32(h+2), color.NRGBA{0, 0, 0, 170}, false)
		drawCentered(img, key, small, x, y, color.NRGBA{250, 244, 230, 255})
	}

	// legend in the top-left corner
	y := 14.0
	for i, sp := range r.Spots {
		c := spotColors[i%len(spotColors)]
		line := sp.Label + "  " + sp.Name
		w, h := text.Measure(line, label, 0)
		vector.FillRect(img, 10, float32(y-4), float32(w+50), float32(h+8), color.NRGBA{0, 0, 0, 170}, false)
		vector.FillCircle(img, 30, float32(y+h/2), 16, c, true)
		drawCentered(img, sp.Label, small, 28, y+h/2, color.White)
		op := &text.DrawOptions{}
		op.GeoM.Translate(54, y)
		text.Draw(img, sp.Name, label, op)
		y += h + 14
	}
	return img
}

func colorScale(c color.NRGBA) ebiten.ColorScale {
	var cs ebiten.ColorScale
	cs.ScaleWithColor(c)
	return cs
}

func drawCentered(dst *ebiten.Image, s string, f *text.GoTextFace, x, y float64, c color.Color) {
	w, h := text.Measure(s, f, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(x-w/2, y-h/2)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, f, op)
}

func writeImagePNG(img *ebiten.Image, path string) error {
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	img.ReadPixels(rgba.Pix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, rgba); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
