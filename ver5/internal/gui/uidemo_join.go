package gui

import (
	"srpg/internal/core"
	"srpg/internal/session"
)

func init() { demos["join"] = demoJoin }

// demoJoin plays an officer joining in the story (-node: the join node; default the last one) from the line
// before it (frame 30): the acquire window turns up the officer's card under 합류, the officer's window opens at
// 2 s after it shows, the first trait's at 3 s, both close by 4.4 s and 확인 at 5.2 s goes on to the next line.
func demoJoin(d *Demo) error {
	g := d.g
	g.request(session.Request{Op: "new", Seed: 1})
	from, last := "", ""
	for _, n := range g.S.Data.Nodes {
		if n.Kind == "join" && (d.node == "" || n.ID == d.node) {
			from = last
		}
		last = n.ID
	}
	s := g.S.Engine.Snapshot()
	s.Node = from
	e, err := core.Restore(g.S.Data, s)
	if err != nil {
		return err
	}
	g.S.Engine = e
	var m *modal
	at := map[float64]func(){
		2: func() { m.piece = m.gains[0].id },
		3: func() {
			if t := g.S.Data.Characters[m.piece].Traits; len(t) > 0 {
				m.chip = equipChip{"trait", t[0]}
			}
		},
		3.8: g.closeModal,
		4.4: g.closeModal,
		5.2: g.closeModal,
	}
	done := map[float64]bool{}
	d.step = func(f int) {
		if f == 30 {
			g.modals = nil
			g.request(session.Request{Op: "command", Command: core.Command{Kind: "next"}})
		}
		if m == nil {
			for _, x := range g.modals {
				if x.kind == "acquire" {
					m = x
				}
			}
			return
		}
		for t, do := range at {
			if !done[t] && m.t >= t {
				done[t] = true
				do()
			}
		}
	}
	return nil
}
