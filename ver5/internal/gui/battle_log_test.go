package gui

import (
	"testing"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func logTestGame(t *testing.T) *Game {
	t.Helper()
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	return &Game{S: &session.Session{Data: d}}
}

func TestBattleLogCommand(t *testing.T) {
	cases := []struct {
		name   string
		cmd    core.Command
		events []core.Event
		want   []string
	}{
		{"attack and experience", core.Command{Kind: "attack", Actor: "liubei", Target: "guanyu"}, []core.Event{
			{Kind: "damage", Actor: "liubei", Target: "guanyu", Amount: 320},
			{Kind: "experience", Actor: "liubei", Amount: 12},
		}, []string{"유비 → 관우: 일반 공격", "유비 → 관우: 피해 320", "유비: 경험치 +12"}},
		{"supply at the cap", core.Command{Kind: "skill", Actor: "liubei", Target: "guanyu", Skill: "petition"}, []core.Event{
			{Kind: "cost", Actor: "liubei", Amount: 24},
			{Kind: "effect", Actor: "liubei", Target: "guanyu", Amount: 3},
		}, []string{"유비 → 관우: 헌책 사용", "유비: 병법치 -24", "유비 → 관우: 병법치 +3"}},
		{"counter stance", core.Command{Kind: "skill", Actor: "liubei", Target: "liubei", Skill: "counter"}, []core.Event{
			{Kind: "effect", Actor: "liubei", Target: "liubei"},
		}, []string{"유비 → 유비: 반격 사용", "유비 → 유비: 반격 태세 적용"}},
		{"MP item", core.Command{Kind: "item", Actor: "liubei", Target: "liubei", Item: "small_scroll"}, []core.Event{
			{Kind: "item", Actor: "liubei", Target: "liubei", Amount: 2},
		}, []string{"유비 → 유비: 소병법단 사용", "유비 → 유비: 병법치 +2"}},
		{"wait without events", core.Command{Kind: "wait", Actor: "liubei"}, nil, []string{"유비: 대기"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := logTestGame(t)
			g.logCommand(tc.cmd, tc.events)
			check(t, g.history, tc.want)
		})
	}
}

func TestLogWindow(t *testing.T) {
	for _, tc := range []struct {
		name         string
		rows, scroll int
		want         []string
		wantScroll   int
	}{
		{"latest", 2, 0, []string{"c", "d"}, 0},
		{"older", 2, 1, []string{"b", "c"}, 1},
		{"scroll limit", 2, 100, []string{"a", "b"}, 2},
		{"expanded", 8, 2, []string{"a", "b", "c", "d"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines, scroll := logWindow([]string{"a", "b", "c", "d"}, tc.rows, tc.scroll)
			check(t, lines, tc.want)
			check(t, scroll, tc.wantScroll)
		})
	}
}

func TestBattleLogRetainsRecentRecords(t *testing.T) {
	g := logTestGame(t)
	for i := 0; i < 310; i++ {
		g.logCommand(core.Command{}, []core.Event{{Kind: "experience", Actor: "liubei", Amount: i}})
	}
	check(t, len(g.history), 300)
	check(t, g.history[0], "유비: 경험치 +10")
	check(t, g.history[299], "유비: 경험치 +309")
}

func TestBattleLogHeight(t *testing.T) {
	for _, tc := range []struct {
		name         string
		height, rows int
	}{
		{"default", 0, 7},
		{"smallest", 80, 1},
		{"largest", 440, 18},
		{"clamp small", 20, 1},
		{"clamp large", 800, 18},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := &Game{logHeight: tc.height}
			r := g.battleLogRect()
			check(t, (r.Dy()-40)/logLineHeight, tc.rows)
			check(t, r.Min.X, 12)
			check(t, r.Max.Y, Height-12)
		})
	}
}
