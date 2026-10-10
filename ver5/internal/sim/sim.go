package sim

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"srpg/internal/ai"
	"srpg/internal/content"
	"srpg/internal/core"
)

type StageReport struct {
	Healing, SupportCasts                                     map[string]int
	Stage, Rounds                                             int
	Entry, Exit, Experience, Damage, EffectiveHits, SpellHits map[string]int
}
type Report struct {
	Core, Rules, Content, ContentHash, AI string
	Seed                                  uint64
	Result                                string
	Commands                              int
	Rounds                                []int
	Levels                                map[string]int
	Stages                                []StageReport
}

func Run(d *content.Data, seed uint64) (Report, error) { return RunConfigured(d, seed, nil, true) }

// playable lists the playable characters of an observation.
func playable(d *content.Data, o core.Observation) []core.Character {
	out := []core.Character{}
	for _, c := range o.Characters {
		if d.Characters[c.Def].Playable {
			out = append(out, c)
		}
	}
	return out
}

// RunConfigured plays the whole campaign with the AI on both sides. party, when given, is the deployment
// every battle uses; duels allows the stage duels.
func RunConfigured(d *content.Data, seed uint64, party []string, duels bool) (Report, error) {
	e := core.New(d, seed)
	r := Report{Core: core.CoreVersion, Rules: core.RulesVersion, Content: d.Version, ContentHash: d.Hash, AI: ai.Version, Seed: seed, Levels: map[string]int{}}
	entry := map[string]int{}
	tried := map[string]bool{}
	stageStats := StageReport{}
	record := func(events []core.Event) {
		for _, v := range events {
			switch v.Kind {
			case "experience":
				stageStats.Experience[v.Actor] += v.Amount
			case "heal":
				stageStats.Healing[v.Actor] += v.Amount
				stageStats.SupportCasts[v.Actor]++
			case "spell-hit":
				stageStats.SpellHits[v.Actor]++
			case "damage":
				if v.Amount > 0 {
					stageStats.Damage[v.Actor] += v.Amount
					stageStats.EffectiveHits[v.Actor]++
				}
			}
		}
	}
	levels := func(o core.Observation) {
		for _, c := range playable(d, o) {
			r.Levels[c.ID] = c.Level
		}
	}
	for i := 0; i < 20000; i++ {
		o := e.Observe()
		r.Commands = i
		var c core.Command
		if learn, ok := ai.Learning(e); ok {
			if _, err := e.Apply(learn); err != nil {
				return r, err
			}
			continue
		}
		switch o.Phase {
		case "complete":
			r.Result = "complete"
			levels(o)
			return r, nil
		case "scenario":
			switch o.Dialogue.Kind {
			case "dialogue":
				c.Kind = "next"
			case "choice":
				options := make([]string, 0, len(o.Dialogue.Choices))
				for k := range o.Dialogue.Choices {
					options = append(options, k)
				}
				slices.Sort(options)
				c = core.Command{Kind: "choose", Option: options[0]}
			case "preparation":
				deploy := []string{}
				for _, id := range party {
					for _, of := range o.Characters {
						if of.ID == id {
							deploy = append(deploy, id)
						}
					}
				}
				if equipAny(e, o, tried) {
					continue
				}
				limit := d.Stages[o.Stage].Limit
				if len(party) == 0 && len(o.Deployment) > limit {
					deploy = trim(d, o.Deployment, limit)
					party = deploy
				}
				if len(party) > 0 && !reflect.DeepEqual(deploy, o.Deployment) {
					c = core.Command{Kind: "deploy", Deployment: deploy}
				} else {
					c.Kind = "start"
					entry = map[string]int{}
					for _, v := range o.Characters {
						if slices.Contains(o.Deployment, v.ID) {
							entry[v.ID] = v.Level
						}
					}
					stageStats = StageReport{Healing: map[string]int{}, SupportCasts: map[string]int{}, Stage: o.Stage + 1, Entry: entry, Exit: map[string]int{}, Experience: map[string]int{}, Damage: map[string]int{}, EffectiveHits: map[string]int{}, SpellHits: map[string]int{}}
				}
			}
		case "result":
			r.Rounds = append(r.Rounds, o.Round)
			stageStats.Rounds = o.Round
			for _, v := range o.Characters {
				if _, ok := entry[v.ID]; ok {
					stageStats.Exit[v.ID] = v.Level
				}
			}
			r.Stages = append(r.Stages, stageStats)
			if o.Result == "defeat" {
				r.Result = "defeat"
				levels(o)
				return r, nil
			}
			c.Kind = "continue"
		case "battle":
			var err error
			if c, err = ai.NextWithDuels(e, duels); err != nil {
				return r, err
			}
			step, err := e.Apply(c)
			if err != nil {
				return r, err
			}
			record(step.Events)
			continue
		}
		applied, err := e.Apply(c)
		if err != nil {
			return r, err
		}
		record(applied.Events)
	}
	return r, fmt.Errorf("simulation command limit")
}

// equipAny puts each warehouse item once on the first deployed character who can carry it; it reports whether one was equipped.
func equipAny(e *core.Engine, o core.Observation, tried map[string]bool) bool {
	for _, item := range slices.Sorted(maps.Keys(o.Warehouse)) {
		if o.Warehouse[item] < 1 || tried[item] {
			continue
		}
		tried[item] = true
		for _, id := range o.Deployment {
			if _, err := e.Apply(core.Command{Kind: "equip", Actor: id, Item: item}); err == nil {
				return true
			}
		}
	}
	return false
}

// trim keeps the lords and then the earliest of the deployment, up to limit.
func trim(d *content.Data, deployment []string, limit int) []string {
	out := []string{}
	for _, id := range deployment {
		if d.Characters[id].Lord {
			out = append(out, id)
		}
	}
	for _, id := range deployment {
		if len(out) < limit && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}
