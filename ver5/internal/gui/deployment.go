package gui

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"image/color"
	"math"
	"slices"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

// deploymentBar is the height of the deployment cards along the bottom, in UI px.
const deploymentBar = Height / 4

// deploymentTop is the top of the deployment cards, in UI px.
func deploymentTop() int { return uiH - deploymentBar }

const deploymentCardWidth = 172

func (g *Game) deploying() bool { return g.S != nil && g.S.Engine != nil && g.S.Engine.Deploying() }
func (g *Game) battleViewport() image.Rectangle {
	if g.deploying() {
		return image.Rect(0, 0, worldW, int(float64(deploymentTop())*uiPx/worldPx))
	}
	return image.Rect(0, 0, worldW, worldH) // the whole window, HUD drawn over it
}
func deploymentPulse(seconds float64) float64 { return .12 + .18*(math.Sin(2*math.Pi*seconds)+1)/2 }
func (g *Game) resetDeployment() {
	g.deploymentSelected = ""
	g.deploymentScroll = 0
	g.sortieTried = ""
	g.dragging, g.dragMoved = false, false
	g.clearOrder()
	g.selected = ""
}

// startSortie leaves a battle node for its deployment at once: the sortie is trimmed to the stage's
// limit (the lord and the required officers kept first) and the officers stand on the field, where
// the player changes who goes, their gear and their deputies.
func (g *Game) startSortie() {
	o := g.S.View()
	if g.sortieTried == o.Node {
		return
	}
	g.sortieTried = o.Node
	st := g.S.Data.Stages[o.Stage]
	if len(o.Deployment) > st.Max {
		ids := []string{}
		for _, keep := range []bool{true, false} {
			for _, id := range o.Deployment {
				if g.sortieLocked(id) == keep && len(ids) < st.Max {
					ids = append(ids, id)
				}
			}
		}
		g.command(core.Command{Kind: "deploy", Deployment: ids})
	}
	g.command(core.Command{Kind: "start"})
}

// placeSortie stands every officer of the sortie on the first free ally cells.
func (g *Game) placeSortie() {
	if ids := g.S.View().Deployment; len(ids) > 0 {
		g.request(session.Request{Op: "command", Command: core.Command{Kind: "deploy", Deployment: ids}})
	}
}

// sortieLocked reports whether the officer always goes: the lord and those the stage requires.
func (g *Game) sortieLocked(id string) bool {
	o := g.S.View()
	of := g.officer(o, id)
	return of != nil && g.S.Data.Characters[of.Def].Lord || slices.Contains(g.S.Data.Stages[o.Stage].Required, id)
}

// sortieCards are the living officers: those in the sortie first, in its order, then those waiting,
// then the deputies, who cannot go.
func (g *Game) sortieCards(o core.Observation) []string {
	out := slices.Clone(o.Deployment)
	for _, deputies := range []bool{false, true} {
		for _, c := range g.roster(o) {
			if !c.Dead && (g.servedBy(o, c.ID) != "") == deputies && !slices.Contains(out, c.ID) {
				out = append(out, c.ID)
			}
		}
	}
	return out
}

// servedBy is the officer whom id serves as deputy, or "".
func (g *Game) servedBy(o core.Observation, id string) string {
	for _, c := range o.Characters {
		if c.Deputy == id && id != "" {
			return c.ID
		}
	}
	return ""
}

// toggleSortie sends a waiting officer out onto the field or calls one back; a locked officer is
// only selected.
func (g *Game) toggleSortie(id string) {
	o := g.S.View()
	if lead := g.servedBy(o, id); lead != "" {
		g.toast(g.name(lead)+"의 부관이라 출전할 수 없습니다", bad)
		return
	}
	if g.sortieLocked(id) {
		g.selectDeployment(id, true)
		return
	}
	if i := slices.Index(o.Deployment, id); i >= 0 {
		if r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "deploy", Deployment: slices.Delete(slices.Clone(o.Deployment), i, i+1)}}); r.OK && g.deploymentSelected == id {
			g.deploymentSelected = ""
		}
		return
	}
	if len(o.Deployment) >= g.S.Data.Stages[o.Stage].Max {
		g.toast("출전 인원이 찼습니다", bad)
		return
	}
	if r := g.request(session.Request{Op: "command", Command: core.Command{Kind: "deploy", Deployment: append(slices.Clone(o.Deployment), id)}}); r.OK {
		g.selectDeployment(id, true)
	}
}

