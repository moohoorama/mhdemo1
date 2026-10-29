package gui

import (
	"testing"

	"srpg/internal/content"
)

func TestBriefingListsGoalsAndBonusObjectives(t *testing.T) {
	g := battleController(t)
	st := &g.S.Data.Stages[g.S.Engine.Observe().Stage]
	st.Victory = []content.Goal{{If: []string{"defeated:boss"}, Text: "보스 격파"}, {If: []string{"alldefeated:enemy"}, Text: "적군 전멸"}}
	st.Defeat = []content.Goal{{If: []string{"at:enemy:1"}, Text: "진영 점령"}}
	st.TurnLimit = 12
	st.Sub = []content.Objective{{ID: "a", Text: "전원 생존", If: []string{"alive:hero"}}, {ID: "b", Text: "보스 격파 전 일기토", If: []string{"duel:none"}}}
	b := g.objective(g.S.Engine.Observe())
	if b.win != "보스 격파 또는 적군 전멸" {
		t.Fatalf("win: %q", b.win)
	}
	if b.lose != "hero 퇴각 / 진영 점령 / 12턴 초과" && b.lose != g.lordName()+" 퇴각 / 진영 점령 / 12턴 초과" {
		t.Fatalf("lose: %q", b.lose)
	}
	if len(b.subs) != 2 || b.subs[0] != (briefSub{"전원 생존", "지키는 중", true}) || b.subs[1] != (briefSub{"보스 격파 전 일기토", "미달성", false}) {
		t.Fatalf("subs: %+v", b.subs)
	}
}
