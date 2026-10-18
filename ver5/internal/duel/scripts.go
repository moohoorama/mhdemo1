package duel

import (
	"fmt"
	"math"

	"srpg/internal/content"
)

// Script describes presentation only; Outcome must match the resolved battle event.
type Script struct {
	Outcome  string
	Supports []string
	Beats    []Beat
	Variants map[string][]Beat // by the officer who moved to meet the other
}

type Scripts map[string]Script

// checkResults keeps the staging from contradicting the fixed result: a fighter who is shown
// leaving the field must have the result retreat, and one who is shown beaten must not stay
// untouched while the other falls.
func checkResults(d content.Duel, beats []Beat) error {
	last := beats[len(beats)-1]
	if last.Kind != "escaped" && last.Kind != "withdrawn" {
		return nil
	}
	result := d.AllyResult
	if last.Side == 1 {
		result = d.EnemyResult
	}
	if result != "retreat" {
		return fmt.Errorf("the staging ends with side %d leaving the field but its result is %q", last.Side, result)
	}
	return nil
}

func beatsOf(list []content.Beat) []Beat {
	out := make([]Beat, len(list))
	for i, b := range list {
		out[i] = Beat(b)
	}
	return out
}

// FromData collects the staging of every duel the stages hold; duels without beats are staged by
// the default timeline.
func FromData(d *content.Data) (Scripts, error) {
	scripts := Scripts{}
	for _, st := range d.Stages {
		for _, du := range st.Duels {
			if len(du.Beats) == 0 {
				continue
			}
			script := Script{Outcome: du.Outcome, Supports: du.Supports, Beats: beatsOf(du.Beats)}
			if err := script.Validate(); err != nil {
				return nil, fmt.Errorf("duel %s: %w", du.ID, err)
			}
			if err := checkResults(du, script.Beats); err != nil {
				return nil, fmt.Errorf("duel %s: %w", du.ID, err)
			}
			for _, v := range du.Variants {
				alt := Script{Outcome: du.Outcome, Supports: du.Supports, Beats: beatsOf(v.Beats)}
				if _, dup := script.Variants[v.First]; dup || v.First == "" || alt.Validate() != nil {
					return nil, fmt.Errorf("duel %s: variant %q", du.ID, v.First)
				}
				if script.Variants == nil {
					script.Variants = map[string][]Beat{}
				}
				script.Variants[v.First] = alt.Beats
			}
			scripts[du.ID] = script
		}
	}
	return scripts, nil
}

func (s Script) Validate() error {
	if s.Outcome != "victory" && s.Outcome != "defeat" && s.Outcome != "draw" {
		return fmt.Errorf("invalid outcome %q", s.Outcome)
	}
	if len(s.Beats) == 0 {
		return fmt.Errorf("empty story")
	}
	if len(s.Supports) != 0 && len(s.Supports) != 2 {
		return fmt.Errorf("supports must contain two fighters")
	}
	for i, id := range s.Supports {
		if id == "" || i > 0 && id == s.Supports[0] {
			return fmt.Errorf("invalid support identity")
		}
	}
	joined := false
	for i, b := range s.Beats {
		kind := strikeKind(b.Kind)
		if kind != b.Kind && (kind != strikeKind(kind) || !isStrike(kind)) {
			return fmt.Errorf("invalid rush motion %q", b.Kind)
		}
		if b.Side < 0 || b.Side > 1+len(s.Supports) || b.Duration <= 0 || math.IsNaN(b.Duration) || math.IsInf(b.Duration, 0) {
			return fmt.Errorf("invalid beat %d", i)
		}
		if b.Side > 1 && ((kind != "talk" && kind != "engage" && kind != "charge-ready" && kind != "block" && kind != "critical-block" && kind != "dodge") || !joined) {
			return fmt.Errorf("unsupported support action or action before arrival")
		}
		if b.Kind == "reinforcements" {
			if len(s.Supports) != 2 || joined {
				return fmt.Errorf("invalid reinforcement arrival")
			}
			joined = true
		}
		switch kind {
		case "charge-ready", "engage", "press", "reinforcements", "retreat", "withdrawn", "yielding-block", "critical-dodge", "talk", "withdraw", "critical-block", "straight-chase", "advance", "approach", "close", "backstep", "crossing-charge", "slash-through", "turn", "hit", "block", "dodge", "critical", "backhit", "rest", "win", "draw", "escape", "catch", "escaped":
		default:
			return fmt.Errorf("unknown motion %q", b.Kind)
		}
		if i != len(s.Beats)-1 && (b.Kind == "win" || b.Kind == "draw" || b.Kind == "escaped" || b.Kind == "withdrawn") {
			return fmt.Errorf("terminal beat %d before ending", i)
		}
	}
	last := s.Beats[len(s.Beats)-1]
	valid := s.Outcome == "draw" && last.Kind == "draw" ||
		s.Outcome == "victory" && (last.Kind == "win" && last.Side == 0 || (last.Kind == "escaped" || last.Kind == "withdrawn") && last.Side == 1) ||
		s.Outcome == "defeat" && (last.Kind == "win" && last.Side == 1 || (last.Kind == "escaped" || last.Kind == "withdrawn") && last.Side == 0)
	if !valid {
		return fmt.Errorf("ending conflicts with outcome %s", s.Outcome)
	}
	return nil
}

func (s Scripts) Story(id, outcome string) Story { return s.StoryFor(id, outcome, "") }

// StoryFor is the staging of a duel that first was started by moving first (an officer id, or none).
func (s Scripts) StoryFor(id, outcome, first string) Story {
	if script, ok := s[id]; ok && script.Outcome == outcome {
		if beats, ok := script.Variants[first]; ok {
			return prepareStory(beats)
		}
		return prepareStory(script.Beats)
	}
	ending := 0
	switch outcome {
	case "defeat":
		ending = 1
	case "draw":
		ending = 2
	}
	return New(ending)
}