// selectDeployment selects a placed officer, or clears the selection when it is picked again; picked
// from the cards, the camera goes to it.
func (g *Game) selectDeployment(id string, pan bool) {
	u := g.unitView(g.S.View(), id)
	if !g.deploying() || u == nil {
		return
	}
	if g.deploymentSelected == id {
		g.deploymentSelected = ""
		return
	}
	g.deploymentSelected = id
	if pan {
		g.panToCell(u.X, u.Y)
	}
}

// deploymentMapClick selects the officer of the sortie on the cell (another one picked over the
// selected one, the same one again clears it), else moves the selected officer there.
func (g *Game) deploymentMapClick(x, y int) {
	o := g.S.View()
	for _, u := range o.UnitViews {
		if u.X == x && u.Y == y && u.HP > 0 && slices.Contains(o.Deployment, u.ID) {
			g.selectDeployment(u.ID, false)
			return
		}
	}
	if c, ok := g.repositionTo(x, y); ok {
		g.request(session.Request{Op: "command", Command: c})
		return
	}
	g.deploymentSelected = ""
}

// repositionTo is the selected officer's move to the empty ally cell (x, y).
func (g *Game) repositionTo(x, y int) (core.Command, bool) {
	for _, c := range g.repositions() {
		if c.X == x && c.Y == y {
			return c, true
		}
	}
	return core.Command{}, false
}

// repositions are the selected officer's moves to empty ally cells; the engine would also trade
// cells with an officer standing there, but a click on one picks it instead.
func (g *Game) repositions() []core.Command {
	if g.deploymentSelected == "" {
		return nil
	}
	taken := map[[2]int]bool{}
	for _, u := range g.S.View().UnitViews {
		if u.HP > 0 {
			taken[[2]int{u.X, u.Y}] = true
		}
	}
	out := []core.Command{}
	for _, c := range g.S.Engine.LegalActions(g.deploymentSelected).Commands {
		if c.Kind == "reposition" && !taken[[2]int{c.X, c.Y}] {
			out = append(out, c)
		}
	}
	return out
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
	if u := g.unitView(g.S.View(), g.deploymentSelected); u != nil {
		out = append(out, Highlight{U: u.X, V: u.Y, Color: color.RGBA{R: 255, G: 250, B: 220, A: 150}})
	}
	for _, c := range g.repositions() {
		out = append(out, Highlight{U: c.X, V: c.Y, Color: color.RGBA{R: 255, G: 224, B: 120, A: uint8(deploymentPulse(g.uiSeconds) * 255)}})
	}
	return out
}

