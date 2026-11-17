package autoplay

import (
	"sort"

	"srpg/internal/core"
)

func choices(e *core.Engine) []string {
	var out []string
	for option := range e.Observe().Dialogue.Choices {
		out = append(out, option)
	}
	sort.Strings(out)
	return out
}

// ChooseStory avoids a premature ending by reading ahead through dialogue on copies of the engine.
func ChooseStory(e *core.Engine, p Player) core.Command {
	options := choices(e)
	best := options[0]
	for _, option := range options {
		trial, err := core.Restore(e.Data(), e.Snapshot())
		if err != nil {
			continue
		}
		if _, err := trial.Apply(core.Command{Kind: "choose", Option: option}); err == nil && storyContinues(trial, p, 200) {
			best = option
			break
		}
	}
	return core.Command{Kind: "choose", Option: best}
}

func storyContinues(e *core.Engine, p Player, budget int) bool {
	for ; budget > 0; budget-- {
		o := e.Observe()
		switch {
		case o.Phase == "complete":
			for _, stage := range e.Data().Stages {
				if stage.Order > 0 && !o.Executed["won-"+stage.ID] {
					return false
				}
			}
			return true
		case len(o.Learning) > 0:
			if _, err := e.Apply(p.Learn(e)); err != nil {
				return false
			}
		case e.Walking():
			commands := e.ExploreNext()
			if len(commands) == 0 {
				return false
			}
			for _, c := range commands {
				if _, err := e.Apply(c); err != nil {
					return false
				}
			}
		case o.Phase == "scenario" && o.Dialogue.Kind == "choice":
			for _, option := range choices(e) {
				trial, err := core.Restore(e.Data(), e.Snapshot())
				if err != nil {
					continue
				}
				if _, err := trial.Apply(core.Command{Kind: "choose", Option: option}); err == nil && storyContinues(trial, p, budget-1) {
					return true
				}
			}
			return false
		case o.Phase == "scenario" && (o.Dialogue.Kind == "say" || o.Dialogue.Kind == "narrate"):
			if _, err := e.Apply(core.Command{Kind: "next"}); err != nil {
				return false
			}
		default:
			return true
		}
	}
	return false
}
