package autoplay

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// BattleSample records the legal distribution offered to the policy and what the battle finally earned.
type BattleSample struct {
	Inputs      BattleChoices
	Chosen      int
	LogProb     float64
	Return      float64
	Temperature float64
}

type BattleStart struct {
	Stage string
	Try   int
	State core.State
}

// BattleUtility rewards experience only in a won battle. There is no reward for ending sooner.
func BattleUtility(d *content.Data, e *core.Engine, grow map[string]float64) float64 {
	if e.Observe().Result != "victory" {
		return -2
	}
	return 2 + .2*Measure(d, e, grow).XP
}

// ReadBattleExamples reads real campaign traces, verifying every command before using it as an example.
func ReadBattleExamples(d *content.Data, paths, stages []string) (starts []BattleStart, rows []BattleSample, err error) {
	for _, path := range paths {
		f, openErr := os.Open(path)
		if openErr != nil {
			return nil, nil, openErr
		}
		dec := json.NewDecoder(f)
		var e *core.Engine
		var grow map[string]float64
		var pending []BattleSample
		var start BattleStart
		selected := false
		for {
			var r struct {
				Stage    string
				Try      int
				Snapshot *core.State
				Command  core.Command
				Events   []core.Event
				Units    []core.UnitView
				Result   string
			}
			if readErr := dec.Decode(&r); readErr != nil {
				if readErr != io.EOF {
					f.Close()
					return nil, nil, readErr
				}
				break
			}
			if r.Snapshot != nil {
				selected = len(stages) == 0 || slices.Contains(stages, r.Stage)
				if !selected {
					continue
				}
				e, err = core.Restore(d, *r.Snapshot)
				if err != nil {
					f.Close()
					return nil, nil, err
				}
				start = BattleStart{r.Stage, r.Try, *r.Snapshot}
				grow, pending = Grow(d, e), nil
				if _, err = e.Apply(core.Command{Kind: "start"}); err != nil {
					f.Close()
					return nil, nil, err
				}
				continue
			}
			if !selected {
				continue
			}
			if e == nil {
				f.Close()
				return nil, nil, fmt.Errorf("trace command without prepared snapshot")
			}
			in := BattleInputs(d, e)
			for i, c := range in.Commands {
				if reflect.DeepEqual(c, r.Command) {
					pending = append(pending, BattleSample{Inputs: in, Chosen: i})
					break
				}
			}
			result, applyErr := e.Apply(r.Command)
			after := e.Observe()
			if applyErr != nil || !reflect.DeepEqual(result.Events, r.Events) || !reflect.DeepEqual(after.UnitViews, r.Units) || after.Result != r.Result {
				f.Close()
				return nil, nil, fmt.Errorf("trace %s %s differs at %+v: %v", path, d.Stages[e.Observe().Stage].ID, r.Command, applyErr)
			}
			if r.Result != "" {
				starts = append(starts, start)
				utility := BattleUtility(d, e, grow)
				for i := range pending {
					pending[i].Return = utility
				}
				if e.Observe().Result == "victory" {
					rows = append(rows, pending...)
				}
				pending = nil
			}
		}
		f.Close()
	}
	return starts, rows, nil
}

type BattleOutcome struct {
	Stage        string
	RolloutSeed  uint64
	Won          bool
	Round, Steps int
	XP, Return   float64
	Experience   map[string]int
	EnemiesLeft  int
	OthersLeft   int // surviving enemies other than the victory targets
	ItemsUsed    int // successful consumable commands by player-controlled units
	Duels        int
	DuelRounds   []int
	Error        string `json:",omitempty"`
}

// GrowthReturn rewards actual experience, clearing ordinary enemies and saving supplies, only after a win.
func (o BattleOutcome) GrowthReturn() float64 {
	if !o.Won || o.Error != "" {
		return -10
	}
	return 10 + .5*o.XP + 4/float64(1+max(0, o.OthersLeft-1)) - min(8, .75*float64(o.ItemsUsed))
}