// deploymentPreviewCommand is the move to the empty cell under the cursor, shown as a faint officer there.
func (g *Game) deploymentPreviewCommand() (core.Command, bool) {
	if !g.deploying() || !g.hover.ok || g.overlay() != "" || len(g.modals) > 0 || g.input {
		return core.Command{}, false
	}
	return g.repositionTo(g.hover.x, g.hover.y)
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
		mount := ""
		if slices.Contains(g.S.Data.Classes[ch.Class].Tags, content.TagMounted) {
			mount = ch.Equipment["accessory"]
		}
		art := g.battleClassArt(ch.Def, ch.Class, g.S.Data.Equipment[ch.Equipment["weapon"]].Type, mount, g.S.Data.Characters[ch.Def].Look.Faction)
		if art == nil {
			return
		}
		u := &unitVis{art: art, ally: true, actor: true, dir: "SE", anim: "idle", dying: -1}
		x, y := g.field.Center(float64(c.X), float64(c.Y))
		g.field.drawUnit(u, x, y-g.field.Lift(c.X, c.Y), .5)
		return
	}
}
func (g *Game) deploymentCardsRect() image.Rectangle {
	return image.Rect(48, deploymentTop()+48, uiW-48, uiH-52)
}
func (g *Game) scrollDeployment(delta int) {
	if !g.deploying() {
		return
	}
	maxScroll := max(0, len(g.sortieCards(g.S.View()))*deploymentCardWidth-g.deploymentCardsRect().Dx())
	g.deploymentScroll = max(0, min(maxScroll, g.deploymentScroll+delta))
}
func (g *Game) updateDeploymentPanel() bool {
	if !g.deploying() {
		return false
	}
	if g.equip.alone && g.equip.slot == "" && g.overlay() == "equipment" {
		g.closeOverlay()
	}
	g.equip.alone = g.equip.alone && g.overlay() == "equipment"
	x, y := cursorPos()
	if y < deploymentTop() || g.popupPending() {
		return false
	}
	if mouseJustPressed(ebiten.MouseButtonRight) {
		g.deploymentSelected = ""
	}
	if wx, wy := ebiten.Wheel(); wx != 0 || wy != 0 {
		g.scrollDeployment(int(-(wx + wy) * 70))
	}
	return x >= 0 && x < uiW
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
		ids := g.S.View().Deployment
		if len(ids) == 0 {
			return
		}
		next := ids[(slices.Index(ids, g.deploymentSelected)+1)%len(ids)]
		g.deploymentSelected = ""
		g.selectDeployment(next, true)
		g.scrollDeployment(slices.Index(g.sortieCards(g.S.View()), next)*deploymentCardWidth - g.deploymentScroll)
	}
}
func (g *Game) drawDeployment(dst *ebiten.Image, o core.Observation) {
	limit := g.S.Data.Stages[o.Stage].Max
	head := fmt.Sprintf("출전 %d / %d", len(o.Deployment), limit)
	g.battleLeftPanel(dst, o, head)
	bottom := g.rightPanel(dst, g.partySummary(o), func() { g.openOverlay("roster") })
	if g.deploymentSelected != "" && g.unitView(o, g.deploymentSelected) != nil {
		g.deploymentOfficer(dst, o, g.deploymentSelected, image.Rect(uiW-12-infoW, bottom+12, uiW-12, deploymentTop()-12))
	} else {
		g.minimap(dst, o, bottom+20)
	}
	r := image.Rect(0, deploymentTop(), uiW, uiH)
	rect(dst, 0, float32(deploymentTop()), float32(uiW), float32(uiH-deploymentTop()), color.NRGBA{R: 28, G: 23, B: 20, A: 255})
	rect(dst, 0, float32(deploymentTop()), float32(uiW), 2, gold)
	g.block(r)
	hint := "장수 카드: 출전 · 대기 · 전장의 아군 클릭: 선택 후 밝은 빈칸으로 옮기기 · Tab: 다음 장수 · 우클릭/Esc: 선택 취소"
	if g.hover.ok {
		hint += fmt.Sprintf("   지형: %s", g.S.Data.CellName(g.engine().Cell(g.hover.x, g.hover.y)))
	}
	g.label(dst, hint, 16, float64(deploymentTop()+12), 14, ink)
	cards := g.deploymentCardsRect()
	clip := dst.SubImage(image.Rect(int(float64(cards.Min.X)*drawPx), int(float64(cards.Min.Y)*drawPx), int(float64(cards.Max.X)*drawPx), int(float64(cards.Max.Y)*drawPx))).(*ebiten.Image)
	for i, id := range g.sortieCards(o) {
		x := cards.Min.X + i*deploymentCardWidth - g.deploymentScroll
		card := image.Rect(x, cards.Min.Y, x+deploymentCardWidth-10, cards.Max.Y)
		if !card.Overlaps(cards) {
			continue
		}
		out := slices.Contains(o.Deployment, id)
		state, tint := "대기 · 누르면 출전", muted
		lead := g.servedBy(o, id)
		switch {
		case lead != "":
			state, tint = g.name(lead)+"의 부관 · 출전 불가", bad
		case out && g.sortieLocked(id):
			state, tint = "필수 출전", good
		case out:
			state, tint = "출전 · 누르면 대기", good
		}
		if g.deploymentSelected == id {
			tint = gold
		}
		g.window(clip, card)
		strokeRect(clip, float32(card.Min.X), float32(card.Min.Y), float32(card.Dx()), float32(card.Dy()), 2, tint)
		of := g.officer(o, id)
		g.portrait(clip, id, of.Class, float64(x+10), float64(cards.Min.Y+10), 54)
		if !out {
			rect(clip, float32(card.Min.X+2), float32(card.Min.Y+2), float32(card.Dx()-4), float32(card.Dy()-4), color.NRGBA{R: 20, G: 16, B: 14, A: 120})
		}
		name := ink
		if !out {
			name = muted
		}
		g.label(clip, g.name(id), float64(x+74), float64(cards.Min.Y+10), 17, name)
		g.label(clip, fmt.Sprintf("Lv%d", of.Level), float64(x+74), float64(cards.Min.Y+38), 13, muted)
		g.label(clip, g.className(of.Class), float64(x+10), float64(cards.Min.Y+72), 13, name)
		g.label(clip, state, float64(x+10), float64(cards.Max.Y-24), 12, tint)
		g.addButton(card.Intersect(cards), "", func() { g.toggleSortie(id) })
	}
	g.btn(dst, image.Rect(8, cards.Min.Y+32, 40, cards.Min.Y+66), "‹", func() { g.scrollDeployment(-deploymentCardWidth) })
	g.btn(dst, image.Rect(uiW-40, cards.Min.Y+32, uiW-8, cards.Min.Y+66), "›", func() { g.scrollDeployment(deploymentCardWidth) })
	var finish func()
	if len(g.S.Engine.LegalActions("").Commands) > 0 {
		finish = g.finishDeployment
	}
	g.btn(dst, image.Rect(uiW-180, uiH-42, uiW-16, uiH-8), "전투 시작 Enter", finish)
	g.label(dst, head, 16, float64(uiH-34), 16, gold)
	if g.editorReturn != nil {
		g.btn(dst, image.Rect(uiW-336, uiH-42, uiW-192, uiH-8), "에디터로 돌아가기", g.endEditorTrial)
	}
}

