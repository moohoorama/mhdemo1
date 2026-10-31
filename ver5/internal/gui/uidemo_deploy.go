package gui

import (
	"slices"

	"srpg/internal/core"
	"srpg/internal/session"
)

func init() {
	demos["deploy"] = demoDeploy
	demos["deploy-deputy"] = demoDeployDeputy
}

// demoDeploy opens a battle at its deployment, where the sortie already stands on the field, and walks
// it for cmd/_uidemo-vm: the field (frame 20), the lord picked from its card with the officer window
// (60), the last officer called back to wait (80) and sent out again (130), its weapon picker (150),
// the second officer picked on the map and then the third over it, its empty cells lit and the cursor
// on one (172), the third moved there (185), and the edge arrows on the right (200), bottom (215), left (230) and lower
// right (245).
func demoDeploy(d *Demo) error {
	g := d.g
	if err := demoDeploySetup(d); err != nil {
		return err
	}
	ids := g.S.View().Deployment
	cell := func(id string) (int, int) {
		u := g.unitView(g.S.View(), id)
		return u.X, u.Y
	}
	at := func(x, y int) { cursorPos = func() (int, int) { return x, y } }
	aim := func(u, v int) { // the cursor on a cell
		x, y := g.field.Center(float64(u), float64(v))
		sx, sy := worldToUI(g.toScreen(x, y))
		at(int(sx), int(sy))
	}
	d.step = func(f int) {
		if len(ids) < 3 {
			return
		}
		g.updateDeploymentPanel() // the demo only advances the game, so the panel and the edge arrow
		g.edgeDir, g.edgeOn = g.edgeArrow(false)
		last := ids[len(ids)-1]
		switch f {
		case 25:
			g.toggleSortie(ids[0])
		case 70:
			g.toggleSortie(last)
		case 85:
			g.toggleSortie(last)
		case 135:
			g.selected = last
			g.openOverlay("equipment")
			g.openPicker("weapon")
			g.equip.alone = true
		case 155:
			g.equip.slot = ""
		case 160:
			g.deploymentSelected = ""
			g.mapClick(cell(ids[1]))
		case 165:
			g.mapClick(cell(ids[2]))
			if moves := g.repositions(); len(moves) > 0 {
				aim(moves[0].X, moves[0].Y)
			}
		case 175:
			if moves := g.repositions(); len(moves) > 0 {
				g.mapClick(moves[0].X, moves[0].Y)
			}
		case 190:
			g.deploymentSelected = ""
			at(uiW-12, 380)
		case 205:
			at(640, deploymentTop()-8)
		case 220:
			at(8, 230)
		case 235:
			at(uiW-12, deploymentTop()-8)
		}
	}
	return nil
}

// demoDeploySetup opens the demo stage's deployment from the editor's trial, the sortie standing.
func demoDeploySetup(d *Demo) error {
	g := d.g
	g.editor = &mapEditor{level: 1, facing: "NW", selected: -1, zoom: 1, brushSize: 4}
	g.editorOpen(g.S.Data.StageIndex(d.stage))
	g.editorTrial()
	if _, err := g.S.Engine.Apply(core.Command{Kind: "start"}); err != nil {
		return err
	}
	g.editorReturn = nil
	g.ensureField(g.S.View())
	g.placeSortie()
	g.sync(g.S.View())
	g.modals = nil
	cursorPos = func() (int, int) { return -1000, -1000 }
	return nil
}

// demoDeployDeputy gives every officer the deputy_appoint trait and names deputies at the deployment:
// the second officer's deputy window on 전투력 (70), a bench officer previewed (85), the lord picked
// with why it cannot serve (88), the table sorted by attack with an officer of the sortie picked (100), 열전 sorted by 무력 (115), the bench officer named (130), 기본 (145), the cards
// with the new deputy unable to go (175), and the last officer of the sortie named the third's
// deputy, leaving the field (220).
func demoDeployDeputy(d *Demo) error {
	g := d.g
	if err := demoDeploySetup(d); err != nil {
		return err
	}
	d.edit(func(s *core.State) {
		for i := range s.Characters {
			if g.S.Data.Characters[s.Characters[i].Def].Playable {
				s.Characters[i].Learned = append(s.Characters[i].Learned, "deputy_appoint")
			}
		}
	})
	o := g.S.View()
	ids, cards := o.Deployment, g.sortieCards(o)
	if len(cards) > len(ids) { // the bench officer learns a dozen talents, to show how a long list of traits reads
		d.edit(func(s *core.State) {
			for i := range s.Characters {
				if s.Characters[i].ID != cards[len(ids)] {
					continue
				}
				for _, t := range sorted(g.S.Data.Traits) {
					if g.S.Data.Traits[t].Kind == "talent" && len(s.Characters[i].Learned) < 12 && !slices.Contains(s.Characters[i].Learned, t) {
						s.Characters[i].Learned = append(s.Characters[i].Learned, t)
					}
				}
			}
		})
	}
	d.step = func(f int) {
		if len(ids) < 3 || len(cards) <= len(ids) {
			return
		}
		g.updateDeploymentPanel()
		bench, last := cards[len(ids)], ids[len(ids)-1]
		switch f {
		case 20:
			g.toggleSortie(ids[1])
		case 55:
			g.selected = ids[1]
			g.openOverlay("deputy")
			g.deputyList.kind = 1
		case 75:
			g.deputyPick = bench
		case 80:
			g.deputyInfo = equipChip{"deputy", g.S.Engine.DeputyTraits(bench)[0]}
		case 83:
			g.deputyInfo = equipChip{}
		case 86:
			g.deputyPick = ids[0]
		case 89:
			g.deputyPick = ids[3]
		case 90:
			g.deputyList.order = tableSort{"attack", 2}
		case 105:
			g.deputyList.kind, g.deputyList.order = 2, tableSort{"a1", 2}
			g.deputyPick = bench
		case 120:
			g.request(session.Request{Op: "command", Command: core.Command{Kind: "deputy", Actor: ids[1], Target: bench}})
			g.deputyPick = ""
		case 135:
			g.deputyList.kind = 0
		case 150:
			g.closeOverlay()
		case 180:
			g.command(core.Command{Kind: "deputy", Actor: ids[2], Target: last})
			g.deploymentSelected = ""
			g.toggleSortie(ids[2])
		}
	}
	return nil
}
