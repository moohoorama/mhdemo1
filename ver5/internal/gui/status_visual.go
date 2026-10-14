package gui

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

type statusIcon struct {
	kind string
	down bool
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
		icons = append(icons, statusIcon{kind: kind, down: down})
	}
	return icons
}

var ailmentOrder = []string{"poison", "root", "disarm", "burn", "confusion", "seal"}

func unitStatusKeys(status map[string]core.Status) []string {
	var keys []string
	for _, kind := range ailmentOrder {
		if status[kind].Turns > 0 {
			keys = append(keys, kind)
		}
	}
	for _, icon := range unitStatusIcons(status) {
		direction := "-up"
		if icon.down {
			direction = "-down"
		}
		keys = append(keys, icon.kind+direction)
	}
	return keys
}

// HP and MP bars sit at a fixed spot below the unit; status pixels hang under them.
const unitBarDrop = 4

func (f *Field) drawUnitStatuses(u *unitVis, x, y, _ float64) {
	if u.gone || !u.visible() || u.actor || u.shownHP <= 0 {
		return
	}
	keys := unitStatusKeys(u.status)
	z := f.renderZoom
	if z <= 0 {
		z = 1
	}
	rx, ry := f.renderPoint(x, y+unitBarDrop)
	for i, key := range keys {
		cx := math.Round(rx) + float64(i*10-(len(keys)-1)*5)*z
		drawStatusIcon(f.Canvas, f.A.StatusIcons[key], cx, math.Round(ry)+9*z, z)
	}
}

func drawStatusIcon(dst, icon *ebiten.Image, x, y, z float64) {
	if icon == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	op.GeoM.Scale(z, z)
	op.GeoM.Translate(math.Round(x-4*z), math.Round(y))
	dst.DrawImage(icon, op)
}