// PlayBattleModel changes only the dice of a real preparation; training chooses stochastically, evaluation greedily.
func PlayBattleModel(d *content.Data, start BattleStart, m *BattleModel, seed uint64, training bool) ([]BattleSample, BattleOutcome) {
	out := BattleOutcome{Stage: start.Stage, RolloutSeed: seed}
	snap := start.State
	snap.RNG = seed
	e, err := core.Restore(d, snap)
	if err != nil {
		out.Error = err.Error()
		return nil, out
	}
	p := CampaignPlayer(d)
	grow := Grow(d, e)
	if _, err = e.Apply(core.Command{Kind: "start"}); err != nil {
		out.Error = err.Error()
		return nil, out
	}
	if m != nil {
		m = m.stageModel(e)
	}
	var rows []BattleSample
	rng := rand.New(rand.NewPCG(seed, 0x706f6c696379))
	for step := range 4000 {
		o := e.Observe()
		if o.Phase == "result" {
			break
		}
		var c core.Command
		switch {
		case len(o.Learning) > 0:
			c = p.Learn(e)
		case o.Phase == "deployment":
			commands, planErr := e.DeploymentPlan()
			if planErr != nil {
				err = planErr
				break
			}
			c = commands[0]
		case o.Turn == "ally":
			if m == nil {
				c, err = next(e, o, p)
				break
			}
			in := m.inputs(d, e)
			if len(in.Commands) == 0 {
				c = core.Command{Kind: "end"}
				break
			}
			var sampling *rand.Rand
			if training {
				sampling = rng
			}
			i, logp := m.selectChoice(in, sampling)
			c = in.Commands[i]
			rows = append(rows, BattleSample{Inputs: in, Chosen: i, LogProb: logp})
		default:
			c, err = ai.NextWithDuels(e, true)
		}
		if err != nil {
			break
		}
		r, applyErr := e.Apply(c)
		if applyErr != nil {
			err = applyErr
			break
		}
		if c.Kind == "item" {
			for _, u := range o.UnitViews {
				if u.ID == c.Actor && mine(u) {
					out.ItemsUsed++
					break
				}
			}
		}
		for _, ev := range r.Events {
			if ev.Kind == "duel" {
				out.Duels++
				out.DuelRounds = append(out.DuelRounds, e.Observe().Round)
			}
		}
		out.Steps = step + 1
	}
	o := e.Observe()
	if o.Merit != nil {
		out.Experience = o.Merit.XP
	}
	out.Won, out.Round, out.XP, out.Return = o.Result == "victory", o.Round, Measure(d, e, grow).XP, BattleUtility(d, e, grow)
	if err != nil {
		out.Error = err.Error()
	} else if o.Phase != "result" {
		out.Error = "episode command limit reached"
	}
	for _, u := range o.UnitViews {
		if u.Faction == "enemy" && u.HP > 0 {
			out.EnemiesLeft++
			if !slices.Contains(o.Map.VictoryTargets(), u.ID) {
				out.OthersLeft++
			}
		}
	}
	if m != nil && m.Objective == "growth" {
		out.Return = out.GrowthReturn()
	}
	for i := range rows {
		rows[i].Return = out.Return
	}
	return rows, out
}

type battleGradient struct{ w, b [][]float64 }

func newBattleGradient(n *MLP) battleGradient {
	return battleGradient{zerosLike(n.W), zerosLike(n.B)}
}

type battleAdam struct {
	mw, vw, mb, vb [][]float64
	step           int
}

func newBattleAdam(n *MLP) *battleAdam {
	return &battleAdam{mw: zerosLike(n.W), vw: zerosLike(n.W), mb: zerosLike(n.B), vb: zerosLike(n.B)}
}

func (a *battleAdam) update(n *MLP, g battleGradient, rate float64) {
	a.step++
	norm := 0.0
	for _, group := range [][][]float64{g.w, g.b} {
		for _, row := range group {
			for _, v := range row {
				norm += v * v
			}
		}
	}
	scale := 1 / max(1, math.Sqrt(norm))
	c1, c2 := 1-math.Pow(.9, float64(a.step)), 1-math.Pow(.999, float64(a.step))
	for k, params := range [][][]float64{n.W, n.B} {
		grad, mm, vv := g.w, a.mw, a.vw
		if k == 1 {
			grad, mm, vv = g.b, a.mb, a.vb
		}
		for l := range params {
			for i := range params[l] {
				v := grad[l][i] * scale
				mm[l][i] = .9*mm[l][i] + .1*v
				vv[l][i] = .999*vv[l][i] + .001*v*v
				params[l][i] -= rate * (mm[l][i] / c1) / (math.Sqrt(vv[l][i]/c2) + 1e-8)
			}
		}
	}
}

type BattleTrainer struct {
	Model        *BattleModel
	Entropy      float64
	actor, value *battleAdam
	rng          *rand.Rand
}

