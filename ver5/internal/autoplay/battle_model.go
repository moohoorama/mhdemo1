package autoplay

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"sort"

	"srpg/internal/ai"
	"srpg/internal/core"
)

// BattleModel scores legal commands and predicts the return of the rest of the battle.
type BattleModel struct {
	Version           int
	Actor, Value      *MLP
	Episodes          int
	CampaignEpisodes  int `json:",omitempty"`
	ContentHash       string
	SourceContentHash string `json:",omitempty"`
	Training          []string
	Objective         string                  `json:",omitempty"`
	StageModels       map[string]*BattleModel `json:",omitempty"`
	Preparation       *BattleModel            `json:",omitempty"`
	RichInputs        bool                    `json:",omitempty"`
	GoalInputs        bool                    `json:",omitempty"`
	EffectInputs      bool                    `json:",omitempty"`
}

func NewBattleModel(hidden int, seed uint64) *BattleModel {
	net := func(in int, seed uint64) *MLP {
		m := NewMLP([]int{in, hidden, hidden, 1}, seed)
		m.InMean, m.InStd = make([]float64, in), make([]float64, in)
		for i := range m.InStd {
			m.InStd[i] = 1
		}
		m.OutMean, m.OutStd = []float64{0}, []float64{1}
		return m
	}
	return &BattleModel{Version: BattleFeatureVersion, Actor: net(BattleStateLen+BattleActionLen, seed), Value: net(BattleStateLen, seed+1)}
}

func probabilities(logits []float64) []float64 {
	if len(logits) == 0 {
		return nil
	}
	hi := logits[0]
	for _, v := range logits {
		hi = max(hi, v)
	}
	p, sum := make([]float64, len(logits)), 0.0
	for i, v := range logits {
		p[i] = math.Exp(v - hi)
		sum += p[i]
	}
	for i := range p {
		p[i] /= sum
	}
	return p
}

func (m *BattleModel) policy(rows [][]float64) []float64 {
	return m.policyAt(rows, 1)
}

func (m *BattleModel) policyAt(rows [][]float64, temperature float64) []float64 {
	if temperature <= 0 {
		temperature = 1
	}
	logits := make([]float64, len(rows))
	for i, row := range rows {
		logits[i] = m.Actor.Predict(row)[0] / temperature
	}
	return probabilities(logits)
}

func sampleChoice(p []float64, rng *rand.Rand) int {
	left := rng.Float64()
	for i, q := range p {
		left -= q
		if left <= 0 {
			return i
		}
	}
	return len(p) - 1
}

func (m *BattleModel) selectChoice(in BattleChoices, rng *rand.Rand) (int, float64) {
	return m.selectAt(in, rng, 1)
}

func (m *BattleModel) selectAt(in BattleChoices, rng *rand.Rand, temperature float64) (int, float64) {
	p := m.policyAt(in.Features, temperature)
	best := 0
	if rng != nil {
		best = sampleChoice(p, rng)
	} else {
		for i := range p {
			if p[i] > p[best] {
				best = i
			}
		}
	}
	return best, math.Log(max(1e-30, p[best]))
}

// Select chooses a unit and its command; terminal wins receive no hard-coded override.
func (m *BattleModel) Select(e *core.Engine) core.Command {
	m = m.stageModel(e)
	in := m.inputs(e.Data(), e)
	if len(in.Commands) == 0 {
		return core.Command{Kind: "end"}
	}
	i, _ := m.selectChoice(in, nil)
	return in.Commands[i]
}

func (m *BattleModel) stageModel(e *core.Engine) *BattleModel {
	if len(m.StageModels) > 0 {
		if specific := m.StageModels[e.Data().Stages[e.Observe().Stage].ID]; specific != nil {
			return specific
		}
	}
	return m
}

