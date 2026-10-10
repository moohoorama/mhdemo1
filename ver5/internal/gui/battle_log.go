package gui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"srpg/internal/content"
	"srpg/internal/core"
)

const logLineHeight = 22

func (g *Game) battleLogRect() image.Rectangle {
	h := g.logHeight
	if h == 0 {
		h = 200
	}
	return image.Rect(12, Height-12-max(80, min(440, h)), 492, Height-12)
}

func (g *Game) battleLogVisible() bool {
	if g.S.Screen != "playing" || g.S.Engine == nil {
		return false
	}
	p := g.S.Engine.Observe().Phase
	return p == "battle" || p == "result"
}

// updateBattleLog consumes mouse input before the map camera and selection.
func (g *Game) updateBattleLog() bool {
	if !g.battleLogVisible() || g.overlay != "" || len(g.modals) > 0 || len(g.learn) > 0 || g.input {
		g.logDragging = false
		return false
	}
	x, y := cursorPos()
	r := g.battleLogRect()
	inside := image.Pt(x, y).In(r)
	if inside && y < r.Min.Y+28 && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !g.dragging {
		g.logDragging = true
		g.logDragOffset = y - r.Min.Y
	}
	if g.logDragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.logHeight = max(80, min(440, r.Max.Y-y+g.logDragOffset))
		} else {
			g.logDragging = false
		}
		return true
	}
	if inside && !g.dragging {
		if _, wheel := ebiten.Wheel(); wheel != 0 {
			g.logScroll = max(0, g.logScroll+int(wheel*3))
		}
		return true
	}
	return false
}

func logWindow(lines []string, rows, scroll int) ([]string, int) {
	scroll = max(0, min(scroll, max(0, len(lines)-rows)))
	end := len(lines) - scroll
	return lines[max(0, end-rows):end], scroll
}

func (g *Game) drawBattleLog(dst *ebiten.Image) {
	if !g.battleLogVisible() {
		return
	}
	r := g.battleLogRect()
	g.block(r)
	rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), color.NRGBA{A: 175})
	rect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), 28, color.NRGBA{A: 100})
	strokeRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), 1, color.NRGBA{R: 180, G: 170, B: 150, A: 140})
	g.label(dst, "전투 로그", float64(r.Min.X+12), float64(r.Min.Y+4), 14, gold)
	g.label(dst, "상단 드래그: 높이 조절 · 휠: 이전 기록", float64(r.Min.X+126), float64(r.Min.Y+6), 12, color.NRGBA{R: 210, G: 210, B: 210, A: 255})
	lines := []string{}
	for _, entry := range g.history {
		lines = append(lines, strings.Split(wrap(entry, 32), "\n")...)
	}
	rows := (r.Dy() - 40) / logLineHeight
	lines, g.logScroll = logWindow(lines, rows, g.logScroll)
	if len(lines) == 0 {
		lines = []string{"전투 행동과 결과가 여기에 기록됩니다."}
	}
	y := r.Max.Y - 10 - len(lines)*logLineHeight
	for _, line := range lines {
		g.label(dst, line, float64(r.Min.X+12), float64(y), 14, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
		y += logLineHeight
	}
}

func (g *Game) logCommand(c core.Command, events []core.Event) {
	if len(events) == 0 && c.Kind != "wait" {
		return
	}
	lines := []string{}
	a, t := g.name(c.Actor), g.name(c.Target)
	switch c.Kind {
	case "attack":
		lines = append(lines, fmt.Sprintf("%s → %s: 일반 공격", a, t))
	case "skill":
		lines = append(lines, fmt.Sprintf("%s → %s: %s 사용", a, t, g.skillName(c.Skill)))
	case "item":
		lines = append(lines, fmt.Sprintf("%s → %s: %s 사용", a, t, g.itemName(c.Item)))
	case "wait":
		lines = append(lines, a+": 대기")
	}
	for _, e := range events {
		line, _ := g.eventText(e)
		switch e.Kind {
		case "effect":
			line = g.skillEffectText(c.Skill, e)
		case "item":
			unit := "병력"
			if g.S.Data.Items[c.Item].Effect == content.ItemMP {
				unit = "병법치"
			}
			line = fmt.Sprintf("%s → %s: %s +%d", g.name(e.Actor), g.name(e.Target), unit, e.Amount)
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > 0 {
		g.logScroll = 0
	}
	g.history = append(g.history, lines...)
	if len(g.history) > 300 {
		g.history = g.history[len(g.history)-300:]
	}
}

func (g *Game) skillEffectText(id string, e core.Event) string {
	s := g.S.Data.Skills[id]
	prefix := fmt.Sprintf("%s → %s: ", g.name(e.Actor), g.name(e.Target))
	switch s.Effect {
	case "heal":
		return ""
	case "refill":
		return prefix + fmt.Sprintf("병법치 +%d", e.Amount)
	case "cure":
		return prefix + "해로운 상태 해제"
	case "counter":
		return prefix + "반격 태세 적용"
	case "attack", "defense", "morale", "agility":
		n := g.S.Data.Rules.BuffPercent
		if s.Kind == content.KindStatus {
			n = -n
		}
		return prefix + fmt.Sprintf("%s %+d%%", statusLabels[s.Effect], n)
	case "speed":
		return prefix + fmt.Sprintf("이동력 +%d", int(s.Coeff))
	case "range":
		return prefix + fmt.Sprintf("사거리 +%d", int(s.Coeff))
	}
	if label := statusLabels[s.Effect]; label != "" {
		return prefix + label + " 적용"
	}
	return prefix + "효과 적용"
}
