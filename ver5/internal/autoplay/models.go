package autoplay

import (
	"math/rand/v2"
	"sync"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/storage"
)

// Sample is one training row.
type Sample struct {
	X, Y  []float64
	Group int // the battle played; validation holds whole battles back
}

// BattleNodes are the battle nodes in story order.
func BattleNodes(d *content.Data) []string {
	var out []string
	for _, w := range d.Waypoints() {
		if w.Stage != "" {
			out = append(out, w.Node)
		}
	}
	return out
}

// StartAt opens a game at a battle node as the scene jump does (everyone raised to the stage level), with the
// learning windows answered by the genome. Data is gathered this way so every battle is covered, also the ones
// a player has not reached yet.
func StartAt(d *content.Data, node string, seed uint64, difficulty string, g Genome) (*core.Engine, error) {
	e, err := core.NewAt(d, seed, node, difficulty)
	if err != nil {
		return nil, err
	}
	e.SetDamagePct(storage.DefaultScalePct)
	e.SetXPPct(storage.DefaultScalePct)
	for range 200 {
		if len(e.Observe().Learning) == 0 {
			break
		}
		if _, err := e.Apply(g.Learn(d, e)); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// explorer plays by the genome but now and then takes one of the evaluator's next best commands, and records the
// state at each own unit's turn.
type explorer struct {
	RulePlayer
	rng    *rand.Rand
	eps    float64
	grow   map[string]float64
	states [][]float64
}

func (x *explorer) Act(e *core.Engine, actor string) core.Command {
	for _, u := range e.Observe().UnitViews {
		if u.ID == actor && u.Moved {
			return x.G.Act(x.D, e, actor)
		}
	}
	x.states = append(x.states, StateFeatures(x.D, e, x.grow))
	r := ai.Rank(e, actor, x.G.Eval(x.D, e), true)
	if len(r) == 0 {
		return core.Command{Kind: "wait", Actor: actor}
	}
	if x.rng.Float64() < x.eps {
		return r[x.rng.IntN(min(3, len(r)))].Command
	}
	return r[0].Command
}

// CollectValue plays every battle node once per seed with an explorer (and one of the genome's preparation
// variants) and labels each recorded state, and the state each battle starts in, with the value the battle ended at.
func CollectValue(d *content.Data, nodes []string, seeds []uint64, difficulty string, g Genome, eps float64, par int) (states, starts []Sample) {
	var mu sync.Mutex
	parallel(len(nodes)*len(seeds), par, func(job int) {
		node, seed := nodes[job%len(nodes)], seeds[job/len(nodes)]
		e, err := StartAt(d, node, seed, difficulty, g)
		if err != nil {
			return
		}
		variants := PrepVariants(g, 4, seed)
		variants[seed%4].Prepare(d, e)
		x := &explorer{RulePlayer: RulePlayer{D: d, G: g}, rng: rand.New(rand.NewPCG(seed, uint64(job))), eps: eps, grow: Grow(d, e)}
		start := StartFeatures(d, e.Snapshot(), x.grow, g)
		if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
			return
		}
		if Battle(e, x, 20000) != nil {
			return
		}
		v := Value(d, e, x.grow)
		mu.Lock()
		defer mu.Unlock()
		for _, s := range x.states {
			states = append(states, Sample{s, []float64{v}, job})
		}
		if start != nil {
			starts = append(starts, Sample{start, []float64{v}, job})
		}
	})
	return states, starts
}

// StartFeatures are the features of the state a prepared party starts the battle in, once deployed.
func StartFeatures(d *content.Data, prepared core.State, grow map[string]float64, g Genome) []float64 {
	eng, err := core.Restore(d, prepared)
	if err != nil {
		return nil
	}
	if _, err := eng.Apply(core.Command{Kind: "start"}); err != nil {
		return nil
	}
	base := &RulePlayer{D: d, G: g}
	for range 200 {
		o := eng.Observe()
		if o.Phase == "battle" || o.Phase == "result" {
			return StateFeatures(d, eng, grow)
		}
		c, err := next(eng, o, base)
		if err != nil {
			return nil
		}
		if _, err := eng.Apply(c); err != nil {
			return nil
		}
	}
	return nil
}

// CollectPolicy plays every battle node once per seed with the searcher and turns its decisions into rows for the
// policy model (state and candidate → how much better than the decision's mean the search found it) and its
// preparation trials into rows for the preparation model.
func CollectPolicy(d *content.Data, nodes []string, seeds []uint64, difficulty string, s Searcher, par int) (acts, preps []Sample) {
	var mu sync.Mutex
	parallel(len(nodes)*len(seeds), par, func(job int) {
		node, seed := nodes[job%len(nodes)], seeds[job/len(nodes)]
		e, err := StartAt(d, node, seed, difficulty, s.G)
		if err != nil {
			return
		}
		me := s
		me.Seed = seed
		me.OnDecision = func(dec Decision) {
			mean := 0.0
			for _, v := range dec.Values {
				mean += v / float64(len(dec.Values))
			}
			mu.Lock()
			defer mu.Unlock()
			for i, c := range dec.Candidates {
				acts = append(acts, Sample{append(append([]float64{}, dec.State...), c...), []float64{dec.Values[i] - mean}, job})
			}
		}
		me.OnPrep = func(t PrepTrial) {
			if t.State != nil {
				mu.Lock()
				preps = append(preps, Sample{t.State, []float64{t.Value}, job})
				mu.Unlock()
			}
		}
		me.Prepare(e, 1)
		if _, err := e.Apply(core.Command{Kind: "start"}); err != nil {
			return
		}
		Battle(e, &me, 20000)
	})
	return acts, preps
}

// Train fits a network with two hidden layers of the width given.
func Train(rows []Sample, hidden, epochs int, seed uint64) *MLP {
	X := make([][]float64, len(rows))
	Y := make([][]float64, len(rows))
	G := make([]int, len(rows))
	for i, r := range rows {
		X[i], Y[i], G[i] = r.X, r.Y, r.Group
	}
	m := NewMLP([]int{len(X[0]), hidden, hidden, len(Y[0])}, seed)
	m.Fit(X, Y, G, epochs, 64, 3e-3, seed)
	return m
}

// prepWith tries the genome's preparation variants on copies and applies the one the model rates best.
func prepWith(d *content.Data, e *core.Engine, g Genome, n int, seed uint64, rate func([]float64) float64) {
	snap := e.Snapshot()
	grow := Grow(d, e)
	variants := PrepVariants(g, n, seed)
	best, score := 0, -1e18
	seen := map[string]bool{}
	for i, v := range variants {
		eng, err := core.Restore(d, snap)
		if err != nil {
			continue
		}
		v.Prepare(d, eng)
		key := PrepKey(eng)
		if seen[key] {
			continue
		}
		seen[key] = true
		if f := StartFeatures(d, eng.Snapshot(), grow, g); f != nil {
			if s := rate(f); s > score {
				best, score = i, s
			}
		}
	}
	variants[best].Prepare(d, e)
}

// ValuePlayer plays by a value network learned from how battles ended: in battle it plays each of the
// evaluator's K best commands on a copy and keeps the one whose state the network rates best; before a battle it
// rates the starting state of each preparation variant the same way.
type ValuePlayer struct {
	Label string
	D     *content.Data
	G     Genome
	Net   *MLP
	K, P  int
}

func (v *ValuePlayer) Name() string                      { return v.Label }
func (v *ValuePlayer) Learn(e *core.Engine) core.Command { return v.G.Learn(v.D, e) }
func (v *ValuePlayer) Prepare(e *core.Engine, try int) {
	prepWith(v.D, e, v.G, v.P, uint64(e.Observe().Stage)<<8^uint64(try), func(f []float64) float64 { return v.Net.Predict(f)[0] })
}
func (v *ValuePlayer) Act(e *core.Engine, actor string) core.Command {
	for _, u := range e.Observe().UnitViews {
		if u.ID == actor && u.Moved {
			return v.G.Act(v.D, e, actor)
		}
	}
	ranked := ai.Rank(e, actor, v.G.Eval(v.D, e), true)
	if len(ranked) == 0 {
		return core.Command{Kind: "wait", Actor: actor}
	}
	cands := Candidates(ranked, v.K)
	grow := Grow(v.D, e)
	snap := e.Snapshot()
	best, score := 0, -1e18
	for i, c := range cands {
		eng := After(v.D, snap, c.Command)
		if eng == nil {
			continue
		}
		if s := v.Net.Predict(StateFeatures(v.D, eng, grow))[0]; s > score {
			best, score = i, s
		}
	}
	return cands[best].Command
}

// PolicyPlayer plays by networks that imitate the searcher: one scores each of the evaluator's K best commands
// as the search would have, one rates preparations as the searcher's trial battles did.
type PolicyPlayer struct {
	Label           string
	D               *content.Data
	G               Genome
	ActNet, PrepNet *MLP
	K, P            int
}

func (p *PolicyPlayer) Name() string                      { return p.Label }
func (p *PolicyPlayer) Learn(e *core.Engine) core.Command { return p.G.Learn(p.D, e) }
func (p *PolicyPlayer) Prepare(e *core.Engine, try int) {
	prepWith(p.D, e, p.G, p.P, uint64(e.Observe().Stage)<<8^uint64(try), func(f []float64) float64 { return p.PrepNet.Predict(f)[0] })
}
func (p *PolicyPlayer) Act(e *core.Engine, actor string) core.Command {
	for _, u := range e.Observe().UnitViews {
		if u.ID == actor && u.Moved {
			return p.G.Act(p.D, e, actor)
		}
	}
	ranked := ai.Rank(e, actor, p.G.Eval(p.D, e), true)
	if len(ranked) == 0 {
		return core.Command{Kind: "wait", Actor: actor}
	}
	cands := Candidates(ranked, p.K)
	grow := Grow(p.D, e)
	snap := e.Snapshot()
	state := StateFeatures(p.D, e, grow)
	best, score := 0, -1e18
	for i, c := range cands {
		x := append(append([]float64{}, state...), CandidateFeatures(p.D, e, snap, c, grow, ranked[0].Score)...)
		if s := p.ActNet.Predict(x)[0]; s > score {
			best, score = i, s
		}
	}
	return cands[best].Command
}
