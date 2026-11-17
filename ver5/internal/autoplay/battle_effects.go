package autoplay

import (
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
)

const BattleEffectLen = 16

func (m *BattleModel) EnableEffectInputs() {
	if m.EffectInputs {
		return
	}
	m.extendActionInputs(BattleEffectLen, m.Actor.Sizes[0])
	m.EffectInputs = true
}

func (m *BattleModel) inputs(d *content.Data, e *core.Engine) BattleChoices {
	in := m.goalInputs(d, e)
	if !m.EffectInputs {
		return in
	}
	o := e.Observe()
	units := map[string]core.UnitView{}
	for _, u := range o.UnitViews {
		units[u.ID] = u
	}
	for i, c := range in.Commands {
		in.Features[i] = append(in.Features[i], effectInputs(d, o.Inventory, units[c.Target], c)...)
	}
	return in
}

func effectInputs(d *content.Data, inventory map[string]int, target core.UnitView, c core.Command) []float64 {
	x := make([]float64, BattleEffectLen)
	if c.Kind == "skill" {
		s := d.Skills[c.Skill]
		k := slices.Index([]string{"attack", "defense", "mind", "agility", "morale"}, s.Effect)
		if k >= 0 {
			x[k] = 1
		} else if s.Effect != "" {
			x[5] = 1
		}
		x[6], x[7], x[8], x[9], x[10] = float64(s.Duration)/10, s.Coeff, s.HPCost, s.Pierce, float64(s.Strikes)/4
		x[14], x[15] = float64(target.Status[s.Effect].Value)/100, float64(target.Status[s.Effect].Turns)/10
	}
	if c.Kind == "item" {
		it := d.Items[c.Item]
		x[11], x[12], x[13] = float64(it.Amount)/10000, float64(it.Price)/10000, float64(inventory[c.Item])/20
	}
	return x
}
