package gui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/sprite"
)

// temporary: draft of option C — each sheet split into a fixed layer and a grey faction layer
// (shade 1·2·3 as grey levels) coloured at draw time by a small shader — and of officer
// colours (a little brighter, hue turned a little) against the soldiers' faction colours.

const factionShaderSrc = `//kage:unit pixels
package main

var Light vec3
var Mid vec3
var Dark vec3

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	c := imageSrc0At(src)
	if c.a == 0 {
		return vec4(0)
	}
	g := c.r / c.a
	col := Dark
	if g > 0.8 {
		col = Light
	} else if g > 0.45 {
		col = Mid
	}
	return vec4(col*c.a, c.a)
}
`

var factionShader *ebiten.Shader

// layered is a sheet split for option C.
type layered struct {
	s           *sprite.Sheet
	fixed, grey *ebiten.Image
}

func splitSheet(s *sprite.Sheet) *layered {
	mark := func(a, b, c color.RGBA) *image.RGBA {
		return s.Render(sprite.Palette{"faction": {a, b, c}})
	}
	m1 := mark(color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255}, color.RGBA{0, 0, 255, 255})
	m2 := mark(color.RGBA{0, 0, 0, 255}, color.RGBA{0, 0, 0, 255}, color.RGBA{0, 0, 0, 255})
	fixed := s.Render(sprite.Palette{})
	grey := image.NewRGBA(fixed.Rect)
	for i := 0; i < len(fixed.Pix); i += 4 {
		if m1.Pix[i] == m2.Pix[i] && m1.Pix[i+1] == m2.Pix[i+1] && m1.Pix[i+2] == m2.Pix[i+2] {
			continue
		}
		a := m1.Pix[i+3]
		v := 0.3
		switch {
		case m1.Pix[i] > 0:
			v = 1
		case m1.Pix[i+1] > 0:
			v = 0.6
		}
		g := uint8(v * float64(a))
		grey.Pix[i], grey.Pix[i+1], grey.Pix[i+2], grey.Pix[i+3] = g, g, g, a
		fixed.Pix[i], fixed.Pix[i+1], fixed.Pix[i+2], fixed.Pix[i+3] = 0, 0, 0, 0
	}
	return &layered{s: s, fixed: ebiten.NewImageFromImage(fixed), grey: ebiten.NewImageFromImage(grey)}
}

func (l *layered) cell(dir string) image.Rectangle {
	w, h := l.s.Cell[0], l.s.Cell[1]
	row := 0
	for i, d := range l.s.Rows {
		if d == dir {
			row = i
		}
	}
	col := l.s.Animations["idle"].FirstColumn
	return image.Rect(col*w, row*h, (col+1)*w, (row+1)*h)
}

// frame draws the idle frame facing dir with the faction layer in shades c, at 1x.
func (l *layered) frame(dir string, c [3]color.RGBA) *ebiten.Image {
	r := l.cell(dir)
	out := ebiten.NewImage(r.Dx(), r.Dy())
	out.DrawImage(l.fixed.SubImage(r).(*ebiten.Image), nil)
	v := func(c color.RGBA) []float32 { return []float32{float32(c.R) / 255, float32(c.G) / 255, float32(c.B) / 255} }
	op := &ebiten.DrawRectShaderOptions{Images: [4]*ebiten.Image{l.grey.SubImage(r).(*ebiten.Image)},
		Uniforms: map[string]any{"Light": v(c[0]), "Mid": v(c[1]), "Dark": v(c[2])}}
	out.DrawRectShader(r.Dx(), r.Dy(), factionShader, op)
	return out
}

// officerShades turns a faction's shades into its officers': brighter by up (a share of the way to
// full value, saturation eased so a colour already at full value still lightens) and hue turned by deg.
func officerShades(cs []color.RGBA, up, deg float64) [3]color.RGBA {
	var out [3]color.RGBA
	for i := range out {
		h, s, v := rgbToHSV(cs[i])
		h = math.Mod(h+deg+360, 360)
		nv := v + (1-v)*up + v*up*0.5
		if up < 0 {
			nv = v * (1 + up)
		}
		if nv > 1 {
			s *= 1 - (nv-1)*1.5
			nv = 1
		}
		if up > 0 {
			s *= 1 - up*0.3
		}
		out[i] = hsvToRGB(h, math.Max(0, s), nv)
	}
	return out
}

func rgbToHSV(c color.RGBA) (h, s, v float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	v, d := mx, mx-mn
	if mx > 0 {
		s = d / mx
	}
	switch {
	case d == 0:
	case mx == r:
		h = 60 * math.Mod((g-b)/d, 6)
	case mx == g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}
	return math.Mod(h+360, 360), s, v
}

