package gui

import (
	"image"

	"srpg/internal/core"
)

func init() { demos["menu"] = demoMenu }

// demoMenu opens the command menu of an ally with skills, points at 공격, opens 병법 and
// walks the pointer down the skills, like the mockup's cursor.
func demoMenu(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	actor, _, err := d.pair()
	if err != nil {
		return err
	}
	// the mockup's skills: 초열, 탁류 and 낙석
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
	g.sync(g.S.Engine.Observe())
	point := func(label string, i int) {
		n := 0
		for _, b := range g.buttons {
			if b.label == label || label == "" {
				if n == i {
					c := b.rect.Min.Add(image.Pt(b.rect.Dx()*7/10, b.rect.Dy()/2))
					cursorPos = func() (int, int) { return c.X, c.Y }
					return
				}
				n++
			}
		}
	}
	d.step = func(f int) {
		switch f {
		case 20:
			g.begin(*g.unitView(g.S.Engine.Observe(), actor))
			g.ord.stage = "menu"
		case 60:
			point("공격", 0)
		case 100:
			point("병법", 0)
		case 115:
			for _, b := range g.buttons {
				if b.label == "병법" {
					b.click()
				}
			}
		case 150, 190, 230:
			right := 0
			for _, b := range g.buttons {
				if b.label == "병법" {
					right = b.rect.Max.X
				}
			}
			n := 0
			for _, b := range g.buttons {
				if b.rect.Min.X > right && b.rect.Max.X < uiW-160 {
					if n == (f-150)/40 {
						c := b.rect.Min.Add(image.Pt(b.rect.Dx()*7/10, b.rect.Dy()/2))
						cursorPos = func() (int, int) { return c.X, c.Y }
					}
					n++
				}
			}
		}
	}
	return nil
}