// deploymentOfficer is the window of the selected officer on the right: who it is, then its gear and
// deputy, each row opening the window that changes it.
func (g *Game) deploymentOfficer(dst *ebiten.Image, o core.Observation, id string, r image.Rectangle) {
	of := g.officer(o, id)
	rows := len(equipSlots)
	if g.S.Engine.CanDeputize(id) {
		rows++
	}
	h := 14 + 80 + 4*24 + 8 + rows*50 + 6
	if !g.sortieLocked(id) {
		h += 46
	}
	r.Max.Y = min(r.Max.Y, r.Min.Y+h)
	g.window(dst, r)
	g.block(r)
	x, y := float64(r.Min.X+14), float64(r.Min.Y+14)
	g.portrait(dst, id, of.Class, x, y, 64)
	g.label(dst, g.name(id), x+78, y+2, 20, gold)
	g.label(dst, fmt.Sprintf("%s · Lv%d", g.className(of.Class), of.Level), x+78, y+34, 14, ink)
	if g.sortieLocked(id) {
		g.label(dst, "필수 출전", float64(r.Max.X-14)-g.width("필수 출전", 12), y+4, 12, good)
	}
	y += 80
	if stats, err := g.S.Engine.CharacterStats(id); err == nil {
		for i, s := range statRows {
			sx, sy := x+float64(i%2*148), y+float64(i/2*24)
			g.label(dst, s.name, sx, sy, 13, muted)
			g.label(dst, fmt.Sprint(s.get(stats)), sx+64, sy, 13, ink)
		}
	}
	y += 4*24 + 8
	rowH := 50
	row := func(label string, icon string, dim bool, value string, click func()) {
		rr := image.Rect(r.Min.X+10, int(y), r.Max.X-10, int(y)+rowH-6)
		g.equipPanel(dst, rr, click != nil && hovered(rr), false)
		if click != nil {
			g.addButton(rr, "", click)
		}
		if icon != "" {
			g.itemIcon(dst, icon, float64(rr.Min.X+8), float64(rr.Min.Y+6), 1, dim)
		}
		g.label(dst, label, float64(rr.Min.X+50), float64(rr.Min.Y+4), 12, muted)
		c := ink
		if dim {
			c = muted
		}
		g.label(dst, value, float64(rr.Min.X+50), float64(rr.Min.Y+20), 15, c)
		if click != nil {
			g.label(dst, "›", float64(rr.Max.X-20), float64(rr.Min.Y+12), 18, muted)
		}
		y += float64(rowH)
	}
	for _, slot := range equipSlots {
		s := slot
		open := func() {
			g.selected = id
			g.openOverlay("equipment")
			g.openPicker(s)
			g.equip.alone = true
		}
		if item, ok := of.Equipment[slot]; ok {
			row(slotNames[slot], g.S.Data.Equipment[item].Icon, false, g.itemName(item), open)
		} else {
			row(slotNames[slot], slotIcons[slot], true, "비어 있음", open)
		}
	}
	if g.S.Engine.CanDeputize(id) {
		deputy := "없음"
		if of.Deputy != "" {
			deputy = g.name(of.Deputy)
		}
		row("부관", "", of.Deputy == "", deputy, func() {
			g.selected = id
			g.page = 0
			g.openOverlay("deputy")
		})
	}
	if !g.sortieLocked(id) {
		g.btn(dst, image.Rect(r.Min.X+10, r.Max.Y-46, r.Max.X-10, r.Max.Y-10), "대기로 돌리기", func() { g.toggleSortie(id) })
	}
}
