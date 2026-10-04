package core

import (
	"srpg/internal/content"
	"testing"
)

func TestOfficerStatsMatchesBattleStats(t *testing.T) {
	e := battle(t)
	s, err := e.OfficerStats("유비")
	if err != nil {
		t.Fatal(err)
	}
	if s != e.stats(e.unit("유비")) {
		t.Fatalf("officer %+v, unit %+v", s, e.stats(e.unit("유비")))
	}
	if _, err := e.OfficerStats("nobody"); err == nil {
		t.Fatal("unknown officer accepted")
	}
}

func TestCounterDamage(t *testing.T) {
	t.Run("no stance", func(t *testing.T) {
		e := battle(t)
		enemy := firstEnemy(e)
		if n := e.CounterDamage(Command{Kind: "attack", Actor: "유비", Target: enemy.ID}); n != 0 {
			t.Fatalf("counter without stance: %d", n)
		}
	})
	t.Run("stance in reach", func(t *testing.T) {
		e := battle(t)
		enemy := firstEnemy(e)
		u := e.unit("유비")
		enemy.X, enemy.Y = u.X+1, u.Y
		enemy.Status["counter"] = Status{Turns: 2}
		if n := e.CounterDamage(Command{Kind: "attack", Actor: "유비", Target: enemy.ID}); n <= 0 {
			t.Fatal("counter stance ignored")
		}
		if n := e.CounterDamage(Command{Kind: "skill", Skill: "초열", Actor: "유비", Target: enemy.ID}); n != 0 {
			t.Fatalf("magic provoked a counter: %d", n)
		}
	})
}

func TestThreatCoversReachAndRange(t *testing.T) {
	e := battle(t)
	enemy := firstEnemy(e)
	enemy.Moved, enemy.Done = true, true
	cells := map[content.Point]bool{}
	for _, p := range e.Threat(enemy.ID) {
		cells[p] = true
	}
	if !cells[content.Point{X: enemy.X + 1, Y: enemy.Y}] && !cells[content.Point{X: enemy.X - 1, Y: enemy.Y}] {
		t.Fatal("adjacent cell not threatened")
	}
	if len(cells) <= 4 {
		t.Fatalf("spent unit's reach ignored: %d cells", len(cells))
	}
	if e.Threat("nobody") != nil {
		t.Fatal("unknown unit has threat")
	}
}

func firstEnemy(e *Engine) *Unit {
	for _, id := range keys(e.unitIDs) {
		if u := e.unit(id); u.Faction == "enemy" && u.HP > 0 {
			return u
		}
	}
	return nil
}
