package gui

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"math"
	"slices"
	"srpg/internal/core"
	"srpg/internal/session"
)

const deploymentTop = Height * 3 / 4
const deploymentCardWidth = 172

func (g *Game) deploying() bool { return g.S != nil && g.S.Engine != nil && g.S.Engine.Deploying() }
func (g *Game) battleViewport() image.Rectangle {
	if g.deploying() {
		return image.Rect(0, 0, Width, deploymentTop)
	}
	return viewport
}
func deploymentPulse(seconds float64) float64 { return .12 + .18*(math.Sin(2*math.Pi*seconds)+1)/2 }
func (g *Game) resetDeployment() {
	g.deploymentSelected = ""
	g.deploymentScroll = 0
	g.dragging, g.dragMoved = false, false
	g.clearOrder()
	g.selected = ""
}
func (g *Game) selectDeployment(id string) {
	if !g.deploying() {
		return
	}
	o := g.S.Engine.Observe()
	if !slices.Contains(o.Deployment, id) {
		return
	}
	if g.unitView(o, id) != nil {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "unplace", Actor: id}})
		g.deploymentSelected = id
	} else if g.deploymentSelected == id {
		g.deploymentSelected = ""
	} else {
		g.deploymentSelected = id
	}
}
func (g *Game) deploymentMapClick(x, y int) {
	o := g.S.Engine.Observe()
	for _, u := range o.UnitViews {
		if u.X == x && u.Y == y && u.HP > 0 {
			if slices.Contains(o.Deployment, u.ID) {
				g.selectDeployment(u.ID)
			}
			return
		}
	}
	for _, c := range g.S.Engine.LegalActions(g.deploymentSelected).Commands {
		if c.Kind == "place" && c.X == x && c.Y == y {
			if r := g.request(session.Request{Op: "command", Command: c}); r.OK {
				g.deploymentSelected = ""
			}
			return
		}
	}
}
func (g *Game) finishDeployment() {
	if !g.deploying() {
		return
	}
	if r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "deployment_finish"}}); r.OK {
		g.resetDeployment()
	}
}
func (g *Game) deploymentMarks() []Highlight {
	if g.deploymentSelected == "" {
		return nil
	}
	out := []Highlight{}
	for _, c := range g.S.Engine.LegalActions(g.deploymentSelected).Commands {
		if c.Kind == "place" {
			out = append(out, Highlight{U: c.X, V: c.Y, Color: color.RGBA{R: 255, G: 224, B: 120, A: uint8(deploymentPulse(g.uiSeconds) * 255)}})
		}
	}
	return out
}

func (g *Game) deploymentPreviewCommand() (core.Command, bool) {
	if !g.deploying() || g.deploymentSelected == "" || !g.hover.ok || g.overlay != "" || len(g.modals) > 0 || g.input {
		return core.Command{}, false
	}
	for _, c := range g.S.Engine.LegalActions(g.deploymentSelected).Commands {
		if c.Kind == "place" && c.X == g.hover.x && c.Y == g.hover.y {
			return c, true
		}
	}
	return core.Command{}, false
}

