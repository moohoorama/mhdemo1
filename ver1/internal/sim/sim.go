package sim

import (
	"fmt"
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
func RunConfigured(d *content.Data, seed uint64, party []string, duels bool) (Report, error) {
	e := core.New(d, seed)
	r := Report{Core: core.CoreVersion, Rules: core.RulesVersion, Content: d.Version, ContentHash: d.Hash, AI: ai.Version, Seed: seed, Levels: map[string]int{}}

	entry := map[string]int{}
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
	for i := 0; i < 20000; i++ {
		o := e.Observe()
		r.Commands = i
		var c core.Command
		switch o.Phase {
		case "complete":
			r.Result = "complete"
			for _, v := range o.Officers {
				if v.ID == "유비" || v.ID == "관우" || v.ID == "장비" || v.ID == "간옹" {
					r.Levels[v.ID] = v.Level
				}
			}
			return r, nil
		case "scenario":
			switch o.Dialogue.Kind {
			case "dialogue":
				c.Kind = "next"
			case "choice":
				c = core.Command{Kind: "choose", Option: "결의"}
			case "preparation":
				deploy := []string{}
				for _, id := range party {
					for _, of := range o.Officers {
						if of.ID == id {
							deploy = append(deploy, id)
						}
					}
				}
				if len(party) > 0 && !reflect.DeepEqual(deploy, o.Deployment) {
					c = core.Command{Kind: "deploy", Deployment: deploy}
				} else if o.Warehouse["item_024"] > 0 {
					c = core.Command{Kind: "equip", Actor: "유비", Item: "item_024"}
				} else {
					c.Kind = "start"
					entry = map[string]int{}
					for _, v := range o.Officers {
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
			for _, v := range o.Officers {
				if _, ok := entry[v.ID]; ok {
					stageStats.Exit[v.ID] = v.Level
				}
			}
			r.Stages = append(r.Stages, stageStats)
			if o.Result == "defeat" {
				r.Result = "defeat"
				for _, v := range o.Officers {
					if v.ID == "유비" || v.ID == "관우" || v.ID == "장비" || v.ID == "간옹" {
						r.Levels[v.ID] = v.Level
					}
				}
				return r, nil
			}
			c.Kind = "continue"
		case "battle":
			c = core.Command{Kind: "end"}
			for _, u := range o.UnitViews {
				if u.HP > 0 && u.Faction == o.Turn && !u.Done {
					var err error
					c, err = ai.ChooseWithDuels(e, u.ID, duels)
					if err != nil {
						return r, err
					}
					break
				}
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
