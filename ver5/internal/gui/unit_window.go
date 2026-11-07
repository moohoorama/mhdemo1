package gui

import (
	"fmt"
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/core"
)

// unitWindowState is what the unit's detail shows: its tab, on the trait tab whose shelves (the lead's or the
// deputy's) and the trait picked, and whether it only shows (in battle).
type unitWindowState struct {
	tab, side int
	trait     string
	readOnly  bool
}

// openUnit opens the detail of the unit led by g.selected; readOnly leaves out every button but 닫기.
func (g *Game) openUnit(readOnly bool) {
	g.unitWin = unitWindowState{readOnly: readOnly}
	g.openOverlay("unit")
}

// unitLeads are the units of the sortie the detail steps through with ‹ ›.
func (g *Game) unitLeads(o core.Observation) []string {
	out := []string{}
	for _, id := range o.Deployment {
		if g.servedBy(o, id) == "" {
			out = append(out, id)
		}
	}
	return out
}

// unitWindow is the unit's detail, filling the screen: the lead with the deputy in four tabs — 전투력 with the
// deputy's weight and the lead alone, with the bonus and as a unit; 특성 split into the lead's shelves and the
// deputy's; 열전 of both; 장비 of the lead.
func (g *Game) unitWindow(dst *ebiten.Image, x, y, w, h int) {
	o := g.S.View()
	u := &g.unitWin
	of := g.officer(o, g.selected)
	if of == nil {
		return
	}
	leads := g.unitLeads(o)
	step := func(d int) func() {
		i := slices.Index(leads, of.ID)
		if i < 0 || len(leads) < 2 {
			return nil
		}
		return func() { g.selected, u.trait = leads[(i+d+len(leads))%len(leads)], "" }
	}
	g.mBtn(dst, image.Rect(x+8, y+4, x+8+touchH, y+4+touchH), "‹", false, step(-1))
	name := g.name(of.ID) + "대"
	g.label(dst, name, float64(x+touchH+18), float64(y+12), 18, gold)
	nx := x + touchH + 28 + int(g.width(name, 18))
	g.mBtn(dst, image.Rect(nx, y+4, nx+touchH, y+4+touchH), "›", false, step(1))
	g.tabRow(dst, nx+touchH+20, y+4, officerTabs, u.tab, func(i int) { u.tab = i })
	rect(dst, float32(x+8), float32(y+52), float32(w-16), 1, goldDim)
	body := image.Rect(x+16, y+62, x+w-16, y+h-66)
	var acts []action
	switch u.tab {
	case 0:
		g.unitPower(dst, o, of, body)
	case 1:
		acts = g.unitTraits(dst, o, of, body)
	case 2:
		g.unitBio(dst, o, of, body)
	case 3:
		g.officerEquipment(dst, of, body)
		g.label(dst, "부관은 장비를 들지 않습니다. 임명하면 부관의 장비는 창고로 갑니다.", float64(body.Min.X), float64(body.Min.Y+226), 13, muted)
	}
	if u.readOnly {
		g.actionRow(dst, x+w-16, y+h-60, action{"닫기", false, g.closeOverlay})
		return
	}
	var swap, dismiss func()
	if g.S.Engine.CanDeputize(of.ID) {
		swap = func() { g.openOverlay("deputy") }
	}
	if of.Deputy != "" {
		id := of.ID
		dismiss = func() { g.command(core.Command{Kind: "undeputy", Actor: id}) }
	}
	all := append([]action{{"장비 변경", false, func() { g.openOverlay("equipment") }}, {"부관 바꾸기", false, swap}, {"부관 해임", false, dismiss}, {"닫기", false, g.closeOverlay}}, acts...)
	g.actionRow(dst, x+w-16, y+h-60, all...)
}

