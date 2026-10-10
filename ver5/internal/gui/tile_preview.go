package gui

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// RenderTilePreview captures a rendering-only prototype without changing runtime assets.
func RenderTilePreview(out string) error {
	d, err := content.Load("assets/data")
	if err != nil {
		return err
	}
	s := &session.Session{Data: d, Engine: core.New(d, 1)}
	for n := 0; s.Engine.Observe().Phase != "battle"; n++ {
		if n == 100 {
			return fmt.Errorf("first battle not reached")
		}
		o := s.Engine.Observe()
		c := core.Command{Kind: "next"}
		switch o.Dialogue.Kind {
		case "choice":
			keys := sorted(o.Dialogue.Choices)
			if len(keys) == 0 {
				return fmt.Errorf("empty scenario choice")
			}
			c = core.Command{Kind: "choose", Option: keys[0]}
		case "preparation":
			c.Kind = "start"
		}
		if _, err := s.Engine.Apply(c); err != nil {
			return err
		}
	}
	g, err := New(s, "assets/fonts/NotoSansKR.ttf", 1, "")
	if err != nil {
		return err
	}
	defer g.Close()
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	g.ensureField(s.Engine.Observe())
	p := &tilePreview{g: g, out: out}
	ebiten.SetWindowSize(Width, Height)
	ebiten.SetWindowTitle("1스테이지 타일 배율 비교")
	if err := ebiten.RunGame(p); err != nil {
		return err
	}
	return p.err
}

type tilePreview struct {
	g    *Game
	out  string
	done bool
	err  error
}

func (p *tilePreview) Layout(int, int) (int, int) { return Width, Height }
func (p *tilePreview) Update() error {
	if p.done {
		return ebiten.Termination
	}
	return nil
}

func (p *tilePreview) Draw(screen *ebiten.Image) {
	if p.done {
		return
	}
	g, f := p.g, p.g.field
	units := make([]*unitVis, 0, len(g.vis))
	for _, id := range sorted(g.vis) {
		units = append(units, g.vis[id])
	}
	before, after := ebiten.NewImage(Width, Height), ebiten.NewImage(Width, Height)
	f.Draw(0, units, nil)
	p.panel(before, "기존 · 타일/구조물 2배", "대타일 64×32 · 유닛과 숲 나무 원본 배율")
	p.drawNative(units)
	p.panel(after, "시제품 · 타일/구조물 원본 배율", "대타일 크기 유지 · 소타일 2×2 · 유닛/숲 나무 크기 유지")
	comparison := ebiten.NewImage(Width*2, Height)
	comparison.DrawImage(before, nil)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(Width, 0)
	comparison.DrawImage(after, op)
	for _, entry := range []struct {
		name string
		img  *ebiten.Image
	}{{"B01-before.png", before}, {"B01-prototype.png", after}, {"B01-comparison.png", comparison}} {
		if err := saveTilePreview(filepath.Join(p.out, entry.name), entry.img); err != nil {
			p.err = err
			break
		}
	}
	screen.DrawImage(after, nil)
	p.done = true
}

