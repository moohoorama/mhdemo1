package autoplay

import (
	"context"
	"slices"
	"sync"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// Searcher plays by trying things out in the engine itself.
//
// In battle, at the start of each of its unit's turns, it takes the evaluator's K best commands and plays each
// on through Depth rounds (R times, each with its own dice, the same dice for every candidate), then keeps the
// one whose outcome is worth most by Value. The rest of a unit's turn
// (the action after a move) goes by the evaluator.
//
// Before each battle it builds P preparations — the genome's own and ones with its preparation weights
// shaken — plays the whole battle with each B times by the genome alone, and keeps the one that did best.
type Searcher struct {
	Label      string
	D          *content.Data
	G          Genome
	K, R       int
	Depth      int // rounds to look ahead, including the current round; 0 means 1
	P, B       int
	Par        int
	Seed       uint64
	Context    context.Context // optional cancellation for interactive play
	OnDecision func(Decision)  // when set, every searched decision is handed over (the policy model learns from them)
	OnPrep     func(PrepTrial)
}

// Decision is one searched choice: the state, the candidates and what playing each on was worth.
type Decision struct {
	State      []float64
	Candidates [][]float64
	Values     []float64
	Chosen     int
}

// PrepTrial is one preparation tried before a battle: the state it starts the battle in and what it came to.
type PrepTrial struct {
	State []float64
	Value float64
}

func (s *Searcher) Name() string { return s.Label }

func (s *Searcher) stopped() bool { return s.Context != nil && s.Context.Err() != nil }

func (s *Searcher) Learn(e *core.Engine) core.Command { return s.G.Learn(s.D, e) }

func (s *Searcher) Act(e *core.Engine, actor string) core.Command {
	if s.stopped() {
		return core.Command{Kind: "wait", Actor: actor}
	}
	o := e.Observe()
	for _, u := range o.UnitViews {
		if u.ID == actor && u.Moved {
			return s.G.Act(s.D, e, actor)
		}
	}
	ranked := ai.Rank(e, actor, s.G.Eval(s.D, e), true)
	if len(ranked) == 0 {
		return core.Command{Kind: "wait", Actor: actor}
	}
	cands := Candidates(ranked, s.K)
	if len(cands) == 1 {
		return cands[0].Command
	}
	grow := Grow(s.D, e)
	snap := e.Snapshot()
	values := make([]float64, len(cands))
	var mu sync.Mutex
	parallel(len(cands)*s.R, s.Par, func(job int) {
		i, r := job/s.R, job%s.R
		v := s.rollout(snap, uint64(r), cands[i].Command, grow)
		mu.Lock()
		values[i] += v / float64(s.R)
		mu.Unlock()
	})
	best := 0
	for i := range values {
		if values[i] > values[best] {
			best = i
		}
	}
	if s.OnDecision != nil {
		dec := Decision{Values: values, Chosen: best}
		dec.State = StateFeatures(s.D, e, grow)
		for _, c := range cands {
			dec.Candidates = append(dec.Candidates, CandidateFeatures(s.D, e, snap, c, grow, ranked[0].Score))
		}
		s.OnDecision(dec)
	}
	return cands[best].Command
}

// Candidates are the K best of the ranked commands.
func Candidates(ranked []ai.Scored, k int) []ai.Scored {
	return ranked[:min(k, len(ranked))]
}

// dice gives the r-th trial its own dice, the same for every candidate tried.
func dice(rng, r uint64) uint64 { return rng ^ (r+1)*0x9e3779b97f4a7c15 }

// rollout plays c and the lookahead rounds from the snapshot with the r-th dice and values where it ends.
func (s *Searcher) rollout(snap core.State, r uint64, c core.Command, grow map[string]float64) float64 {
	if s.stopped() {
		return -1e9
	}
	snap.RNG = dice(snap.RNG, r)
	eng, err := core.Restore(s.D, snap)
	if err != nil {
		return -1e9
	}
	if _, err := eng.Apply(c); err != nil {
		return -1e9
	}
	base := &RulePlayer{D: s.D, G: s.G}
	r0 := eng.Observe().Round
	for range 2000 {
		if s.stopped() {
			return -1e9
		}
		o := eng.Observe()
		if o.Phase == "result" || o.Turn == "ally" && o.Round >= r0+max(1, s.Depth) {
			break
		}
		cmd, err := next(eng, o, base)
		if err != nil {
			break
		}
		if _, err := eng.Apply(cmd); err != nil {
			break
		}
	}
	return Value(s.D, eng, grow)
}

// Prepare tries the preparations and keeps the best.
func (s *Searcher) Prepare(e *core.Engine, try int) {
	s.prepare(e, try, nil)
}

// PrepareCommands prepares a copy of a game and returns the accepted commands, for a GUI to play in its real session.
func (s *Searcher) PrepareCommands(e *core.Engine, try int) []core.Command {
	var commands []core.Command
	s.prepare(e, try, func(c core.Command) { commands = append(commands, c) })
	return commands
}

func (s *Searcher) prepare(e *core.Engine, try int, record func(core.Command)) {
	if s.stopped() {
		return
	}
	snap := e.Snapshot()
	genomes := PrepVariants(s.G, s.P, s.Seed^uint64(e.Observe().Stage)<<8^uint64(try))
	type trial struct {
		state core.State
		key   string
		value float64
		feat  []float64
	}
	trials := make([]trial, len(genomes))
	for i, g := range genomes {
		if s.stopped() {
			return
		}
		eng, _ := core.Restore(s.D, snap)
		g.Prepare(s.D, eng)
		trials[i] = trial{state: eng.Snapshot(), key: PrepKey(eng)}
	}
	// the same preparation reached by different weights is played once
	var uniq []int
	for i := range trials {
		if !slices.ContainsFunc(uniq, func(j int) bool { return trials[j].key == trials[i].key }) {
			uniq = append(uniq, i)
		}
	}
	if len(uniq) > 1 {
		var mu sync.Mutex
		parallel(len(uniq)*s.B, s.Par, func(job int) {
			t := &trials[uniq[job/s.B]]
			v, feat := playOut(s.D, t.state, s.G, uint64(job%s.B), s.stopped)
			mu.Lock()
			t.value += v / float64(s.B)
			t.feat = feat
			mu.Unlock()
		})
	}
	best := uniq[0]
	if s.stopped() {
		return
	}
	for _, i := range uniq {
		if trials[i].value > trials[best].value {
			best = i
		}
	}
	if s.OnPrep != nil && len(uniq) > 1 {
		for _, i := range uniq {
			s.OnPrep(PrepTrial{State: trials[i].feat, Value: trials[i].value})
		}
	}
	// carry the chosen preparation over to the real engine by replaying its weights there
	genomes[best].prepare(s.D, e, record)
}

// PlayOut starts the battle from a prepared state with the r-th dice, plays it through by the genome and
// values the end; it also gives the features of the state the battle starts in.
func PlayOut(d *content.Data, prepared core.State, g Genome, r uint64) (float64, []float64) {
	return playOut(d, prepared, g, r, func() bool { return false })
}

func playOut(d *content.Data, prepared core.State, g Genome, r uint64, stopped func() bool) (float64, []float64) {
	if stopped() {
		return -1e9, nil
	}
	prepared.RNG = dice(prepared.RNG, r)
	eng, err := core.Restore(d, prepared)
	if err != nil {
		return -1e9, nil
	}
	grow := Grow(d, eng)
	if _, err := eng.Apply(core.Command{Kind: "start"}); err != nil {
		return -1e9, nil
	}
	base := &RulePlayer{D: d, G: g}
	var feat []float64
	for range 20000 {
		if stopped() {
			return -1e9, nil
		}
		o := eng.Observe()
		if o.Phase == "result" {
			break
		}
		if feat == nil && o.Phase == "battle" {
			feat = StateFeatures(d, eng, grow)
		}
		c, err := next(eng, o, base)
		if err != nil {
			break
		}
		if _, err := eng.Apply(c); err != nil {
			break
		}
	}
	return Value(d, eng, grow), feat
}

// PrepKey tells preparations apart: classes, gear, deputies, traits and the sortie.
func PrepKey(e *core.Engine) string {
	o := e.Observe()
	k := ""
	for _, c := range o.Characters {
		k += c.ID + ":" + c.Class + ":" + c.Deputy + ":"
		for _, slot := range sortedKeys(c.Equipment) {
			k += c.Equipment[slot] + ","
		}
		k += ":"
		for _, t := range c.Learned {
			k += t + ","
		}
		k += ";"
	}
	for _, id := range o.Deployment {
		k += id + ","
	}
	return k
}

// parallel runs n jobs on up to par goroutines.
func parallel(n, par int, job func(int)) {
	if par < 1 {
		par = 1
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range min(par, n) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				job(j)
			}
		}()
	}
	for j := range n {
		jobs <- j
	}
	close(jobs)
	wg.Wait()
}
