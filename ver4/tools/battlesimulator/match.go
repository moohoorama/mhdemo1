package main

import (
	"fmt"
	"io"
	"sort"
	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/skirmish"
)

// batch plays n matches from seed on and prints the win rates and per-officer averages.
func batch(w io.Writer, d *content.Data, c skirmish.Config, n int) error {
	wins := map[string]int{}
	rounds := 0
	type tally struct {
		view            core.UnitView
		dealt, hp, lost int
	}
	units := map[string]*tally{}
	for i := range n {
		out, err := skirmish.Play(d, c.Seed+uint64(i), c.MaxRounds)
		if err != nil {
			return fmt.Errorf("seed %d: %w", c.Seed+uint64(i), err)
		}
		wins[out.Winner]++
		rounds += out.Rounds
		for _, u := range out.Units {
			t := units[u.ID]
			if t == nil {
				t = &tally{view: u}
				units[u.ID] = t
			}
			t.dealt += out.Dealt[u.ID]
			t.hp += u.HP * 100 / max(1, u.Stats.MaxHP)
			if u.HP <= 0 {
				t.lost++
			}
		}
	}
	pct := func(k int) float64 { return float64(k) * 100 / float64(n) }
	fmt.Fprintf(w, "%s · %d판 (seed %d~%d) · 최대 %d턴\n", c.Terrain, n, c.Seed, c.Seed+uint64(n)-1, c.MaxRounds)
	fmt.Fprintf(w, "아군 승 %.0f%% · 적군 승 %.0f%% · 무승부 %.0f%% · 평균 %.1f턴\n\n", pct(wins["ally"]), pct(wins["enemy"]), pct(wins["draw"]), float64(rounds)/float64(n))
	ids := make([]string, 0, len(units))
	for id := range units {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	fmt.Fprintf(w, "%-4s %-6s %4s %-6s %6s %6s %6s %6s %7s %7s\n", "진영", "장수", "Lv", "병종", "병력", "공격", "방어", "순발", "평균피해", "남은병력")
	for _, id := range ids {
		t := units[id]
		u := t.view
		side := map[string]string{"ally": "아군", "enemy": "적군"}[u.Faction]
		fmt.Fprintf(w, "%-4s %-6s %4d %-6s %6d %6d %6d %6d %8d %6d%% %s\n", side, u.Name, u.Level, u.Class, u.Stats.MaxHP, u.Stats.Attack, u.Stats.Defense, u.Stats.Agility,
			t.dealt/n, t.hp/n, fmt.Sprintf("퇴각 %.0f%%", pct(t.lost)))
	}
	return nil
}
