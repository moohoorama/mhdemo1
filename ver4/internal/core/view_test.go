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

func TestDuelSetOffByEitherSide(t *testing.T) {
	setup := func(t *testing.T) *Engine {
		e := battle(t)
		e.data = copyOf(e.data)
		e.data.Hash = e.state.ContentHash
		e.data.Stages[0].Duels = []content.Duel{{ID: "fixture", Ally: "관우", Enemy: "등무", Outcome: "victory", AllyResult: "stay", EnemyResult: "retreat"}}
		u, v := e.unit("관우"), e.unit("등무")
		u.X, u.Y, v.X, v.Y = 4, 4, 5, 4
		return e
	}
	t.Run("enemy attacks", func(t *testing.T) {
		e := setup(t)
		e.state.Turn = "enemy"
		hp := e.unit("관우").HP
		r := apply(t, e, Command{Kind: "attack", Actor: "등무", Target: "관우"})
		if len(r.Events) == 0 || r.Events[0].Kind != "duel" || r.Events[0].Actor != "관우" {
			t.Fatalf("events %+v", r.Events)
		}
		if e.unit("등무").HP != 0 || e.unit("관우").HP != hp || e.officer("관우").Level != 2 {
			t.Fatal("duel outcome")
		}
	})
	t.Run("out of reach", func(t *testing.T) {
		e := setup(t)
		e.unit("관우").X = 3
		if e.DuelFor(Command{Kind: "attack", Actor: "관우", Target: "등무"}) {
			t.Fatal("duel at range 2")
		}
	})
	t.Run("no duel command", func(t *testing.T) {
		e := setup(t)
		assertRejected(t, e, Command{Kind: "duel", Actor: "관우", Target: "등무"})
		for _, c := range e.LegalActions("관우").Commands {
			if c.Kind == "duel" {
				t.Fatal("duel offered as a command")
			}
		}
	})
}