func (g *Game) drawDeploymentPreview(o core.Observation) {
	c, ok := g.deploymentPreviewCommand()
	if !ok {
		return
	}
	for _, ch := range o.Characters {
		if ch.ID != c.Actor {
			continue
		}
		art := g.battleClassArt(ch.Def, ch.Class, g.S.Data.Characters[ch.Def].Look.Faction)
		if art == nil {
			return
		}
		u := &unitVis{art: art, ally: true, actor: true, dir: "SE", anim: "idle", dying: -1}
		x, y := g.field.Center(float64(c.X), float64(c.Y))
		g.field.drawUnit(u, x, y-g.field.Lift(c.X, c.Y), true)
		return
	}
}
func (g *Game) deploymentCardsRect() image.Rectangle {
	return image.Rect(48, deploymentTop+48, Width-48, Height-52)
}
func (g *Game) scrollDeployment(delta int) {
	if !g.deploying() {
		return
	}
	maxScroll := max(0, len(g.S.Engine.Observe().Deployment)*deploymentCardWidth-g.deploymentCardsRect().Dx())
	g.deploymentScroll = max(0, min(maxScroll, g.deploymentScroll+delta))
}
func (g *Game) updateDeploymentPanel() bool {
	if !g.deploying() {
		return false
	}
	x, y := cursorPos()
	if y < deploymentTop {
		return false
	}
	if mouseJustPressed(ebiten.MouseButtonRight) {
		g.deploymentSelected = ""
	}
	if wx, wy := ebiten.Wheel(); wx != 0 || wy != 0 {
		g.scrollDeployment(int(-(wx + wy) * 70))
	}
	return x >= 0 && x < Width
}
func (g *Game) deploymentKeys() {
	pressed := keyJustPressed
	if pressed(ebiten.KeyE) || pressed(ebiten.KeyEnter) {
		if len(g.S.Engine.LegalActions("").Commands) > 0 {
			g.finishDeployment()
		}
		return
	}
	if pressed(ebiten.KeyTab) {
		ids := g.S.Engine.Observe().Deployment
		start := -1
		for i, id := range ids {
			if id == g.deploymentSelected {
				start = i
			}
		}
		for n := 1; n <= len(ids); n++ {
			id := ids[(start+n)%len(ids)]
			if g.unitView(g.S.Engine.Observe(), id) == nil {
				g.deploymentSelected = id
				g.scrollDeployment((start+n)%len(ids)*deploymentCardWidth - g.deploymentScroll)
				break
			}
		}
	}
}
func (g *Game) drawDeployment(dst *ebiten.Image, o core.Observation) {
	placed := 0
	for _, id := range o.Deployment {
		if g.unitView(o, id) != nil {
			placed++
		}
	}
	rect(dst, 0, 0, Width, 40, color.NRGBA{R: 20, G: 14, B: 12, A: 240})
	g.block(image.Rect(0, 0, Width, 40))
	g.label(dst, fmt.Sprintf("아군 배치 · %s   배치 %d / %d", o.Map.Name, placed, len(o.Deployment)), 16, 8, 18, gold)
	g.btn(dst, image.Rect(1176, 5, 1272, 34), "메뉴", func() { g.openOverlay("menu") })
	r := image.Rect(0, deploymentTop, Width, Height)
	rect(dst, 0, float32(deploymentTop), Width, Height-deploymentTop, color.NRGBA{R: 28, G: 23, B: 20, A: 255})
	rect(dst, 0, float32(deploymentTop), Width, 2, gold)
	g.block(r)
	hint := "장수를 고른 뒤 밝은 빈칸을 누르세요. 배치한 아군 클릭: 해제 후 재배치 · Tab: 다음 장수 · 우클릭/Esc: 선택 취소"
	if g.hover.ok {
		hint += fmt.Sprintf("   지형: %s", g.S.Data.Tiles[string(g.field.Tile(g.hover.x, g.hover.y))].Name)
	}
	g.label(dst, hint, 16, float64(deploymentTop+12), 14, ink)
	cards := g.deploymentCardsRect()
	clip := dst.SubImage(image.Rect(int(float64(cards.Min.X)*scale), int(float64(cards.Min.Y)*scale), int(float64(cards.Max.X)*scale), int(float64(cards.Max.Y)*scale))).(*ebiten.Image)
	for i, id := range o.Deployment {
		x := cards.Min.X + i*deploymentCardWidth - g.deploymentScroll
		card := image.Rect(x, cards.Min.Y, x+deploymentCardWidth-10, cards.Max.Y)
		if !card.Overlaps(cards) {
			continue
		}
		state, tint := "미배치", muted
		if g.unitView(o, id) != nil {
			state, tint = "배치됨 · 클릭 시 재배치", good
		}
		if g.deploymentSelected == id {
			state, tint = "선택 · 빈칸을 누르세요", gold
		}
		window(clip, card)
		strokeRect(clip, float32(card.Min.X), float32(card.Min.Y), float32(card.Dx()), float32(card.Dy()), 2, tint)
		g.portrait(clip, id, float64(x+10), float64(cards.Min.Y+10), 54)
		cl := ""
		level := 0
		for _, c := range o.Characters {
			if c.ID == id {
				cl, level = c.Class, c.Level
			}
		}
		g.label(clip, g.name(id), float64(x+74), float64(cards.Min.Y+10), 17, ink)
		g.label(clip, fmt.Sprintf("Lv%d", level), float64(x+74), float64(cards.Min.Y+38), 13, muted)
		g.label(clip, g.className(cl), float64(x+10), float64(cards.Min.Y+72), 13, ink)
		g.label(clip, state, float64(x+10), float64(cards.Max.Y-24), 12, tint)
		g.buttons = append(g.buttons, button{rect: card.Intersect(cards), click: func() { g.selectDeployment(id) }})
	}
	g.btn(dst, image.Rect(8, cards.Min.Y+32, 40, cards.Min.Y+66), "‹", func() { g.scrollDeployment(-deploymentCardWidth) })
	g.btn(dst, image.Rect(Width-40, cards.Min.Y+32, Width-8, cards.Min.Y+66), "›", func() { g.scrollDeployment(deploymentCardWidth) })
	var finish func()
	if placed == len(o.Deployment) {
		finish = g.finishDeployment
	}
	g.btn(dst, image.Rect(Width-180, Height-42, Width-16, Height-8), "전투 시작 Enter", finish)
	g.label(dst, fmt.Sprintf("배치 %d / %d", placed, len(o.Deployment)), 16, Height-34, 16, gold)
	if g.editorReturn != nil {
		g.btn(dst, image.Rect(Width-336, Height-42, Width-192, Height-8), "에디터로 돌아가기", g.endEditorTrial)
	}
}
