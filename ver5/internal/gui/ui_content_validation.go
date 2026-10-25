package gui

import (
	"fmt"
	"strings"

	"srpg/internal/core"
)

// validateUIContent uses the game font and wrapping rules before any story can play.
func (g *Game) validateUIContent() error {
	const narrowW, narrowH = 800, 531 // 1280×850 at the largest supported 160% UI scale
	checkTalk := func(where, s string, narration bool) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s: empty dialogue", where)
		}
		if narration {
			w := min(760, float64(narrowW)*.8)
			wrapped := wrapWidth(s, g.serif(28, 700), w-84)
			lines := strings.Count(wrapped, "\n") + 1
			if max(170, 58+float64(lines)*28*1.45) <= float64(narrowH)*.8 {
				return nil
			}
		} else if _, _, ok := g.dialogueFit(s, narrowW, narrowH); ok {
			return nil
		}
		return fmt.Errorf("%s: dialogue exceeds the available UI area", where)
	}
	for _, n := range g.S.Data.Nodes {
		if n.Kind == "say" || n.Kind == "narrate" {
			if err := checkTalk("node "+n.ID, n.Text, n.Kind == "narrate"); err != nil {
				return err
			}
		}
	}
	for name, explore := range g.S.Data.Explores {
		for areaID, area := range explore.Areas {
			for personID, person := range area.People {
				for i, line := range person.Talk {
					if err := checkTalk(fmt.Sprintf("explore %s/%s/%s line %d", name, areaID, personID, i+1), line[1], line[0] == ""); err != nil {
						return err
					}
				}
				if person.Sortie != nil {
					if _, _, ok := g.confirmLayout(person.Sortie.Ask, narrowW, narrowH); !ok {
						return fmt.Errorf("explore %s/%s/%s: sortie question exceeds 80%% of the UI", name, areaID, personID)
					}
				}
			}
		}
	}
	for i, st := range g.S.Data.Stages {
		wins, losses := []string{}, []string{g.lordName() + " 퇴각"}
		for _, goal := range st.Victory {
			wins = append(wins, goal.Text)
		}
		for _, goal := range st.Defeat {
			if goal.Text != "" {
				losses = append(losses, goal.Text)
			}
		}
		if st.TurnLimit > 0 {
			losses = append(losses, fmt.Sprintf("%d턴 초과", st.TurnLimit))
		}
		brief := briefing{win: strings.Join(wins, " 또는 "), lose: strings.Join(losses, " / ")}
		for _, sub := range st.Sub {
			brief.subs = append(brief.subs, briefSub{text: sub.Text, done: true})
		}
		rows := g.objectiveRows(core.Observation{State: core.State{Stage: i, Round: max(1, st.TurnLimit)}}, brief)
		if _, ok := g.objectiveLayout(rows, narrowW, narrowH); !ok {
			return fmt.Errorf("stage %s: objective exceeds 80%% of the UI", st.ID)
		}
		for _, event := range st.Events {
			for _, action := range event.Do {
				if action.Kind != "say" {
					continue
				}
				for j, line := range action.Lines {
					if err := checkTalk(fmt.Sprintf("stage %s event %s line %d", st.ID, event.ID, j+1), line.Text, false); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
