package gui

import (
	"fmt"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() {
	demos["acquire"] = demoAcquire
	demos["smith-gifts"] = demoSmithGifts
}

func demoSmithGifts(d *Demo) error {
	g := d.g
	if d.node == "" {
		return fmt.Errorf("smith gifts: -node names the gift talk to start from")
	}
	r := g.request(session.Request{Op: "new", Seed: 1})
	if !r.OK {
		return fmt.Errorf("smith gifts: %s", r.Error)
	}
	s := g.S.Engine.Snapshot()
	s.Node = d.node
	e, err := core.Restore(g.S.Data, s)
	if err != nil {
		return err
	}
	g.S.Engine = e
	g.S.Settings.InstantText = true
	d.step = func(f int) {
		if f == 0 || f%180 != 0 {
			return
		}
		if len(g.modals) > 0 {
			g.closeModal()
		} else {
			g.advanceDialogue()
		}
	}
	return nil
}

// demoAcquire walks the acquire window on a new game's first scene, timed from when the window shows (after the
// scene's beats): an equipment with a trait (first), an item of each use and money turn up (0–1.3 s), the equipment's window
// (2 s), its trait's (2.8 s), back to the equipment (3.6 s) and to the cards (4.3 s) without them turning up again,
// an item's window (5 s), back (6 s), then 확인 (6.7 s).
func demoAcquire(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	var m *modal
	type step struct {
		at   float64
		do   func()
		done bool
	}
	steps := []*step{
		{at: 2, do: func() { m.piece = m.gains[0].id }},
		{at: 2.8, do: func() {
			if eq := g.S.Data.Equipment[m.piece]; len(eq.Traits) > 0 {
				m.chip = equipChip{"trait", eq.Traits[0]}
			}
		}},
		{at: 3.6, do: g.closeModal},
		{at: 4.3, do: g.closeModal},
		{at: 5, do: func() { m.piece = m.gains[1].id }},
		{at: 6, do: g.closeModal},
		{at: 6.7, do: g.closeModal},
	}
	d.step = func(f int) {
		if f == 30 {
			var gains []gain
			for _, id := range sorted(g.S.Data.Equipment) {
				if len(g.S.Data.Equipment[id].Traits) > 0 {
					gains = append(gains, gain{id: id, n: 1, first: true})
					break
				}
			}
			effects := map[string]bool{}
			for _, id := range sorted(g.S.Data.Items) {
				if e := g.S.Data.Items[id].Effect; !effects[e] {
					effects[e] = true
					gains = append(gains, gain{id: id, n: 8 - 2*len(gains)})
				}
			}
			gains = append(gains, gain{id: content.Money, n: 50})
			m = &modal{kind: "acquire", gains: gains}
			g.modals = append(g.modals, m)
		}
		if m == nil {
			return
		}
		for _, s := range steps {
			if !s.done && m.t >= s.at {
				s.done = true
				s.do()
			}
		}
	}
	return nil
}
