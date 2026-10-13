package duel

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// Script describes presentation only; Outcome must match the resolved battle event.
type Script struct {
	Outcome  string
	Supports []string
	Beats    []Beat
}

type Scripts map[string]Script

func LoadScripts(path string) (Scripts, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var scripts Scripts
	if err := json.Unmarshal(b, &scripts); err != nil {
		return nil, fmt.Errorf("duel scripts: %w", err)
	}
	for id, script := range scripts {
		if err := script.Validate(); err != nil {
			return nil, fmt.Errorf("duel %s: %w", id, err)
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

func (s Scripts) Story(id, outcome string) Story {
	if script, ok := s[id]; ok && script.Outcome == outcome {
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
