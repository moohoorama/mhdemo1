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

func (g *Game) deploying() bool { return g.S != nil && g.S.Engine != nil && g.S.Engine.Deploying() }
func (g *Game) battleViewport() image.Rectangle {
	return image.Rect(0, 0, worldW, worldH) // the whole window, HUD drawn over it
}
func deploymentPulse(seconds float64) float64 { return .12 + .18*(math.Sin(2*math.Pi*seconds)+1)/2 }
func (g *Game) resetDeployment() {
	g.deploymentSelected = ""
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

// updateDeploymentPanel drops the selection on a right click.
func (g *Game) updateDeploymentPanel() {
	if g.deploying() && !g.popupPending() && mouseJustPressed(ebiten.MouseButtonRight) {
		g.deploymentSelected = ""
	}
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
	}
}

// deploymentListRect is the sortie's list at the bottom right, as tall as its rows up to the top-right bar.
func (g *Game) deploymentListRect(o core.Observation) image.Rectangle {
	top := hudTop() + 64 + 12
	h := min(uiH-12-top, 48+len(o.Deployment)*touchH+8)
	return image.Rect(uiW-12-300, uiH-12-h, uiW-12, uiH-12)
}

// drawDeployment lays the deployment out like the battle: the help (speed, zoom, the goal) at the top left, the
// count with 메뉴 and 전투 시작 at the top right, and the sortie at the bottom right, a row picking an officer to
// place. Who goes, gear and deputies were settled in the sortie preparation.
func (g *Game) drawDeployment(dst *ebiten.Image, o core.Observation) {
	limit := g.S.Data.Stages[o.Stage].Max
	g.battleLeftPanel(dst, o, fmt.Sprintf("출전 %d / %d", len(o.Deployment), limit), nil)
	bar := image.Rect(uiW-12-infoW, hudTop(), uiW-12, hudTop()+64)
	g.window(dst, bar)
	g.block(bar)
	placed := 0
	for _, id := range o.Deployment {
		if g.unitView(o, id) != nil {
			placed++
		}
	}
	g.label(dst, fmt.Sprintf("배치 %d / %d", placed, len(o.Deployment)), float64(bar.Min.X+14), float64(bar.Min.Y+20), 15, ink)
	var finish func()
	if len(g.S.Engine.LegalActions("").Commands) > 0 {
		finish = g.finishDeployment
	}
	g.actionRow(dst, bar.Max.X-8, bar.Min.Y+8, action{"메뉴", false, func() { g.openOverlay("menu") }}, action{"전투 시작", true, finish})

	r := g.deploymentListRect(o)
	g.window(dst, r)
	g.block(r)
	g.label(dst, "출전 부대", float64(r.Min.X+14), float64(r.Min.Y+12), 15, gold)
	g.label(dst, "고르고 칸을 누름", float64(r.Max.X-14)-g.width("고르고 칸을 누름", 12), float64(r.Min.Y+15), 12, muted)
	ids := o.Deployment
	g.listRows(dst, &g.party.placing, image.Rect(r.Min.X+4, r.Min.Y+44, r.Max.X-4, r.Max.Y-6), len(ids), touchH, func(i int, rr image.Rectangle) {
		id := ids[i]
		of := g.officer(o, id)
		if of == nil {
			return
		}
		rowBack(dst, rr, id == g.deploymentSelected)
		g.portrait(dst, id, of.Class, float64(rr.Min.X+10), float64(rr.Min.Y+7), 34)
		name := g.name(id)
		g.label(dst, name, float64(rr.Min.X+54), float64(rr.Min.Y+13), 16, ink)
		gx := float64(rr.Min.X+62) + g.width(name, 16)
		if of.Deputy != "" {
			g.label(dst, "· "+g.name(of.Deputy), gx-2, float64(rr.Min.Y+17), 12, muted)
			gx += g.width("· "+g.name(of.Deputy), 12) + 4
		}
		g.missingGear(dst, of.Class, of.Equipment, gx, float64(rr.Min.Y+11))
		state, c := "배치됨", muted
		switch {
		case id == g.deploymentSelected:
			state, c = "선택", gold
		case g.unitView(o, id) == nil:
			state, c = "미배치", gold
		}
		g.label(dst, state, float64(rr.Max.X-12)-g.width(state, 12), float64(rr.Min.Y+17), 12, c)
	}, func(i int) { g.selectDeployment(ids[i], true) })

	hint := "아군을 누르고 밝은 칸을 누르면 옮깁니다 · Tab 다음 장수 · 우클릭 선택 취소"
	if g.hover.ok {
		hint += "   지형: " + g.S.Data.CellName(g.engine().Cell(g.hover.x, g.hover.y))
	}
	g.outlined(dst, hint, 16, float64(uiH-30), 13)
	if g.editorReturn != nil {
		g.btn(dst, image.Rect(12, uiH-90, 192, uiH-42), "에디터로 돌아가기", g.endEditorTrial)
	}
}
