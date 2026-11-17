package autoplay

import (
	"fmt"
	"slices"

	"srpg/internal/content"
	"srpg/internal/core"
)

var prepEffects = []string{"move", "long_reach", "diagonal", "crit_always", "crit_boost", "double_always", "counter_boost", "retaliate", "assist", "opening", "physical", "magic", "heal", "item_share", "item_power", "mp_cost", "mp_recovery", "defense", "mind", "damage_reduction"}

func prepSignature(d *content.Data, v core.UnitView) []float64 {
	x := make([]float64, 32)
	s := statsVec(v.Stats)
	for i := range s {
		x[i] = s[i] / 1000
	}
	for _, id := range v.Traits {
		for i, effect := range prepEffects {
			x[6+i] += d.Traits[id].Effects[effect]
		}
	}
	least := 10000
	for _, id := range v.Skills {
		sk := d.Skills[id]
		x[26] += .1
		if sk.Cost > 0 {
			x[27] += .1
			least = min(least, sk.Cost)
		}
		x[29] += .1 * b2f(sk.Kind == content.KindHeal)
		x[30] += .1 * b2f(sk.Kind == content.KindMagic)
		x[31] += .1 * b2f(sk.Kind == content.KindBuff)
	}
	if least != 10000 {
		x[28] = float64(least) / 100
	}
	return x
}

func preparationState(d *content.Data, e *core.Engine, id string) []float64 {
	o := e.Observe()
	x := make([]float64, BattleStateLen)
	if o.Stage >= 0 && o.Stage < len(d.Stages) {
		st := d.Stages[o.Stage]
		x[0], x[1], x[2] = float64(st.Order)/float64(max(1, len(d.Stages))), float64(len(st.Enemies))/50, float64(st.TurnLimit)/50
	}
	x[3], x[4], x[5] = float64(len(o.Characters))/30, float64(o.Money)/20000, float64(len(o.Deployment))/20
	for item, n := range o.Inventory {
		if d.Items[item].Effect == content.ItemHP {
			x[6] += float64(n*d.Items[item].Amount) / 10000
		}
		if d.Items[item].Effect == content.ItemMP {
			x[7] += float64(n*d.Items[item].Amount) / 1000
		}
	}
	c := charOf(e, id)
	v, _ := e.CharacterView(id)
	x[8], x[9], x[10], x[11] = float64(c.Level)/50, float64(c.Points)/2000, b2f(slices.Contains(o.Deployment, id)), b2f(d.Characters[c.Def].Lord)
	x[12], x[13] = float64(len(c.Learned))/20, float64(d.Classes[c.Class].Tier)/3
	x[14], x[15] = float64(d.Classes[c.Class].Move)/10, float64(d.Classes[c.Class].Range)/10
	copy(x[16:48], prepSignature(d, v))
	for _, other := range Officers(d, e) {
		w, err := e.CharacterView(other.ID)
		if err != nil {
			continue
		}
		x[48] += float64(other.Level) / 100
		x[49] += float64(other.Points) / 10000
		x[50] += float64(max(w.Stats.Attack, w.Stats.Mind)) / 10000
		for _, skill := range w.Skills {
			if d.Skills[skill].Cost > 0 {
				x[51] += .01
			}
		}
	}
	x[52] = b2f(o.Phase == "battle")
	return x
}

