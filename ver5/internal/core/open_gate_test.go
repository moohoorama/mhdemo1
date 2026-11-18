package core

import (
	"testing"

	"srpg/internal/content"
)

func TestOpenGateIsCastle(t *testing.T) {
	real, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	e, hero, _ := arena(t)
	e.data.Grounds, e.data.Structures, e.data.States, e.data.Terrain = real.Grounds, real.Structures, real.States, real.Terrain
	class := e.unitClass(hero).Terrain
	castle, open, closed := "io-", "m-O", "m--"
	setCell(e, hero.X, hero.Y, castle)
	wantProps, wantMove := e.CellProps(hero.X, hero.Y), e.data.Move(class, e.Cell(hero.X, hero.Y), false)
	setCell(e, hero.X, hero.Y, open)
	if got := e.CellProps(hero.X, hero.Y); got != wantProps || !got.Castle || got.RestHP != 20 || got.RestMP != 3 {
		t.Fatalf("open gate props %+v, castle %+v", got, wantProps)
	}
	if got := e.data.Move(class, e.Cell(hero.X, hero.Y), false); got != wantMove {
		t.Fatalf("open gate move %+v, castle %+v", got, wantMove)
	}
	castleCell, _ := content.ParseCell(castle)
	for group := range e.data.Terrain {
		if g, c := e.data.Move(group, e.Cell(hero.X, hero.Y), false), e.data.Move(group, castleCell, false); g != c {
			t.Fatalf("%s: open gate %+v, castle %+v", group, g, c)
		}
	}

	st := e.stats(hero)
	hero.HP, hero.MP = st.MaxHP-500, 0
	e.state.Turn = hero.Faction
	apply(t, e, Command{Kind: "end"})
	apply(t, e, Command{Kind: "end"})
	hero = e.unit("hero")
	if want := st.MaxHP - 500 + st.MaxHP*20/100; hero.HP != min(st.MaxHP, want) {
		t.Fatalf("HP %d after resting on an open gate, max %d", hero.HP, st.MaxHP)
	}

	setCell(e, hero.X, hero.Y, closed)
	hero = e.unit("hero")
	if c := e.data.Move(class, e.Cell(hero.X, hero.Y), false).Cost; c != 0 {
		t.Fatalf("closed gate cost %d", c)
	}
	if p := e.CellProps(hero.X, hero.Y); p.Castle || p.RestHP != 0 {
		t.Fatalf("closed gate props %+v", p)
	}
}
