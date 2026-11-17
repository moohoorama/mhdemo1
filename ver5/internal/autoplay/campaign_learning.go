package autoplay

import (
	"math/rand/v2"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
)

// EnableCampaignLearning keeps the old predictions while making the additional MP-use features trainable.
func (m *BattleModel) EnableCampaignLearning(hidden int, seed uint64) {
	if m.Preparation == nil {
		m.Preparation = NewBattleModel(hidden, seed)
		m.Preparation.ContentHash = m.ContentHash
		m.Preparation.Objective = "campaign"
	}
	all := []*BattleModel{m}
	for _, child := range m.StageModels {
		all = append(all, child)
	}
	for _, n := range all {
		if !n.RichInputs {
			for row := 0; row < n.Actor.Sizes[1]; row++ {
				for _, i := range []int{5, 28, 29, 50, 57} {
					n.Actor.W[0][row*n.Actor.Sizes[0]+BattleStateLen+i] = 0
				}
			}
			n.RichInputs = true
		}
		n.Objective = "campaign"
	}
}

type CampaignEpisode struct {
	Report     Report
	Samples    map[string][]BattleSample
	Error      string      `json:",omitempty"`
	FinalState *core.State `json:",omitempty"`
}

// PlayCampaignEpisode starts with the actual campaign and stops on the first loss, without retrying.
func PlayCampaignEpisode(d *content.Data, m *BattleModel, seed uint64, training, warm bool, temperature float64) CampaignEpisode {
	out := CampaignEpisode{Samples: map[string][]BattleSample{}}
	p := &NeuralPlayer{Searcher: CampaignPlayer(d), Model: m, WarmPreparation: warm, BattleTemperature: temperature}
	if training {
		p.RNG = rand.New(rand.NewPCG(seed, 0x63616d706169676e))
	}
	if training || warm {
		p.Record = func(key string, row BattleSample) {
			if !warm || key == "prep" || strings.HasPrefix(key, "prep:") {
				out.Samples[key] = append(out.Samples[key], row)
			}
		}
	}
	r, err := Run(d, p, Options{Difficulty: "hard", Seed: seed, Tries: 1})
	out.Report = r
	if err != nil {
		out.Error = err.Error()
		out.Samples = nil
		return out
	}
	wins, _ := r.FirstLossProgress()
	for key, rows := range out.Samples {
		for i := range rows {
			rows[i].Return = float64(wins)
		}
		out.Samples[key] = rows
	}
	return out
}

type CampaignAssessment struct {
	Reports                                     []Report
	Sum, Min, Max, Completed, Attempts, Battles int
}

func (a *CampaignAssessment) Add(r Report) {
	wins, _ := r.FirstLossProgress()
	if len(a.Reports) == 0 {
		a.Min = wins
	}
	a.Min = min(a.Min, wins)
	a.Max = max(a.Max, wins)
	a.Sum += wins
	if r.Cleared {
		a.Completed++
	}
	a.Attempts++
	a.Battles += len(r.Stages)
	a.Reports = append(a.Reports, r)
}

func (a CampaignAssessment) Mean() float64 { return float64(a.Sum) / float64(max(1, a.Attempts)) }

func (a CampaignAssessment) Better(b CampaignAssessment) bool {
	if a.Mean() != b.Mean() {
		return a.Mean() > b.Mean()
	}
	if a.Min != b.Min {
		return a.Min > b.Min
	}
	return a.Completed > b.Completed
}

type CampaignTrainer struct{ Models map[string]*BattleTrainer }

func NewCampaignTrainer(m *BattleModel, seed uint64) *CampaignTrainer {
	t := &CampaignTrainer{Models: map[string]*BattleTrainer{"": NewBattleTrainer(m, seed), "prep": NewBattleTrainer(m.Preparation, seed+1)}}
	for _, key := range sortedKeys(m.StageModels) {
		t.Models[key] = NewBattleTrainer(m.StageModels[key], seed+uint64(len(t.Models)))
		if child := m.StageModels[key].Preparation; child != nil {
			t.Models["prep:"+key] = NewBattleTrainer(child, seed+uint64(len(t.Models)))
		}
	}
	return t
}

func (t *CampaignTrainer) Update(episodes []CampaignEpisode, warm bool, epochs, batch int, rate float64) {
	for _, key := range sortedKeys(t.Models) {
		var rows []BattleSample
		for _, e := range episodes {
			if e.Error == "" {
				rows = append(rows, e.Samples[key]...)
			}
		}
		if len(rows) == 0 {
			continue
		}
		if warm {
			if key == "prep" || strings.HasPrefix(key, "prep:") {
				t.Models[key].Imitate(rows, epochs, batch, rate)
			}
		} else {
			t.Models[key].Improve(rows, epochs, batch, rate)
		}
	}
}