func (p *tilePreview) panel(dst *ebiten.Image, title, subtitle string) {
	g, f := p.g, p.g.field
	dst.Fill(color.RGBA{24, 30, 36, 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(Width-f.Canvas.Bounds().Dx())/2, 135)
	dst.DrawImage(f.Canvas, op)
	g.label(dst, "1스테이지: "+f.Map.Name, 34, 20, 22, gold)
	g.label(dst, title, 34, 58, 26, ink)
	g.label(dst, subtitle, 34, 99, 17, muted)
	g.label(dst, "청록: 게임상의 대타일 / 노랑: 내부 소타일 (시제품)", 34, Height-42, 17, muted)
	stroke := func(u, v, w, h float64, c color.Color) {
		x, y := f.Center(u, v)
		x += float64(Width-f.Canvas.Bounds().Dx()) / 2
		y += 135
		points := [][2]float64{{x, y - h}, {x + w, y}, {x, y + h}, {x - w, y}}
		for i, a := range points {
			b := points[(i+1)%4]
			vector.StrokeLine(dst, float32(a[0]), float32(a[1]), float32(b[0]), float32(b[1]), 1, c, false)
		}
	}
	stroke(8, 4, 32, 16, color.RGBA{74, 231, 232, 255})
	if title != "기존 · 타일/구조물 2배" {
		stroke(7.75, 3.75, 16, 8, color.RGBA{255, 218, 89, 255})
	}
}

func saveTilePreview(path string, src *ebiten.Image) error {
	im := image.NewRGBA(src.Bounds())
	src.ReadPixels(im.Pix)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = png.Encode(f, im)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// drawNative keeps the existing cell projection, but stamps native-size map art.
// Ground masks follow the existing terrain exporter on a grid with twice as many cells.
func (p *tilePreview) drawNative(units []*unitVis) {
	f, a := p.g.field, p.g.assets
	c := f.Canvas
	c.Fill(color.RGBA{30, 38, 46, 255})
	for _, t := range previewGround(f.Map) {
		for _, id := range t.Layers {
			if id == 0 {
				id = a.Animations["water"].Frames[0]
			}
			f.blit(c, a.Tiles[id], f.OriginX*K+float64(t.X), f.OriginY*K+float64(t.Y), nil)
		}
	}
	// A doubled logical grid lets wall masks connect the four small wall tiles.
	fine := &Field{Map: &MapArt{W: f.Map.W * 2, H: f.Map.H * 2}}
	for _, row := range f.Map.Tiles {
		r := ""
		for _, t := range row {
			r += string(t) + string(t)
		}
		fine.Map.Tiles = append(fine.Map.Tiles, r, r)
	}
	items := []drawItem{}
	add := func(s Sprite, x, y float64) {
		b := s.Image.Bounds()
		r := image.Rect(int(x-s.PivotX), int(y-s.PivotY), int(x-s.PivotX)+b.Dx(), int(y-s.PivotY)+b.Dy())
		items = append(items, drawItem{y: y, rect: r, draw: func() { f.blit(c, s, x, y, nil) }})
	}
	for _, d := range f.Map.Decorations {
		add(a.Tiles[a.TileFrame(d.Animation, float64(d.Phase)*125)], (f.OriginX+float64(d.X))*K, (f.OriginY+float64(d.Y))*K)
	}
	for v, row := range f.Map.Tiles {
		for u, t := range row {
			if t == 'f' {
				x, y := f.Center(float64(u), float64(v))
				add(a.Props["forest_"+string(rune('0'+(u*7+v*3)%4))].Sprite, x, y)
				continue
			}
			for sv := 0; sv < 2; sv++ {
				for su := 0; su < 2; su++ {
					x, y := f.Center(float64(u)+(float64(su)-.5)/2, float64(v)+(float64(sv)-.5)/2)
					switch t {
					case 'i':
						f.blit(c, a.Structures["floor"], x, y, nil)
					case 'v':
						add(a.Structures["village"], x, y)
					case 's':
						add(a.Structures["mountain"], x, y)
					case 'c':
						add(a.Structures[wallName(fine, u*2+su, v*2+sv)], x, y)
					}
				}
			}
			if t == 'k' || t == 'g' || t == 'b' {
				name := []string{"tent", "tent_round"}[(u+v)%2]
				if t == 'g' {
					name = gateName(f, u, v)
				} else if t == 'b' {
					name = f.bridgeName(u, v)
				}
				x, y := f.Center(float64(u), float64(v))
				s := a.Props[name].Sprite
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(.5, .5)
				op.GeoM.Translate(x-s.PivotX*.5, y-s.PivotY*.5)
				items = append(items, drawItem{y: y, draw: func() { c.DrawImage(s.Image, op) }})
			}
		}
	}
	f.clip.Clear()
	f.clip.DrawImage(c, nil)
	f.shadows.Clear()
	for _, original := range units {
		u := *original
		u.lift /= K
		x, y := f.Center(u.u, u.v)
		shadow := Sprite{u.art.Shadow(u.dir, u.anim, 0), u.art.ShadowPivot[0], u.art.ShadowPivot[1]}
		f.blit(f.shadows, shadow, x, y-u.lift, &ebiten.DrawImageOptions{Blend: maxBlend})
		items = append(items, drawItem{y: y, kind: 1, draw: func() { f.drawUnit(&u, x, y-u.lift, false) }})
	}
	f.shadows.DrawImage(f.clip, &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn})
	c.DrawImage(f.shadows, nil)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].y == items[j].y {
			return items[i].kind < items[j].kind
		}
		return items[i].y < items[j].y
	})
	for _, it := range items {
		it.draw()
	}
}

func previewGround(m *MapArt) []GroundTile {
	// Same river/grass/wasteland mapping as ver4/tools/build_assets.py.
	ground := map[byte]int{'.': 1, 'd': 2, '~': 0, 'f': 1, 's': 2, 'v': 2, 'c': 2, 'i': 2, 'b': 0, 'g': 2, 'k': 1}
	vertex := func(sx, sy, kind int) bool {
		for y := (sy - 1) / 2; y <= sy/2; y++ {
			for x := (sx - 1) / 2; x <= sx/2; x++ {
				if x >= 0 && y >= 0 && x < m.W*2 && y < m.H*2 && ground[m.Tiles[y/2][x/2]] >= kind {
					return true
				}
			}
		}
		return false
	}
	mask := func(x, y, kind int) int {
		value := 0
		for bit, p := range [][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
			if vertex(x+p[0], y+p[1], kind) {
				value |= 1 << bit
			}
		}
		return value
	}
	tiles := make([]GroundTile, 0, m.W*m.H*16)
	for y := 0; y < m.H*2; y++ {
		for x := 0; x < m.W*2; x++ {
			for j := 0; j < 2; j++ {
				for i := 0; i < 2; i++ {
					dry, soil := mask(2*x+i, 2*y+j, 1), mask(2*x+i, 2*y+j, 2)
					layers := []int{8}
					if soil != 15 {
						layers = []int{23}
						if dry != 15 {
							layers = []int{0}
							if dry != 0 {
								layers = append(layers, 23+dry)
							}
						}
						if soil != 0 {
							layers = append(layers, 8+soil)
						}
					}
					tiles = append(tiles, GroundTile{X: (x-y)*16 + (i-j)*8, Y: (x+y)*8 + (i+j)*4, Layers: layers})
				}
			}
		}
	}
	return tiles
}