// PreparationInputs offers only engine-accepted changes plus the explicit keep/save decision.
func PreparationInputs(d *content.Data, e *core.Engine, id, slot string, release bool) BattleChoices {
	c := charOf(e, id)
	in := BattleChoices{State: preparationState(d, e, id), Commands: []core.Command{{Kind: "skip", Actor: id, Option: slot}}}
	var commands []core.Command
	if slot == "" {
		for _, trait := range e.Learnable(id) {
			if cost := d.Traits[trait].Cost; cost > 0 && cost <= c.Points {
				commands = append(commands, core.Command{Kind: "learn", Actor: id, Trait: trait})
			}
		}
	} else if release {
		if item := c.Equipment[slot]; item != "" {
			commands = append(commands, core.Command{Kind: "unequip", Actor: id, Item: item})
		}
	} else {
		for _, item := range sortedKeys(e.Observe().Warehouse) {
			if e.Observe().Warehouse[item] > 0 && d.Equipment[item].Slot == slot && c.Equipment[slot] != item {
				commands = append(commands, core.Command{Kind: "equip", Actor: id, Item: item})
			}
		}
	}
	before, _ := e.CharacterView(id)
	base := prepSignature(d, before)
	add := func(cmd core.Command, after core.UnitView) {
		a := make([]float64, BattleActionLen)
		kind := slices.Index([]string{"skip", "learn", "equip", "unequip"}, cmd.Kind)
		if kind >= 0 {
			a[kind] = 1
		}
		if i := slices.Index([]string{"weapon", "armor", "accessory"}, slot); i >= 0 {
			a[4+i] = 1
		}
		if cmd.Kind == "learn" {
			t := d.Traits[cmd.Trait]
			a[7], a[8] = float64(t.Cost)/2000, float64(c.Points-t.Cost)/2000
			if i := slices.Index(tones, t.Tone); i >= 0 {
				a[9+i] = 1
			}
			if i := slices.Index(kinds, t.Kind); i >= 0 {
				a[16+i] = 1
			}
			if i := slices.Index(d.Characters[c.Def].Traits, cmd.Trait); i >= 0 {
				a[19] = 1 / (1 + float64(i))
			}
		}
		if cmd.Item != "" {
			a[20], a[21] = float64(d.Equipment[cmd.Item].Price)/10000, b2f(d.Equipment[cmd.Item].Owner == c.Def)
		}
		changed := prepSignature(d, after)
		for i := range changed {
			a[24+i] = changed[i] - base[i]
		}
		a[56], a[57], a[58] = changed[26], changed[27], changed[28]
		in.Features = append(in.Features, append(slices.Clone(in.State), a...))
	}
	add(in.Commands[0], before)
	snap := e.Snapshot()
	for _, cmd := range commands {
		copy, err := core.Restore(d, snap)
		if err != nil {
			continue
		}
		if _, err = copy.Apply(cmd); err != nil {
			continue
		}
		after, err := copy.CharacterView(id)
		if err != nil {
			continue
		}
		in.Commands = append(in.Commands, cmd)
		add(cmd, after)
	}
	return in
}

func (p *NeuralPlayer) preparationModel(e *core.Engine) (*BattleModel, string) {
	if index := e.Observe().Stage; index >= 0 && index < len(p.D.Stages) {
		stage := p.D.Stages[index].ID
		if expert := p.Model.StageModels[stage]; expert != nil && expert.Preparation != nil {
			return expert.Preparation, "prep:" + stage
		}
	}
	return p.Model.Preparation, "prep"
}

func (p *NeuralPlayer) prepChoice(e *core.Engine, id, slot string, release bool) core.Command {
	in := PreparationInputs(p.D, e, id, slot, release)
	m, key := p.preparationModel(e)
	if !p.WarmPreparation {
		return p.pick(m, key, in, 1)
	}
	best := 0
	if slot == "" {
		want := p.G.bestTrait(p.D, e, id)
		for i, c := range in.Commands {
			if c.Trait == want && c.Kind == "learn" {
				best = i
			}
		}
	} else if !release {
		current := charOf(e, id).Equipment[slot]
		score := 0.0
		if current != "" {
			score = p.G.gearScore(p.D.Equipment[current])
		}
		for i, c := range in.Commands {
			if c.Kind == "equip" {
				if s := p.G.gearScore(p.D.Equipment[c.Item]); s > score {
					best, score = i, s
				}
			}
		}
	}
	if p.Record != nil {
		p.Record(key, BattleSample{Inputs: in, Chosen: best})
	}
	return in.Commands[best]
}

func (p *NeuralPlayer) Prepare(e *core.Engine, try int) { p.PrepareCommands(e, try) }

func (p *NeuralPlayer) PrepareCommands(e *core.Engine, try int) []core.Command {
	if m, _ := p.preparationModel(e); m == nil {
		return p.Searcher.PrepareCommands(e, try)
	}
	var commands []core.Command
	apply := func(c core.Command) bool {
		if p.err != nil || p.stopped() {
			return false
		}
		if _, err := e.Apply(c); err != nil {
			p.err = fmt.Errorf("neural preparation %+v: %w", c, err)
			return false
		}
		commands = append(commands, c)
		return true
	}
	p.G.prepareBase(p.D, e, apply)
	for _, release := range []bool{true, false} {
		for _, c := range Officers(p.D, e) {
			if isDeputy(e, c.ID) {
				continue
			}
			for _, slot := range []string{"weapon", "armor", "accessory"} {
				cmd := p.prepChoice(e, c.ID, slot, release)
				if cmd.Kind != "skip" && !apply(cmd) {
					return commands
				}
			}
		}
	}
	for _, c := range Officers(p.D, e) {
		for range 10 {
			if !e.CanLearn(c.ID) {
				break
			}
			cmd := p.prepChoice(e, c.ID, "", false)
			if cmd.Kind == "skip" {
				break
			}
			if !apply(cmd) {
				return commands
			}
		}
	}
	return commands
}

func (p *NeuralPlayer) Learn(e *core.Engine) core.Command {
	if m, _ := p.preparationModel(e); m == nil {
		return p.Searcher.Learn(e)
	}
	id := e.Observe().Learning[0]
	c := p.prepChoice(e, id, "", false)
	if c.Kind == "skip" {
		return core.Command{Kind: "learn_close", Actor: id}
	}
	return c
}
