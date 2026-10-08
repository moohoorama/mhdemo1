package core

import "srpg/internal/content"

// NewSkirmish starts the first stage of d as a rout battle outside the campaign: its Party
// deploys as allies and its Enemies as the enemy, at their officers' data levels, with no
// items or experience. The side that routs the other wins.
func NewSkirmish(d *content.Data, seed uint64) (*Engine, error) {
	if len(d.Stages) == 0 || d.Stages[0].Goal != "rout" {
		return nil, fail("InvalidStage")
	}
	s := State{Format: 1, AI: AIPolicyVersion, Difficulty: "normal", Core: CoreVersion, Rules: RulesVersion, Content: d.Version, ContentHash: d.Hash, Seed: seed, RNG: seed, Phase: "battle", Turn: "ally", Round: 1, Inventory: map[string]int{}, Warehouse: map[string]int{}, Executed: map[string]bool{}}
	e := fromState(d, s)
	st := d.Stages[0]
	for _, side := range []struct {
		faction string
		spawns  []content.Spawn
	}{{"ally", st.Party}, {"enemy", st.Enemies}} {
		for _, sp := range side.spawns {
			if _, ok := d.Officers[sp.Officer]; !ok || e.officer(sp.Officer) != nil {
				return nil, fail("InvalidOfficer")
			}
			e.addOfficer(sp.Officer)
			e.spawn(sp.Officer, side.faction, sp.X, sp.Y)
			e.unit(sp.Officer).Items = map[string]int{}
		}
	}
	return e, nil
}