// unitPower is the 전투력 tab: who leads and who serves, the deputy's weight, the attributes of both and the
// unit's, and the battle stats alone, the bonus and the unit's.
func (g *Game) unitPower(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) {
	x, y := float64(r.Min.X), float64(r.Min.Y)
	g.portrait(dst, of.ID, of.Class, x, y, 40)
	g.label(dst, g.name(of.ID), x+48, y+8, 16, ink)
	g.label(dst, "주장", x+56+g.width(g.name(of.ID), 16), y+12, 12, muted)
	alone, aloneAttrs, err := g.S.Engine.DeputyStats(of.ID, "")
	if err != nil {
		return
	}
	if of.Deputy == "" {
		g.label(dst, "부관이 없습니다. 부관을 두면 부관이 앞선 열전 능력을 가중치만큼 섞습니다.", x+160, y+10, 14, muted)
		g.officerPower(dst, o, of, image.Rect(r.Min.X, r.Min.Y+56, r.Max.X, r.Max.Y))
		return
	}
	d := g.officer(o, of.Deputy)
	dx := x + 120 + g.width(g.name(of.ID), 16)
	if d != nil {
		g.portrait(dst, d.ID, d.Class, dx, y, 40)
	}
	g.label(dst, g.name(of.Deputy), dx+48, y+8, 16, ink)
	g.label(dst, "부관", dx+56+g.width(g.name(of.Deputy), 16), y+12, 12, muted)
	wr, charm := g.S.Engine.DeputyBlend(of.ID, of.Deputy)
	_, depAttrs, _ := g.S.Engine.DeputyStats(of.Deputy, "")
	unit, unitAttrs, _ := g.S.Engine.DeputyStats(of.ID, of.Deputy)
	wx := dx + 120 + g.width(g.name(of.Deputy), 16)
	ws := fmt.Sprintf("부관 가중치 %.0f%%", wr*100)
	g.label(dst, ws, wx, y+8, 16, ink)
	g.label(dst, fmt.Sprintf("= (%.0f × %d + %d) ÷ %d", charm, g.S.Data.Rules.DeputyWeight, depAttrs[5], g.S.Data.Rules.DeputyDivisor), wx+g.width(ws, 16)+8, y+11, 13, muted)
	y += 56
	colW := float64(r.Dx()-24) / 2
	head := func(cx float64, cols []string) {
		for i, c := range cols {
			px := cx + float64(i)*colW/4
			if i > 0 {
				px += colW/4 - g.width(c, 12) - 12
			}
			g.label(dst, c, px, y, 12, muted)
		}
		rect(dst, float32(cx), float32(y+22), float32(colW), 1, goldDim)
	}
	cell := func(cx float64, i int, s string, c any) {
		px := cx + float64(i)*colW/4
		if i > 0 {
			px += colW/4 - g.width(s, 14) - 12
		}
		col := ink
		switch v := c.(type) {
		case bool:
			if v {
				col = good
			}
		}
		g.label(dst, s, px, y, 14, col)
	}
	head(x, []string{"열전", g.name(of.ID), g.name(of.Deputy), "부대"})
	head(x+colW+24, []string{"전투", "혼자", "보정", "부대"})
	y += 28
	for i := range 7 {
		if i < 6 {
			cell(x, 0, attrNames[i], nil)
			cell(x, 1, fmt.Sprint(aloneAttrs[i]), nil)
			cell(x, 2, fmt.Sprint(depAttrs[i]), nil)
			cell(x, 3, fmt.Sprint(unitAttrs[i]), unitAttrs[i] > aloneAttrs[i])
		}
		s := statRows[i]
		a, b := s.get(alone), s.get(unit)
		bonus := "—"
		if b != a {
			bonus = fmt.Sprintf("%+d", b-a)
		}
		cell(x+colW+24, 0, s.name, nil)
		cell(x+colW+24, 1, fmt.Sprint(a), nil)
		cell(x+colW+24, 2, bonus, b != a)
		cell(x+colW+24, 3, fmt.Sprint(b), nil)
		y += 26
	}
}

