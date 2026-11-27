package gui

import (
	"fmt"
	"image"
	"sort"

	"srpg/internal/content"
	"srpg/internal/core"
)

func init() {
	demos["orderrange"] = demoOrderRange
	demos["skillrange"] = demoSkillRange
	demos["counterpush"] = demoCounterPush
	demos["leech"] = demoLeech
	demos["shopgain"] = demoShopGain
}

// pairFree puts the first enemy beside the first ally on a side with a free cell behind the ally as well.
func pairFree(d *Demo) (ally, enemy string, err error) {
	g := d.g
	e := g.S.Engine
	snap := e.Snapshot()
	free := func(x, y int) bool {
		for _, u := range snap.Units {
			if u.HP > 0 && u.X == x && u.Y == y {
				return false
			}
		}
		return e.StepCost(ally, x, y) > 0
	}
	ei := -1
	for i, u := range snap.Units {
		if u.Faction == "enemy" && u.HP > 0 && ei < 0 {
			enemy, ei = u.ID, i
		}
	}
	for _, u := range snap.Units {
		if u.Faction != "ally" || u.HP <= 0 {
			continue
		}
		ally = u.ID
		for _, p := range [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
			if free(u.X+p[0], u.Y+p[1]) && free(u.X-p[0], u.Y-p[1]) {
				snap.Units[ei].X, snap.Units[ei].Y = u.X+p[0], u.Y+p[1]
				if g.S.Engine, err = core.Restore(g.S.Data, snap); err != nil {
					return
				}
				g.sync(g.S.View())
				g.camX, g.camY = g.field.Center(float64(u.X)+.5, float64(u.Y)+.5)
				return
			}
		}
	}
	return "", "", fmt.Errorf("no room beside %s", ally)
}

func traitWith(g *Game, fx string) string {
	ids := []string{}
	for id, t := range g.S.Data.Traits {
		if t.Effects[fx] > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

// demoOrderRange picks the first ally (frame 20, red ring where it strikes from its cell), moves it
// tentatively (frame 70) and shows the ring from the new cell (frame 140).
func demoOrderRange(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	actor, _, err := pairFree(d)
	if err != nil {
		return err
	}
	g.zoom, g.zoomFrom, g.zoomTo = 1.5, 1.5, 1.5
	d.step = func(f int) {
		switch f {
		case 20:
			u := g.unitView(g.S.View(), actor)
			g.selected = actor
			g.begin(*u)
		case 70:
			far := g.ord.moves[len(g.ord.moves)/2]
			g.tentative(far.X, far.Y)
		}
	}
	return nil
}

// demoSkillRange opens the first ally's 병법 list and points at each skill row in turn (frames 170, 210, 250).
func demoSkillRange(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	actor, _, err := pairFree(d)
	if err != nil {
		return err
	}
	snap := g.S.Engine.Snapshot()
	for i, u := range snap.Units {
		if u.ID == actor {
			for j, c := range snap.Characters {
				if c.ID == u.Character {
					traits := []string{"fire", "water", "earth"}
					snap.Characters[j].TraitsOverride = &traits
				}
			}
			snap.Units[i].MP = 92
		}
	}
	if g.S.Engine, err = core.Restore(g.S.Data, snap); err != nil {
		return err
	}
	g.sync(g.S.View())
	g.zoom, g.zoomFrom, g.zoomTo = 1.5, 1.5, 1.5
	row := func(i int) {
		label := g.subItems()[i].label
		for _, b := range g.buttons {
			if b.label == label {
				c := b.rect.Min.Add(image.Pt(b.rect.Dx()/2, b.rect.Dy()/2))
				cursorPos = func() (int, int) { return c.X, c.Y }
			}
		}
	}
	d.step = func(f int) {
		switch f {
		case 20:
			g.begin(*g.unitView(g.S.View(), actor))
			g.ord.stage = "menu"
		case 100:
			for _, b := range g.buttons {
				if b.label == "병법" {
					b.click()
				}
			}
		case 150:
			cursorPos = func() (int, int) { return -100, -100 }
		case 170:
			row(0)
		case 210:
			row(1)
		case 250:
			row(2)
		}
	}
	return nil
}

// demoCounterPush has the first ally strike an enemy beside it that pushes back with its counter (frame 30).
func demoCounterPush(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	ally, enemy, err := pairFree(d)
	if err != nil {
		return err
	}
	knock := traitWith(g, content.FxKnockback)
	if knock == "" {
		return fmt.Errorf("no knockback trait")
	}
	g.S.Data.Rules.HitFloor, g.S.Data.Rules.HitCeil = 100, 100
	g.S.Data.Rules.BasicAttack.Coeff *= .1
	found := false
	d.edit(func(s *core.State) {
		var ch string
		for i := range s.Units {
			if s.Units[i].ID == enemy {
				ch = s.Units[i].Character
			}
		}
		for i := range s.Characters {
			if s.Characters[i].ID == ch {
				s.Characters[i].Learned = append(s.Characters[i].Learned, knock)
				found = true
			}
		}
	})
	if !found {
		return fmt.Errorf("enemy %s has no character state", enemy)
	}
	g.zoom, g.zoomFrom, g.zoomTo = 1.5, 1.5, 1.5
	d.step = func(f int) {
		if f == 30 {
			g.confirm(core.Command{Kind: "attack", Actor: ally, Target: enemy})
		}
	}
	return nil
}

// demoLeech has a wounded first ally with a life-steal trait strike an enemy beside it (frame 30).
func demoLeech(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	ally, enemy, err := pairFree(d)
	if err != nil {
		return err
	}
	leech := traitWith(g, content.FxLifeSteal)
	if leech == "" {
		return fmt.Errorf("no life-steal trait")
	}
	g.S.Data.Rules.HitFloor, g.S.Data.Rules.HitCeil = 100, 100
	g.S.Data.Rules.BasicAttack.Coeff *= .1
	d.edit(func(s *core.State) {
		for i := range s.Characters {
			if s.Characters[i].ID == ally {
				s.Characters[i].Learned = append(s.Characters[i].Learned, leech)
			}
		}
	})
	d.setHP(ally, .4)
	g.sync(g.S.View())
	g.zoom, g.zoomFrom, g.zoomTo = 1.5, 1.5, 1.5
	d.step = func(f int) {
		if f == 30 {
			g.confirm(core.Command{Kind: "attack", Actor: ally, Target: enemy})
		}
	}
	return nil
}

// demoShopGain opens the shop as demoShop does, then picks a piece of equipment that strengthens someone
// (frame open+30) and one that strengthens nobody (open+90).
func demoShopGain(d *Demo) error {
	if err := demoShop(d); err != nil {
		return err
	}
	inner, g, open := d.step, d.g, 0
	d.step = func(f int) {
		if open == 0 {
			inner(f)
			if g.overlay() == "shop" {
				open = f
			}
			return
		}
		v := &g.shopView
		pick := func(gain bool) {
			shop := g.S.Engine.ExploreFile().Areas[g.S.View().Spot.Area].People[g.shopFrom].Shop
			for _, r := range g.shopRows(g.S.View(), shop, false) {
				if _, ok := g.S.Data.Equipment[r.id]; ok && (len(g.S.Engine.EquipmentUpgradesFor(r.id)) > 0) == gain {
					v.pick, v.n = r.id, 1
					return
				}
			}
		}
		switch f - open {
		case 30:
			pick(true)
		case 90:
			pick(false)
		}
	}
	return nil
}