func NewBattleTrainer(m *BattleModel, seed uint64) *BattleTrainer {
	return &BattleTrainer{Model: m, Entropy: .01, actor: newBattleAdam(m.Actor), value: newBattleAdam(m.Value), rng: rand.New(rand.NewPCG(seed, 7))}
}

// policyGradient is the derivative of clipped PPO loss plus negative entropy. old=nil gives behavior cloning.
func policyGradient(p []float64, chosen int, advantage float64, old *float64, entropy float64) []float64 {
	weight := advantage
	if old != nil {
		ratio := math.Exp(math.Log(max(1e-30, p[chosen])) - *old)
		if advantage > 0 && ratio > 1.2 || advantage < 0 && ratio < .8 {
			weight = 0
		} else {
			weight *= ratio
		}
	}
	meanLog := 0.0
	for _, prob := range p {
		meanLog += prob * math.Log(max(1e-30, prob))
	}
	grad := make([]float64, len(p))
	for j, prob := range p {
		grad[j] = weight * (prob - b2f(j == chosen))
		grad[j] += entropy * prob * (math.Log(max(1e-30, prob)) - meanLog)
	}
	return grad
}

func (t *BattleTrainer) fit(rows []BattleSample, epochs, batch int, rate, entropy float64, imitation bool) float64 {
	if len(rows) == 0 {
		return 0
	}
	adv := make([]float64, len(rows))
	mean, variance := 0.0, 0.0
	for i, r := range rows {
		adv[i] = r.Return - t.Model.Value.Predict(r.Inputs.State)[0]
		mean += adv[i] / float64(len(rows))
	}
	for _, v := range adv {
		variance += (v - mean) * (v - mean) / float64(len(rows))
	}
	std := max(.1, math.Sqrt(variance))
	indices := make([]int, len(rows))
	for i := range indices {
		indices[i] = i
	}
	loss := 0.0
	for range epochs {
		t.rng.Shuffle(len(indices), func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })
		for from := 0; from < len(rows); from += batch {
			ids := indices[from:min(len(rows), from+batch)]
			workers := min(4, len(ids))
			ag, vg := make([]battleGradient, workers), make([]battleGradient, workers)
			losses := make([]float64, workers)
			parallel(workers, workers, func(worker int) {
				ag[worker], vg[worker] = newBattleGradient(t.Model.Actor), newBattleGradient(t.Model.Value)
				for k := worker; k < len(ids); k += workers {
					i, scale := ids[k], 1/float64(len(ids))
					r := rows[i]
					temperature := r.Temperature
					if temperature <= 0 {
						temperature = 1
					}
					p := t.Model.policyAt(r.Inputs.Features, temperature)
					losses[worker] -= math.Log(max(1e-30, p[r.Chosen])) * scale
					weight := (adv[i] - mean) / std
					old := &r.LogProb
					if imitation {
						weight, old = 1, nil
					}
					grad := policyGradient(p, r.Chosen, weight, old, entropy)
					for j, row := range r.Inputs.Features {
						acts := t.Model.Actor.forward(t.Model.Actor.norm(row))
						t.Model.Actor.backprop(acts, []float64{grad[j] * scale / temperature}, ag[worker].w, ag[worker].b)
					}
					t.Model.Value.backward(t.Model.Value.norm(r.Inputs.State), []float64{r.Return}, vg[worker].w, vg[worker].b, scale)
				}
			})
			merge := func(gs []battleGradient) battleGradient {
				g := gs[0]
				for _, other := range gs[1:] {
					for k, group := range [][][]float64{g.w, g.b} {
						from := other.w
						if k == 1 {
							from = other.b
						}
						for l := range group {
							for j := range group[l] {
								group[l][j] += from[l][j]
							}
						}
					}
				}
				return g
			}
			t.actor.update(t.Model.Actor, merge(ag), rate)
			t.value.update(t.Model.Value, merge(vg), rate)
			for _, v := range losses {
				loss += v / float64(epochs*((len(rows)+batch-1)/batch))
			}
		}
	}
	return loss
}

func (t *BattleTrainer) Imitate(rows []BattleSample, epochs, batch int, rate float64) float64 {
	return t.fit(rows, epochs, batch, rate, .001, true)
}

func (t *BattleTrainer) Improve(rows []BattleSample, epochs, batch int, rate float64) float64 {
	return t.fit(rows, epochs, batch, rate, t.Entropy, false)
}