// unitTraits is the 특성 tab: the lead's shelves (고유, 병종, 장비) or the deputy's (부관 고유, 부관 병종), the
// picked trait described under them; it gives the action that learns it, spending the points of whose shelf it is.
func (g *Game) unitTraits(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) []action {
	u := &g.unitWin
	owner := of
	sides := []string{g.name(of.ID) + " · 고유·병종·장비"}
	if of.Deputy != "" {
		sides = append(sides, g.name(of.Deputy)+" · 부관 고유·병종")
	}
	if u.side >= len(sides) {
		u.side = 0
	}
	g.tabRow(dst, r.Min.X, r.Min.Y-6, sides, u.side, func(i int) { u.side, u.trait = i, "" })
	shelves := g.shelvesOf(of)
	if u.side == 1 {
		owner = g.officer(o, of.Deputy)
		if owner == nil {
			return nil
		}
		s := g.shelvesOf(owner)[:2]
		s[0].head, s[1].head = "부관 고유", "부관 병종"
		shelves = s
	}
	all := []traitEntry{}
	for _, s := range shelves {
		all = append(all, s.traits...)
	}
	var picked *traitEntry
	if i := slices.IndexFunc(all, func(e traitEntry) bool { return e.id == u.trait }); i >= 0 {
		picked = &all[i]
	} else if i := slices.IndexFunc(all, func(e traitEntry) bool { return e.affordable }); i >= 0 {
		picked = &all[i]
	} else if len(all) > 0 {
		picked = &all[0]
	}
	pick := ""
	if picked != nil {
		pick = picked.id
	}
	area := image.Rect(r.Min.X, r.Min.Y+48, r.Max.X, r.Max.Y)
	used := g.traitShelves(dst, area, shelves, pick, func(id string) { u.trait = id })
	if picked == nil {
		return nil
	}
	note := ""
	switch {
	case picked.have && u.side == 1:
		note = fmt.Sprintf("%s이 부관인 동안 %s대에 더해집니다", g.name(owner.ID), g.name(of.ID))
	case picked.have:
	case !picked.learnable && picked.requires != "":
		note = "선행 특성 " + picked.requires + "을 먼저 배워야 합니다"
	case !picked.affordable:
		note = fmt.Sprintf("특성치가 %d 모자랍니다 (%d 필요, %d 있음)", picked.cost-owner.Points, picked.cost, owner.Points)
	default:
		note = fmt.Sprintf("%s 특성치 %d → %d", g.name(owner.ID), owner.Points, owner.Points-picked.cost)
	}
	cardY := min(area.Min.Y+used+6, area.Max.Y-84)
	g.traitCard(dst, image.Rect(r.Min.X, cardY, r.Max.X, cardY+84), picked.id, picked.group, note)
	if u.readOnly || picked.have || !picked.learnable {
		return nil
	}
	var learn func()
	if picked.affordable && g.canLearnNow(owner.ID) {
		id, trait := owner.ID, picked.id
		learn = func() { g.command(core.Command{Kind: "learn", Actor: id, Trait: trait}) }
	}
	label := fmt.Sprintf("%s 배우기 · %d", g.traitName(picked.id), picked.cost)
	if u.side == 1 {
		label = fmt.Sprintf("%s이 배우기 · %d", g.name(owner.ID), picked.cost)
	}
	return []action{{label, true, learn}}
}

// unitBio is the 열전 tab: the lead's and the deputy's attributes and biographies side by side.
func (g *Game) unitBio(dst *ebiten.Image, o core.Observation, of *core.Character, r image.Rectangle) {
	ids := []string{of.ID}
	if of.Deputy != "" {
		ids = append(ids, of.Deputy)
	}
	colW := (r.Dx() - 24) / 2
	for i, id := range ids {
		c := g.officer(o, id)
		if c == nil {
			continue
		}
		x, y := float64(r.Min.X+i*(colW+24)), float64(r.Min.Y)
		g.label(dst, g.name(id), x, y, 18, gold)
		role := "주장"
		if i == 1 {
			role = "부관"
		}
		g.label(dst, role, x+g.width(g.name(id), 18)+8, y+5, 12, muted)
		_, attrs, _ := g.S.Engine.DeputyStats(id, "")
		y += 30
		for j, a := range attrNames {
			px, py := x+float64(j%3)*float64(colW)/3, y+float64(j/3)*24
			g.label(dst, a, px, py, 14, muted)
			g.label(dst, fmt.Sprint(attrs[j]), px+40, py, 14, ink)
		}
		y += 56
		g.label(dst, wrapWidth(g.S.Data.Characters[c.Def].Biography, g.face(14), float64(colW)), x, y, 14, ink)
	}
}
