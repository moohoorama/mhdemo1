package gui

import "srpg/internal/core"

func init() { demos["levelup"] = demoLevelUp }

// demoLevelUp starts the -stage battle with the lord two levels up and points to spend, waiting to learn as after a
// level-up: the level-up window with its trait chips and 나중에 / 특성 배우기 (frame 30), then 특성 배우기 opening
// the officer's trait tab to learn (90).
func demoLevelUp(d *Demo) error {
	if err := d.battle(d.stage); err != nil {
		return err
	}
	g := d.g
	var id string
	d.edit(func(s *core.State) {
		for i, c := range s.Characters {
			if g.S.Data.Characters[c.Def].Lord {
				s.Characters[i].Level += 2
				s.Characters[i].Points += 700
				id = c.ID
				s.Learning = append(s.Learning, c.ID)
				break
			}
		}
	})
	if id == "" {
		return nil
	}
	of := g.officer(g.S.View(), id)
	st, _ := g.S.Engine.CharacterStats(id)
	before := st
	before.MaxHP, before.Attack, before.Defense, before.Mind = st.MaxHP-48, st.Attack-21, st.Defense-14, st.Mind-9
	g.learn = g.S.View().Learners
	g.modals = []*modal{{kind: "levelup", level: levelUp{id: id, from: of.Level - 1, to: of.Level, before: before, after: st, progress: 12}}}
	cursorPos = func() (int, int) { return -1000, -1000 }
	d.step = func(f int) {
		if f == 60 && len(g.modals) > 0 {
			g.closeModal()
		}
		g.updateLearning() // the demo only advances the game; the game's Update opens the learning
	}
	return nil
}
