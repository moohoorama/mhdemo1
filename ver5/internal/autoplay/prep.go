package autoplay

import (
	"math"
	"slices"
	"sort"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// Genome is the weights of the rule player: how it picks traits, promotion classes, gear and deputies, and the
// weights of its battle evaluator. Every rule reads only what the data says of a trait, class or piece (its tone,
// kind, cost, bonuses), never its name, so the rules still hold when the balance changes.
type Genome [GenomeLen]float64

const (
	gTone      = 0  // 7: attack guard craft buff debuff heal move
	gKind      = 7  // 3: talent strategy tactic
	gRecommend = 10 // a trait the officer's own plan lists
	gCost      = 11 // per 1000 points of cost
	gBuy       = 12 // a trait is bought only when it scores above this
	gStat      = 13 // 6: attack defense mind agility morale hp, for gear and classes
	gMove      = 19 // a class's move
	gRange     = 20 // a class's range
	gDeputy    = 21 // a deputy is named when its blend gains more than this share of the officer's power
	gN         = 22 // battle evaluator N
	gM         = 23 // battle evaluator M
	gW         = 24 // battle evaluator W
	gXP        = 25 // battle evaluator XP
	gGearTrait = 26 // a piece's traits and skills, each
	gReserve   = 27 // at least .5: save points for the best trait, even when not yet affordable
	GenomeLen  = 28
)

var tones = []string{"attack", "guard", "craft", "buff", "debuff", "heal", "move"}
var kinds = []string{"talent", "strategy", "tactic"}

// DefaultGenome is a plain start: every trait worth buying, the plan's first, gear by attack and mind.
func DefaultGenome() Genome {
	var g Genome
	for i := range tones {
		g[gTone+i] = 1
	}
	g[gRecommend] = 2
	g[gBuy] = 0
	copy(g[gStat:], []float64{1, .6, 1, .4, .2, .3})
	g[gMove], g[gRange] = .5, .5
	g[gDeputy] = .02
	g[gN], g[gM], g[gW], g[gXP] = .5, 1, 1, 0
	g[gGearTrait] = 3
	g[gReserve] = 1
	return g
}

// Eval is the battle evaluator the genome plays with, its experience steered to the officers that count most.
func (g *Genome) Eval(d *content.Data, e *core.Engine) *ai.Eval {
	return &ai.Eval{N: max(.05, g[gN]), M: max(0, g[gM]), Threat: "officers-lethal", W: max(0, g[gW]), XP: max(0, g[gXP]), Grow: Grow(d, e)}
}

// traitScore is what the genome thinks of officer c learning trait t.
func (g *Genome) traitScore(d *content.Data, c core.Character, t string) float64 {
	tr := d.Traits[t]
	s := g[gCost] * float64(tr.Cost) / 1000
	if i := slices.Index(tones, tr.Tone); i >= 0 {
		s += g[gTone+i]
	}
	if i := slices.Index(kinds, tr.Kind); i >= 0 {
		s += g[gKind+i]
	}
	if plan := d.Characters[c.Def].Traits; slices.Contains(plan, t) {
		s += g[gRecommend] * (1 - .15*float64(slices.Index(plan, t)))
	}
	return s
}

// bestTrait is the trait the officer buys now, or "" to keep the points.
func (g *Genome) bestTrait(d *content.Data, e *core.Engine, id string) string {
	var c core.Character
	for _, x := range e.Observe().Characters {
		if x.ID == id {
			c = x
		}
	}
	best, score := "", g[gBuy]
	for _, t := range e.Learnable(id) {
		if d.Traits[t].Cost < 1 || d.Traits[t].Cost > c.Points && g[gReserve] < .5 {
			continue
		}
		if s := g.traitScore(d, c, t); s > score {
			best, score = t, s
		}
	}
	if d.Traits[best].Cost > c.Points {
		return "" // keep saving instead of spending the next trait's points on a cheaper substitute
	}
	return best
}

// Learn answers a learning window: the first waiting officer buys its best trait, or the window closes.
func (g *Genome) Learn(d *content.Data, e *core.Engine) core.Command {
	o := e.Observe()
	id := o.Learning[0]
	if t := g.bestTrait(d, e, id); t != "" {
		return core.Command{Kind: "learn", Actor: id, Trait: t}
	}
	return core.Command{Kind: "learn_close", Actor: id}
}

// statScore weighs six bonuses (attack defense mind agility morale hp).
func (g *Genome) statScore(b [6]float64) float64 {
	s := 0.0
	for i, v := range b {
		s += g[gStat+i] * v
	}
	return s
}

func statsVec(s core.Stats) [6]float64 {
	return [6]float64{float64(s.Attack), float64(s.Defense), float64(s.Mind), float64(s.Agility), float64(s.Morale), float64(s.MaxHP) / 10}
}

func (g *Genome) classScore(cl content.Class) float64 {
	return g.statScore(cl.Bonus) + g[gMove]*float64(cl.Move) + g[gRange]*float64(cl.Range)
}

func (g *Genome) gearScore(eq content.Equipment) float64 {
	return g.statScore(eq.Bonus) + g[gGearTrait]*float64(len(eq.Traits)+len(eq.Skills))
}

// Prepare readies the party by the genome: seals to the highest officers, owners' pieces, the rest of the gear
// by score to the highest officers, deputies where they add enough, and traits bought with the points saved.
func (g *Genome) Prepare(d *content.Data, e *core.Engine) {
	g.prepare(d, e, nil)
}

func (g *Genome) prepare(d *content.Data, e *core.Engine, record func(core.Command)) {
	apply := func(c core.Command) bool {
		_, err := e.Apply(c)
		if err == nil && record != nil {
			record(c)
		}
		return err == nil
	}
	g.prepareBase(d, e, apply)
	g.gear(d, e, apply)
	for _, c := range Officers(d, e) {
		for range 10 {
			t := g.bestTrait(d, e, c.ID)
			if t == "" || !e.CanLearn(c.ID) || !apply(core.Command{Kind: "learn", Actor: c.ID, Trait: t}) {
				break
			}
		}
	}
}

func (g *Genome) prepareBase(d *content.Data, e *core.Engine, apply func(core.Command) bool) {
	g.sortie(d, e, apply)
	g.promote(d, e, apply)
	for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() {
		if !apply(c) {
			break
		}
	}
	g.deputies(d, e, apply)
}

func (g *Genome) sortie(d *content.Data, e *core.Engine, apply func(core.Command) bool) {
	o := e.Observe()
	st := d.Stages[o.Stage]
	var ids []string
	party := Officers(d, e)
	for _, c := range party {
		if slices.Contains(st.Required, c.ID) || d.Characters[c.Def].Lord {
			ids = append(ids, c.ID)
		}
	}
	for _, c := range party {
		if len(ids) >= st.Max {
			break
		}
		if !slices.Contains(ids, c.ID) && !isDeputy(e, c.ID) && !slices.Contains(st.Reserve, c.ID) { // the reserve goes on top of the count
			ids = append(ids, c.ID)
		}
	}
	apply(core.Command{Kind: "deploy", Deployment: ids})
}

func (g *Genome) promote(d *content.Data, e *core.Engine, apply func(core.Command) bool) {
	var seals []string
	for id, it := range d.Items {
		if it.Effect == content.ItemPromote {
			seals = append(seals, id)
		}
	}
	sort.Strings(seals)
	for _, seal := range seals {
		for _, c := range Officers(d, e) {
			if e.Observe().Inventory[seal] < 1 {
				break
			}
			classes, err := e.PromotionsAway(seal, c.ID)
			if err != nil || len(classes) == 0 {
				continue
			}
			best := classes[0]
			for _, cl := range classes[1:] {
				if g.classScore(d.Classes[cl]) > g.classScore(d.Classes[best]) {
					best = cl
				}
			}
			apply(core.Command{Kind: "item", Target: c.ID, Item: seal, Option: best})
		}
	}
}

// gear gives every free piece in the warehouse to the highest officer it improves most, slot by slot.
func (g *Genome) gear(d *content.Data, e *core.Engine, apply func(core.Command) bool) {
	for _, c := range Officers(d, e) {
		if isDeputy(e, c.ID) {
			continue
		}
		for _, slot := range []string{"weapon", "armor", "accessory"} {
			o := charOf(e, c.ID)
			cur := -math.MaxFloat64
			if id := o.Equipment[slot]; id != "" {
				cur = g.gearScore(d.Equipment[id])
				if eq := d.Equipment[id]; eq.Owner == c.Def {
					continue // an owner keeps its own piece
				}
			}
			best := ""
			for _, id := range sortedKeys(e.Observe().Warehouse) {
				eq := d.Equipment[id]
				if e.Observe().Warehouse[id] < 1 || eq.Slot != slot || eq.Owner != "" && eq.Owner != c.Def {
					continue
				}
				if s := g.gearScore(eq); s > cur && d.Carries(d.Classes[o.Class], o.Learned, eq) {
					best, cur = id, s
				}
			}
			if best != "" {
				apply(core.Command{Kind: "equip", Actor: c.ID, Item: best})
			}
		}
	}
}

// deputies names, for each officer who may, the benched officer whose blend adds most, if it adds enough.
func (g *Genome) deputies(d *content.Data, e *core.Engine, apply func(core.Command) bool) {
	for _, c := range Officers(d, e) {
		if !e.CanDeputize(c.ID) || charOf(e, c.ID).Deputy != "" || !slices.Contains(e.Observe().Deployment, c.ID) {
			continue
		}
		base, _, err := e.DeputyStats(c.ID, "")
		if err != nil {
			continue
		}
		power := g.statScore(statsVec(base))
		best, gain := "", g[gDeputy]*power
		for _, x := range Officers(d, e) {
			if x.ID == c.ID || slices.Contains(e.Observe().Deployment, x.ID) || e.DeputyCheck(c.ID, x.ID) != nil {
				continue
			}
			s, _, err := e.DeputyStats(c.ID, x.ID)
			if err != nil {
				continue
			}
			if v := g.statScore(statsVec(s)) - power; v > gain {
				best, gain = x.ID, v
			}
		}
		if best != "" {
			apply(core.Command{Kind: "deputy", Actor: c.ID, Target: best})
		}
	}
}

func charOf(e *core.Engine, id string) core.Character {
	for _, c := range e.Observe().Characters {
		if c.ID == id {
			return c
		}
	}
	return core.Character{}
}

func isDeputy(e *core.Engine, id string) bool {
	for _, c := range e.Observe().Characters {
		if c.Deputy == id {
			return true
		}
	}
	return false
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// RulePlayer plays by a genome alone: its preparation rules and its evaluator in battle.
type RulePlayer struct {
	Label string
	D     *content.Data
	G     Genome
}

func (r *RulePlayer) Name() string                                  { return r.Label }
func (r *RulePlayer) Prepare(e *core.Engine, try int)               { r.G.Prepare(r.D, e) }
func (r *RulePlayer) Learn(e *core.Engine) core.Command             { return r.G.Learn(r.D, e) }
func (r *RulePlayer) Act(e *core.Engine, actor string) core.Command { return r.G.Act(r.D, e, actor) }

// Act is the evaluator's best command for the unit.
func (g *Genome) Act(d *content.Data, e *core.Engine, actor string) core.Command {
	if r := ai.Rank(e, actor, g.Eval(d, e), true); len(r) > 0 {
		return r[0].Command
	}
	return core.Command{Kind: "wait", Actor: actor}
}
