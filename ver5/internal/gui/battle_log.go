package gui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
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
		g.logThumbDragging = false
		return false
	}
	x, y := cursorPos()
	r := g.battleLogRect()
	inside := image.Pt(x, y).In(r)
	if inside && y < r.Min.Y+28 && mouseJustPressed(ebiten.MouseButtonLeft) && !g.dragging {
		g.logDragging = true
		g.logDragOffset = y - r.Min.Y
	}
	if g.logDragging {
		if mousePressed(ebiten.MouseButtonLeft) {
			g.logHeight = max(80, min(440, r.Max.Y-y+g.logDragOffset))
		} else {
			g.logDragging = false
		}
		return true
	}
	lines := g.battleLogLines()
	rows := (r.Dy() - 40) / logLineHeight
	maxScroll := max(0, len(lines)-rows)
	track, thumb := logScrollbar(r, len(lines), rows, g.logScroll)
	if inside && maxScroll > 0 && image.Pt(x, y).In(track) && mouseJustPressed(ebiten.MouseButtonLeft) && !g.dragging {
		g.logThumbDragging = true
		g.logThumbOffset = thumb.Dy() / 2
		if image.Pt(x, y).In(thumb) {
			g.logThumbOffset = y - thumb.Min.Y
		}
	}
	if g.logThumbDragging {
		if mousePressed(ebiten.MouseButtonLeft) {
			g.logScroll = logScrollAt(y-g.logThumbOffset, track, thumb.Dy(), maxScroll)
		} else {
			g.logThumbDragging = false
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
	g.label(dst, "상단: 높이 조절 · 스크롤바: 이전 기록", float64(r.Min.X+126), float64(r.Min.Y+6), 12, color.NRGBA{R: 210, G: 210, B: 210, A: 255})
	lines := g.battleLogLines()
	rows := (r.Dy() - 40) / logLineHeight
	track, thumb := logScrollbar(r, len(lines), rows, g.logScroll)
	rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 150})
	rect(dst, float32(thumb.Min.X), float32(thumb.Min.Y), float32(thumb.Dx()), float32(thumb.Dy()), color.NRGBA{R: 185, G: 170, B: 135, A: 220})
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
	g.syncBattleLogStage()
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
	g.logPending = append(g.logPending, lines...)
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
	case "attack", "defense", "mind", "morale", "agility":
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

func (g *Game) resetBattleLog() {
	g.history = nil
	g.logPending = nil
	g.logStage = ""
	g.logScroll = 0
	g.logDragging = false
	g.logThumbDragging = false
}

func (g *Game) bindBattleLogStage(id string) {
	if id != g.logStage {
		g.resetBattleLog()
		g.logStage = id
	}
}

func (g *Game) syncBattleLogStage() {
	if g.S != nil && g.S.Engine != nil {
		o := g.S.Engine.Observe()
		if o.Map != nil {
			g.bindBattleLogStage(o.Map.ID)
		} else if g.logStage != "" {
			g.resetBattleLog()
		}
	}
}

func (g *Game) publishBattleLog() {
	if len(g.logPending) == 0 {
		return
	}
	entry := g.logPending[0]
	g.logPending[0] = ""
	g.logPending = g.logPending[1:]
	if g.logScroll > 0 {
		g.logScroll += len(strings.Split(wrap(entry, 32), "\n"))
	}
	g.history = append(g.history, entry)
	if len(g.history) > 300 {
		g.history = g.history[len(g.history)-300:]
	}
}

func (g *Game) battleLogLines() []string {
	lines := []string{}
	for _, entry := range g.history {
		lines = append(lines, strings.Split(wrap(entry, 32), "\n")...)
	}
	return lines
}

func logScrollbar(r image.Rectangle, total, rows, scroll int) (image.Rectangle, image.Rectangle) {
	track := image.Rect(r.Max.X-18, r.Min.Y+34, r.Max.X-6, r.Max.Y-10)
	height := track.Dy()
	if total > rows {
		height = max(18, track.Dy()*rows/total)
	}
	offset := 0
	limit := max(0, total-rows)
	if limit > 0 {
		offset = (track.Dy() - height) * (limit - max(0, min(limit, scroll))) / limit
	}
	thumb := image.Rect(track.Min.X, track.Min.Y+offset, track.Max.X, track.Min.Y+offset+height)
	return track, thumb
}

func logScrollAt(top int, track image.Rectangle, height, limit int) int {
	travel := track.Dy() - height
	if travel <= 0 {
		return 0
	}
	top = max(track.Min.Y, min(track.Max.Y-height, top))
	return limit - (top-track.Min.Y)*limit/travel
}
