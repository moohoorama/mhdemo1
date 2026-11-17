package autoplay

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"

	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

// FrontierStart is captured before preparation during a genuine, undefeated new-game prefix.
type FrontierStart struct {
	Stage  string
	Seed   uint64
	Prefix int
	State  core.State
}

func CollectFrontier(d *content.Data, m *BattleModel, seed uint64, stage string) (*FrontierStart, Report, error) {
	p := &NeuralPlayer{Searcher: CampaignPlayer(d), Model: m}
	var found *FrontierStart
	prefix := 0
	r, err := Run(d, p, Options{Difficulty: "hard", Seed: seed, Tries: 1, BeforeBattle: func(e *core.Engine) {
		id := d.Stages[e.Observe().Stage].ID
		if id == stage {
			found = &FrontierStart{Stage: id, Seed: seed, Prefix: prefix, State: e.Snapshot()}
		}
		prefix++
	}})
	return found, r, err
}

// AddStageExpert copies the current common policy; it does not change any other stage's weights.
func (m *BattleModel) AddStageExpert(stage string) error {
	if m.StageModels[stage] != nil {
		return nil
	}
	copy := *m
	copy.StageModels = nil
	copy.Preparation = nil
	b, err := json.Marshal(copy)
	if err != nil {
		return err
	}
	var n BattleModel
	if err = json.Unmarshal(b, &n); err != nil {
		return err
	}
	if m.StageModels == nil {
		m.StageModels = map[string]*BattleModel{}
	}
	m.StageModels[stage] = &n
	return nil
}

func (m *BattleModel) EnableStagePreparation(stage string) error {
	if err := m.AddStageExpert(stage); err != nil {
		return err
	}
	if m.StageModels[stage].Preparation != nil {
		return nil
	}
	if m.Preparation == nil {
		return fmt.Errorf("no learned preparation to copy")
	}
	b, err := json.Marshal(m.Preparation)
	if err != nil {
		return err
	}
	var p BattleModel
	if err = json.Unmarshal(b, &p); err != nil {
		return err
	}
	m.StageModels[stage].Preparation = &p
	return nil
}

type FrontierTeacher struct {
	K         int
	Risk, XP  float64
	Threat    string
	Explore   float64
	LordLast  bool
	Global    bool
	DuelFirst bool
	LordDelay int
}

type FrontierOptions struct {
	Training                    bool
	Teacher                     *FrontierTeacher
	Temperature, ProgressReward float64
}

func PlayFrontier(d *content.Data, m *BattleModel, start FrontierStart, seed uint64, opt FrontierOptions) (CampaignEpisode, error) {
	out := CampaignEpisode{Samples: map[string][]BattleSample{}}
	e, err := core.Restore(d, start.State)
	if err != nil {
		return out, err
	}
	if e.Observe().Phase != "scenario" || d.Stages[e.Observe().Stage].ID != start.Stage {
		return out, fmt.Errorf("invalid real campaign frontier")
	}
	teacher := opt.Teacher
	temperature := opt.Temperature
	if temperature <= 0 {
		temperature = .3
	}
	p := &NeuralPlayer{Searcher: CampaignPlayer(d), Model: m, BattleTeacher: teacher != nil, WarmPreparation: teacher != nil, BattleTemperature: temperature}
	if teacher != nil {
		p.Searcher.K = max(1, teacher.K)
		p.Searcher.G[gW], p.Searcher.G[gXP] = teacher.Risk, teacher.XP
		p.TeacherExplore, p.TeacherLordLast = teacher.Explore, teacher.LordLast
		p.TeacherGlobal = teacher.Global
		p.TeacherDuelFirst, p.TeacherLordDelay = teacher.DuelFirst, teacher.LordDelay
		if teacher.Threat != "" {
			p.TeacherEval = &ai.Eval{N: .5, M: 1, Threat: teacher.Threat, W: teacher.Risk, XP: teacher.XP, Grow: Grow(d, e)}
		}
	}
	if opt.Training || teacher != nil {
		p.RNG = rand.New(rand.NewPCG(seed, 2))
	}
	p.Record = func(key string, s BattleSample) { out.Samples[key] = append(out.Samples[key], s) }
	p.Prepare(e, 1)
	if p.err != nil {
		return out, p.err
	}
	if _, err = e.Apply(core.Command{Kind: "start"}); err != nil {
		return out, err
	}
	if err = Battle(e, p, 20000); err != nil {
		return out, err
	}
	if p.err != nil {
		return out, p.err
	}
	won := e.Observe().Result == "victory"
	if teacher != nil {
		s := e.Snapshot()
		out.FinalState = &s
	}
	out.Report = Report{Player: "frontier", Seed: start.Seed, ContentHash: d.Hash, Stages: []StageReport{{Stage: start.Stage, Tries: 1, Won: won, FirstWin: won, Rounds: e.Observe().Round}}}
	value := float64(start.Prefix) + b2f(won)
	if !won && opt.Training {
		value += min(.25, max(0, opt.ProgressReward)) * frontierProgress(e)
	}
	for key, rows := range out.Samples {
		for i := range rows {
			rows[i].Return = value
		}
		out.Samples[key] = rows
	}
	return out, nil
}
