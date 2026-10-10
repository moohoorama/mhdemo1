package gui

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"srpg/internal/core"
)

type statusIcon struct {
	kind string
	down bool
	tint color.NRGBA
}

var statusIconColors = map[string]color.NRGBA{
	"attack":    {R: 245, G: 75, B: 65, A: 255},
	"defense":   {R: 85, G: 150, B: 255, A: 255},
	"agility":   {R: 255, G: 225, B: 70, A: 255},
	"mind":      {R: 90, G: 225, B: 110, A: 255},
	"morale":    {R: 255, G: 225, B: 70, A: 255},
	"speed":     {R: 245, G: 245, B: 245, A: 255},
	"range":     {R: 255, G: 170, B: 70, A: 255},
	"counter":   {R: 205, G: 140, B: 255, A: 255},
	"eightfold": {R: 80, G: 225, B: 210, A: 255},
	"weak":      {R: 185, G: 185, B: 185, A: 255},
}

func unitStatusIcons(status map[string]core.Status) []statusIcon {
	icons := []statusIcon{}
	for _, id := range []string{"attack", "attack-down", "defense", "defense-down", "agility", "mind", "morale", "speed", "range", "counter", "eightfold", "weak"} {
		s := status[id]
		if s.Turns <= 0 {
			continue
		}
		kind := strings.TrimSuffix(id, "-down")
		down := s.Value < 0 || strings.HasSuffix(id, "-down") || id == "weak"
		if s.Value == 0 && id != "counter" && id != "eightfold" && id != "weak" {
			continue
		}
		icons = append(icons, statusIcon{kind: kind, down: down, tint: statusIconColors[kind]})
	}
	return icons
}

// Status graphics are drawn over scenery so a tree cannot hide a unit's condition.
func (f *Field) drawUnitStatuses(u *unitVis, x, y, ms float64) {
	if len(u.status) == 0 || u.gone || !u.visible() || u.actor {
		return
	}
	rx, ry := f.renderPoint(x, y)
	z := float32(1)
	if f.renderZoom > 0 {
		z = float32(f.renderZoom)
	}
	cx, cy := float32(rx), float32(ry)
	line := func(x1, y1, x2, y2 float32, c color.Color) {
		vector.StrokeLine(f.Canvas, cx+x1*z, cy+y1*z, cx+x2*z, cy+y2*z, max(1, 1.5*z), c, true)
	}
	body := -float32(u.top) * .45
	if u.status["root"].Turns > 0 {
		net := color.NRGBA{R: 225, G: 230, B: 240, A: 225}
		for i := -2; i <= 2; i++ {
			d := float32(i * 5)
			line(-18, body+d, 18, body+d+12, net)
			line(-18, body+d+12, 18, body+d, net)
		}
		line(-18, body-12, 18, body-12, net)
		line(-18, body+22, 18, body+22, net)
	}
	if u.status["disarm"].Turns > 0 {
		blade := color.NRGBA{R: 240, G: 240, B: 245, A: 255}
		line(-9, body+9, 9, body-9, blade)
		line(-10, body+2, -2, body+10, blade)
		line(-12, body+12, -8, body+8, blade)
		red := color.NRGBA{R: 255, G: 65, B: 65, A: 255}
		line(-11, body-11, 11, body+11, red)
		line(-11, body+11, 11, body-11, red)
	}
	if u.status["burn"].Turns > 0 {
		flame := color.NRGBA{R: 255, G: 125, B: 35, A: 240}
		for i := -1; i <= 1; i++ {
			px := float32(i * 12)
			h := float32(9 + 3*math.Sin(ms/130+float64(i)))
			line(px-4, 0, px, -h, flame)
			line(px, -h, px+4, 0, flame)
		}
	}
	if u.status["confusion"].Turns > 0 {
		sy := cy - float32(u.top+9)*z
		for i := 0; i < 3; i++ {
			a := ms/650 + float64(i)*math.Pi*2/3
			vector.FillCircle(f.Canvas, cx+float32(math.Cos(a))*12*z, sy+float32(math.Sin(a))*4*z, max(1, 2*z), color.NRGBA{R: 210, G: 120, B: 255, A: 255}, false)
		}
	}
	if u.status["seal"].Turns > 0 {
		purple := color.NRGBA{R: 205, G: 110, B: 255, A: 250}
		line(12, body-9, 23, body-9, purple)
		line(23, body-9, 23, body+5, purple)
		line(23, body+5, 12, body+5, purple)
		line(12, body+5, 12, body-9, purple)
		line(11, body-10, 24, body+6, purple)
		line(11, body+6, 24, body-10, purple)
	}
	icons := unitStatusIcons(u.status)
	for i, icon := range icons {
		ix := cx + (float32(i)-float32(len(icons)-1)/2)*14*z
		drawStatusIcon(f.Canvas, icon, ix, cy+9*z, z)
	}
}

func drawStatusIcon(dst *ebiten.Image, icon statusIcon, x, y, z float32) {
	vector.FillRect(dst, x-6*z, y-5*z, 12*z, 18*z, color.NRGBA{R: 12, G: 15, B: 20, A: 220}, false)
	line := func(a, b, c, d float32) {
		vector.StrokeLine(dst, x+a*z, y+b*z, x+c*z, y+d*z, max(1, 1.7*z), icon.tint, true)
	}
	tip, tail := float32(-3), float32(5)
	if icon.down {
		tip, tail = 5, -3
	}
	line(0, tail, 0, tip)
	sign := float32(1)
	if icon.down {
		sign = -1
	}
	line(-3, tip+3*sign, 0, tip)
	line(0, tip, 3, tip+3*sign)
	// Small marks distinguish the two yellow attributes and the other stat colors.
	switch icon.kind {
	case "agility":
		line(2, 7, -2, 9)
		line(-2, 9, 2, 9)
		line(2, 9, -2, 12)
	case "morale":
		line(-2, 7, -2, 12)
		line(-2, 7, 3, 8)
		line(3, 8, -2, 9)
	case "defense":
		line(-3, 7, 3, 7)
		line(3, 7, 2, 11)
		line(2, 11, 0, 12)
		line(0, 12, -2, 11)
		line(-2, 11, -3, 7)
	case "mind":
		vector.StrokeCircle(dst, x, y+9*z, 2.5*z, max(1, z), icon.tint, false)
	case "speed":
		line(-3, 9, -1, 11)
		line(1, 7, 3, 9)
	case "attack":
		line(-3, 12, 3, 7)
		line(-3, 9, 0, 12)
	case "range":
		line(-3, 7, 3, 9)
		line(3, 9, -3, 12)
	case "counter":
		line(-3, 9, 3, 9)
		line(-3, 9, -1, 7)
		line(3, 9, 1, 11)
	case "eightfold":
		line(-3, 7, 3, 7)
		line(-3, 11, 3, 11)
		line(-2, 6, -2, 12)
		line(2, 6, 2, 12)
	}
}