func (m *BattleModel) Save(path string) error {
	if err := m.Validate(); err != nil {
		return err
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func LoadBattleModel(path string) (*BattleModel, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m BattleModel
	if err = json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, m.Validate()
}

func (m *BattleModel) Validate() error {
	if m.Version != BattleFeatureVersion {
		return fmt.Errorf("unsupported battle feature version %d", m.Version)
	}
	if m.Objective != "" && m.Objective != "growth" && m.Objective != "campaign" {
		return fmt.Errorf("unsupported battle objective %q", m.Objective)
	}
	if child := m.Preparation; child != nil {
		if child.Preparation != nil || child.GoalInputs || child.EffectInputs || len(child.StageModels) > 0 || child.ContentHash != m.ContentHash {
			return fmt.Errorf("invalid preparation model")
		}
		if err := child.Validate(); err != nil {
			return fmt.Errorf("preparation model: %w", err)
		}
	}
	for stage, child := range m.StageModels {
		if stage == "" || child == nil || len(child.StageModels) > 0 || child.ContentHash != m.ContentHash {
			return fmt.Errorf("invalid stage model %q", stage)
		}
		if err := child.Validate(); err != nil {
			return fmt.Errorf("stage model %s: %w", stage, err)
		}
	}
	actorLen := BattleStateLen + BattleActionLen
	if m.GoalInputs {
		actorLen += BattleGoalLen
	}
	if m.EffectInputs {
		actorLen += BattleEffectLen
	}
	for _, spec := range []struct {
		net *MLP
		in  int
	}{{m.Actor, actorLen}, {m.Value, BattleStateLen}} {
		n := spec.net
		if n == nil || len(n.Sizes) != 4 || n.Sizes[0] != spec.in || n.Sizes[3] != 1 || len(n.W) != 3 || len(n.B) != 3 || len(n.InMean) != spec.in || len(n.InStd) != spec.in || len(n.OutMean) != 1 || len(n.OutStd) != 1 {
			return fmt.Errorf("invalid battle network dimensions")
		}
		for i := range n.W {
			if n.Sizes[i+1] < 1 || n.Sizes[i+1] > 256 || len(n.W[i]) != n.Sizes[i]*n.Sizes[i+1] || len(n.B[i]) != n.Sizes[i+1] {
				return fmt.Errorf("invalid battle layer %d", i)
			}
		}
		for _, group := range [][][]float64{n.W, n.B, {n.InMean, n.InStd, n.OutMean, n.OutStd}} {
			for _, row := range group {
				for _, v := range row {
					if math.IsNaN(v) || math.IsInf(v, 0) {
						return fmt.Errorf("nonfinite battle weight")
					}
				}
			}
		}
		for _, row := range [][]float64{n.InStd, n.OutStd} {
			for _, v := range row {
				if v <= 0 {
					return fmt.Errorf("invalid battle normalization")
				}
			}
		}
	}
	return nil
}

// NeuralPlayer uses the learned policies with the existing campaign and exploration handlers.
type NeuralPlayer struct {
	*Searcher
	Model             *BattleModel
	RNG               *rand.Rand
	Record            func(string, BattleSample)
	WarmPreparation   bool
	BattleTemperature float64
	BattleTeacher     bool
	TeacherEval       *ai.Eval
	TeacherExplore    float64
	TeacherLordLast   bool
	TeacherGlobal     bool
	TeacherDuelFirst  bool
	TeacherLordDelay  int
	err               error
}

func (p *NeuralPlayer) Name() string { return "neural" }
func (p *NeuralPlayer) Act(e *core.Engine, actor string) core.Command {
	m := p.Model.stageModel(e)
	in := m.inputs(p.D, e)
	if len(in.Commands) == 0 {
		return core.Command{Kind: "end"}
	}
	key := ""
	if m != p.Model {
		key = p.D.Stages[e.Observe().Stage].ID
	}
	if p.BattleTeacher {
		if p.TeacherLordLast {
			for _, u := range e.Observe().UnitViews {
				if mine(u) && u.HP > 0 && !u.Done && !p.D.Characters[u.Def].Lord {
					actor = u.ID
					break
				}
			}
		}
		if p.TeacherDuelFirst {
			nearest := int(^uint(0) >> 1)
			o := e.Observe()
			for _, u := range o.UnitViews {
				if !mine(u) || u.HP <= 0 || u.Done {
					continue
				}
				for _, v := range o.UnitViews {
					if v.HP > 0 && e.DuelPending(u.ID, v.ID) {
						if distance := abs(u.X-v.X) + abs(u.Y-v.Y); distance < nearest {
							actor, nearest = u.ID, distance
						}
					}
				}
			}
		}
		want := p.Searcher.Act(e, actor)
		if p.TeacherEval != nil {
			ranked := ai.Rank(e, actor, p.TeacherEval, true)
			if p.TeacherGlobal {
				ranked = nil
				for _, u := range e.Observe().UnitViews {
					if mine(u) && u.HP > 0 && !u.Done {
						r := ai.Rank(e, u.ID, p.TeacherEval, true)
						ranked = append(ranked, r[:min(3, len(r))]...)
					}
				}
				sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })
			}
			if e.Observe().Round <= p.TeacherLordDelay {
				var kept []ai.Scored
				for _, r := range ranked {
					lord := false
					for _, u := range e.Observe().UnitViews {
						if u.ID == r.Command.Actor {
							lord = p.D.Characters[u.Def].Lord
							break
						}
					}
					if !lord || r.Command.Kind != "move" {
						kept = append(kept, r)
					}
				}
				ranked = kept
			}
			if len(ranked) > 0 {
				i := 0
				if p.RNG != nil && p.RNG.Float64() < p.TeacherExplore {
					i = p.RNG.IntN(min(3, len(ranked)))
				}
				want = ranked[i].Command
			}
		}
		for i, c := range in.Commands {
			if c.Kind == want.Kind && c.Actor == want.Actor && c.Target == want.Target && c.Skill == want.Skill && c.Item == want.Item && c.X == want.X && c.Y == want.Y {
				if p.Record != nil {
					p.Record(key, BattleSample{Inputs: in, Chosen: i})
				}
				return c
			}
		}
		p.err = fmt.Errorf("teacher command is not a legal candidate: %+v", want)
		return want
	}
	return p.pick(m, key, in, p.BattleTemperature)
}

func (p *NeuralPlayer) pick(m *BattleModel, key string, in BattleChoices, temperature float64) core.Command {
	if temperature <= 0 {
		temperature = 1
	}
	i, logp := m.selectAt(in, p.RNG, temperature)
	if p.Record != nil {
		p.Record(key, BattleSample{Inputs: in, Chosen: i, LogProb: logp, Temperature: temperature})
	}
	return in.Commands[i]
}

func (p *NeuralPlayer) AutoplayError() error { return p.err }
