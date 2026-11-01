package gui

// focusOfficer centres the camera on an allied officer and selects it, opening its
// orders when it may still act this turn.
func (g *Game) focusOfficer(id string) {
	if g.popupPending() {
		return
	}
	o := g.S.View()
	u := g.unitView(o, id)
	if u == nil {
		return
	}
	if !g.cameraLocked() {
		g.panToCell(u.X, u.Y)
	}
	g.selected = id
	if g.busy() {
		return
	}
	g.clearOrder()
	if o.Turn == "ally" && !u.AI && !u.Done {
		g.begin(*u)
	}
}
