package autoplay

import (
	"math"
	"math/rand/v2"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// StateLen is the length of StateFeatures.
const StateLen = 18

// StateFeatures describe a battle state for the learned models, in the parts Value is made of and a few more,
// none of them tied to a name in the data.
func StateFeatures(d *content.Data, e *core.Engine, grow map[string]float64) []float64 {
	return Measure(d, e, grow).Features()
}

// Features is the parts as a vector.
func (p Parts) Features() []float64 {
	limit := float64(p.TurnLimit)
	if limit == 0 {
		limit = 20
	}
	return []float64{
		p.Mine.Ratio(), p.Allies.Ratio(), p.Enemy.Ratio(),
		math.Log((p.Mine.Left + .5*p.Allies.Left + 1) / (p.Enemy.Left + 1)),
		p.Vital, float64(p.Subs), float64(p.SubsTotal - p.Subs), p.XP,
		float64(p.Goal) / 20, float64(p.Round) / limit, b2f(p.TurnLimit > 0),
		p.LevelGap / 5, float64(p.Lords), float64(p.Units) / 10, float64(p.Others) / 10,
		b2f(p.Win), b2f(p.Loss), p.Allies.Full / max(1, p.Mine.Full),
	}
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// CandidateLen is the length of CandidateFeatures.
const CandidateLen = StateLen + 7

var candidateKinds = []string{"move", "attack", "skill", "item", "wait"}

// CandidateFeatures describe one candidate command: how the state changes once the unit has played it (and, after
// a move, the evaluator's action from there), its score against the best one's, and its kind.
func CandidateFeatures(d *content.Data, e *core.Engine, snap core.State, c ai.Scored, grow map[string]float64, top float64) []float64 {
	now := StateFeatures(d, e, grow)
	after := now
	if eng := After(d, snap, c.Command); eng != nil {
		after = StateFeatures(d, eng, grow)
	}
	out := make([]float64, 0, CandidateLen)
	for i := range now {
		out = append(out, after[i]-now[i])
	}
	out = append(out, (c.Score-top)/max(1, math.Abs(top)), c.Score/max(1, math.Abs(top)))
	for _, k := range candidateKinds {
		out = append(out, b2f(c.Command.Kind == k))
	}
	return out
}

// After plays the command (and after a move the evaluator's action) on a copy of the state; nil if it fails.
func After(d *content.Data, snap core.State, c core.Command) *core.Engine {
	eng, err := core.Restore(d, snap)
	if err != nil {
		return nil
	}
	if _, err := eng.Apply(c); err != nil {
		return nil
	}
	if c.Kind == "move" {
		if r := ai.Rank(eng, c.Actor, ai.Player, true); len(r) > 0 {
			eng.Apply(r[0].Command)
		}
	}
	return eng
}

// prepGenes are the genes that only shape the preparation.
var prepGenes = func() []int {
	var out []int
	for i := range gN {
		out = append(out, i)
	}
	return append(out, gGearTrait)
}()

// PrepVariants is the genome and n−1 copies with its preparation weights shaken.
func PrepVariants(g Genome, n int, seed uint64) []Genome {
	rng := rand.New(rand.NewPCG(seed, 0x5eed))
	out := []Genome{g}
	for len(out) < n {
		v := g
		for _, i := range prepGenes {
			v[i] += rng.NormFloat64() * max(.5, math.Abs(g[i]))
		}
		out = append(out, v)
	}
	return out
}
