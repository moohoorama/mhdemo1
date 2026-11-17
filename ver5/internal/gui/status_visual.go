package gui

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

type statusIcon struct {
	kind string
	down bool
}

func unitStatusIcons(status map[string]core.Status) []statusIcon {
	icons := []statusIcon{}
	for _, id := range []string{"attack", "defense", "agility", "mind", "morale", "speed", "range", "counter", "weak"} {
		s := status[id]
		if s.Turns <= 0 {
			continue
		}
		down := s.Value < 0 || id == "weak"
		if s.Value == 0 && id != "counter" && id != "weak" {
			continue
		}
		icons = append(icons, statusIcon{kind: id, down: down})
	}
	if status["hold_ground"].Turns > 0 && status["defense"].Value <= 0 { // 진지고수 shows as a defense raise
		icons = append(icons, statusIcon{kind: "defense"})
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

// statusMarks are the statuses whose marks turn round a unit's head (status_mark_<status>, baked by
// tools/spellsim_confuse.py), in the order they take places on the ring.
var statusMarks = []string{"confusion", "disarm", "seal", "root", "poison"}

const (
	statusMarkRadius = 13.0 // the ring's radius in map units, round the top of the sprite
	statusMarkTurn   = 1.2  // seconds a mark takes round the ring
)

// drawStatusMarks draws the marks of the unit's statuses turning round its head, upright, on one ring: four of a
// lone status, two of each of two or three, one of each of four or more, taking turns round it. back draws the half behind the unit (before
// its sprite), otherwise the half in front (after it).
func (f *Field) drawStatusMarks(u *unitVis, x, y float64, back bool) {
	if u.gone || u.shownHP <= 0 {
		return
	}
	var kinds []string
	for _, k := range statusMarks {
		if u.status[k].Turns > 0 {
			kinds = append(kinds, k)
		}
	}
	if len(kinds) == 0 {
		return
	}
	n := 4
	switch k := len(kinds); {
	case k >= 4: // one each: two of every one crowded the ring
		n = k
	case k > 1:
		n = 2 * k
	}
	turn := f.clockMS / 1000 / statusMarkTurn
	for i := range n {
		a := 2 * math.Pi * (turn + float64(i)/float64(n))
		wx, wy := math.Cos(a)*statusMarkRadius, math.Sin(a)*statusMarkRadius
		if (wx+wy < 0) != back {
			continue
		}
		sh, ok := f.A.lookupEffect("status_mark_" + kinds[i%len(kinds)])
		if !ok || len(sh.Frames) == 0 {
			continue
		}
		an := sh.anchor(0)
		f.blit(f.Canvas, Sprite{sh.Frames[0], an[0], an[1]}, math.Round(x+u.shakeOffset()+wx-wy), math.Round(y-u.top+(wx+wy)/2), nil)
	}
}