func hsvToRGB(h, s, v float64) color.RGBA {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch int(h/60) % 6 {
	case 0:
		r, g, b = c, x, 0
	case 1:
		r, g, b = x, c, 0
	case 2:
		r, g, b = 0, c, x
	case 3:
		r, g, b = 0, x, c
	case 4:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	u := func(f float64) uint8 { return uint8(math.Round((f + m) * 255)) }
	return color.RGBA{u(r), u(g), u(b), 255}
}

// officer variants shown next to the soldiers' colours.
var officerVariants = []struct {
	name    string
	up, deg float64
}{
	{"병사 안A (어둡게 10%, 색조 -10도)", -0.10, -10},
	{"병사 안B (어둡게 20%, 색조 -10도)", -0.20, -10},
	{"병사 안C (어둡게 30%, 색조 -10도)", -0.30, -10},
	{"병사 안D (어둡게 20%, 색조 그대로)", -0.20, 0},
}

var demoFactions = []string{"shu", "wei", "wu", "turban", "dong", "yuan", "gongsun", "han"}

func hexOf(c color.RGBA) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func init() {
	demos["zz-faction"] = func(d *Demo) error {
		g := d.g
		if factionShader == nil {
			s, err := ebiten.NewShader([]byte(factionShaderSrc))
			if err != nil {
				return err
			}
			factionShader = s
		}
		ids := []string{"light_infantry", "light_cavalry"}
		names := []string{"경보병", "경기병"}
		sheets := map[string]*layered{}
		for _, id := range ids {
			sheets[id] = splitSheet(g.assets.Lib.Sheet(id))
		}
		checked := false
		d.replace = true
		d.overlay = func(dst *ebiten.Image) {
			dst.Fill(color.NRGBA{R: 96, G: 124, B: 66, A: 255})
			const sc, colW, rowH, x0, y0 = 1.5, 120.0, 96.0, 78.0, 74.0
			write(dst, "C안: 진영 겹을 회색으로 분리 → 셰이더가 진영색 3단으로 칠함 (병종 시트는 GPU에 1장)", g.serif(17, 700), 16, 8, color.White)
			for j := range ids {
				labs := []string{"장수 (현재 그대로)"}
				for _, v := range officerVariants {
					labs = append(labs, v.name[len("병사 "):])
				}
				for k, lab := range labs {
					x := x0 + float64(j*5+k)*colW
					write(dst, names[j], g.serif(13, 700), x+6, 36, color.White)
					write(dst, lab, g.serif(9, 500), x+4, 54, color.NRGBA{R: 230, G: 230, B: 210, A: 255})
				}
			}
			for i, f := range demoFactions {
				fc := colorsOf(g.S.Data.Factions[f].Colors)
				y := y0 + float64(i)*rowH
				write(dst, g.S.Data.Factions[f].Name, g.serif(14, 700), 10, y+38, color.White)
				shades := [][3]color.RGBA{{fc[0], fc[1], fc[2]}}
				for _, v := range officerVariants {
					shades = append(shades, officerShades(fc, v.up, v.deg))
				}
				for j, id := range ids {
					l := sheets[id]
					for k, c := range shades {
						img := l.frame("SE", c)
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Scale(sc, sc)
						op.GeoM.Translate(x0+float64(j*5+k)*colW+colW/2-float64(l.s.Pivot[0])*sc, y+rowH-8-float64(l.s.Pivot[1])*sc)
						dst.DrawImage(img, op)
						if j == 0 {
							for n, cc := range c {
								rect(dst, float32(x0+float64(k)*colW+6+float64(n)*9), float32(y+rowH-12), 8, 8, color.NRGBA{R: cc.R, G: cc.G, B: cc.B, A: 255})
							}
						}
					}
				}
				if !checked && i == 0 {
					checkLayers(g, sheets, fc)
				}
			}
			checked = true
		}
		return nil
	}

	demos["zz-factionfield"] = func(d *Demo) error {
		if err := d.battle(d.stage); err != nil {
			return err
		}
		g := d.g
		orig := map[string][]string{}
		for id, f := range g.S.Data.Factions {
			orig[id] = f.Colors
		}
		officer := func(def string) bool { return !g.S.Data.Characters[def].Template }
		variant := -1
		apply := func(v int) {
			for _, u := range g.S.View().UnitViews {
				vis := g.vis[u.ID]
				if vis == nil {
					continue
				}
				if v >= 0 && !officer(u.Def) && len(orig[u.Color]) > 0 {
					f := g.S.Data.Factions[u.Color]
					sh := officerShades(colorsOf(orig[u.Color]), officerVariants[v].up, officerVariants[v].deg)
					f.Colors = []string{hexOf(sh[0]), hexOf(sh[1]), hexOf(sh[2])}
					g.S.Data.Factions[u.Color] = f
				}
				vis.art = g.battleClassArt(u.Def, u.Class, u.Weapon, u.Mount, u.Color)
				for id, cs := range orig {
					f := g.S.Data.Factions[id]
					f.Colors = cs
					g.S.Data.Factions[id] = f
				}
			}
		}
		focus := os.Getenv("FOCUS")
		for _, u := range g.S.View().UnitViews {
			if u.Def == focus {
				g.camX, g.camY = g.field.Center(float64(u.X)+.5, float64(u.Y)+.5)
			}
		}
		labels := []string{"현재 (장수·병사 같은 색)"}
		for _, v := range officerVariants {
			labels = append(labels, v.name)
		}
		d.step = func(frame int) {
			if v := frame/10 - 1; v != variant && v < len(officerVariants) {
				variant = v
				apply(v)
			}
		}
		d.overlay = func(dst *ebiten.Image) {
			rect(dst, 20, 20, 520, 56, color.NRGBA{R: 16, G: 12, B: 10, A: 220})
			write(dst, labels[variant+1], g.serif(28, 700), 36, 28, color.White)
		}
		return nil
	}
}

// checkLayers compares option C's frames with the sheets the game bakes today, in the same colours.
func checkLayers(g *Game, sheets map[string]*layered, fc []color.RGBA) {
	for id, l := range sheets {
		r := l.cell("SE")
		got := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
		l.frame("SE", [3]color.RGBA{fc[0], fc[1], fc[2]}).ReadPixels(got.Pix)
		want := l.s.Render(sprite.Palette{"faction": fc})
		diff := 0
		for y := 0; y < r.Dy(); y++ {
			for x := 0; x < r.Dx(); x++ {
				a, b := got.RGBAAt(x, y), want.RGBAAt(r.Min.X+x, r.Min.Y+y)
				dd := func(p, q uint8) bool { return int(p)-int(q) > 1 || int(q)-int(p) > 1 }
				if dd(a.R, b.R) || dd(a.G, b.G) || dd(a.B, b.B) || dd(a.A, b.A) {
					diff++
				}
			}
		}
		fmt.Fprintf(os.Stderr, "check %s: %d of %d pixels differ from the baked sheet\n", id, diff, r.Dx()*r.Dy())
	}
}

// hanGreys are grey candidates for the imperial (han) colours, after the current gold.
var hanGreys = []struct {
	name   string
	colors []string
}{
	{"한 황실 현재 (금색)", nil},
	{"회색 1: 차가운 회색", []string{"#b4bac2", "#8c939c", "#5c636c"}},
	{"회색 2: 따뜻한 회색", []string{"#bdb5aa", "#948b7f", "#625a50"}},
	{"회색 3: 진회색", []string{"#9aa0a6", "#6e747b", "#464b52"}},
}

func init() {
	demos["zz-hangrey"] = func(d *Demo) error {
		if err := d.battle(d.stage); err != nil {
			return err
		}
		g := d.g
		orig := g.S.Data.Factions["han"].Colors
		focus := os.Getenv("FOCUS")
		for _, u := range g.S.View().UnitViews {
			if u.Def == focus {
				g.camX, g.camY = g.field.Center(float64(u.X)+.5, float64(u.Y)+.5)
			}
		}
		cur := -1
		d.step = func(frame int) {
			v := min(frame/10, len(hanGreys)-1)
			if v == cur {
				return
			}
			cur = v
			f := g.S.Data.Factions["han"]
			f.Colors = orig
			if hanGreys[v].colors != nil {
				f.Colors = hanGreys[v].colors
			}
			g.S.Data.Factions["han"] = f
			for _, u := range g.S.View().UnitViews {
				if vis := g.vis[u.ID]; vis != nil {
					vis.art = g.battleClassArt(u.Def, u.Class, u.Weapon, u.Mount, u.Color)
				}
			}
		}
		d.overlay = func(dst *ebiten.Image) {
			rect(dst, 20, 20, 520, 56, color.NRGBA{R: 16, G: 12, B: 10, A: 220})
			write(dst, hanGreys[max(cur, 0)].name, g.serif(28, 700), 36, 28, color.White)
		}
		return nil
	}
}
