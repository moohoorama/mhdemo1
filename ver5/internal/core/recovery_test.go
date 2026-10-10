package core

import "testing"

func TestNextTurnRecoveryMatchesTurnStart(t *testing.T) {
	for _, tc := range []struct {
		name    string
		turn    string
		terrain bool
		missing int
		traits  []string
		poison  int
		aura    bool
	}{
		{name: "open ground", turn: "ally", missing: 500},
		{name: "healing terrain", turn: "ally", terrain: true, missing: 500},
		{name: "traits and aura", turn: "ally", terrain: true, missing: 500, traits: []string{"focus", "regen_t", "heal_t"}, aura: true},
		{name: "recovery at cap", turn: "ally", terrain: true, missing: 1},
		{name: "full", turn: "ally", terrain: true, missing: 0},
		{name: "opposing turn", turn: "enemy", terrain: true, missing: 500},
		{name: "poison expires", turn: "ally", terrain: true, missing: 1, poison: 1},
		{name: "poison remains", turn: "enemy", terrain: true, missing: 1, poison: 1},
		{name: "cleanse", turn: "enemy", terrain: true, missing: 1, poison: 1, traits: []string{"calm"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, hero, _ := arena(t)
			e.state.Turn = tc.turn
			with(e, hero.ID, tc.traits...)
			if tc.terrain {
				tileID := e.tile(hero.X, hero.Y)
				tile := e.data.Tiles[tileID]
				tile.RestHP, tile.RestMP = 8, 2
				e.data.Tiles[tileID] = tile
			}
			if tc.aura {
				ally := e.unit("archer1")
				ally.X, ally.Y = 5, 5
				with(e, ally.ID, "aura_t")
			}
			st := e.stats(hero)
			hero.HP, hero.MP = st.MaxHP-tc.missing, st.MaxMP-min(tc.missing, 50)
			if tc.poison > 0 {
				hero.Status["poison"] = Status{Turns: tc.poison}
			}
			before := encoded(e.Snapshot())
			predicted := e.NextTurnRecovery(hero.ID)
			if before != encoded(e.Snapshot()) {
				t.Fatal("recovery prediction changed the engine")
			}
			if predicted.TerrainHP != tc.terrain {
				t.Fatalf("terrain prediction %+v", predicted)
			}
			if tc.turn == hero.Faction {
				apply(t, e, Command{Kind: "end"})
			}
			r := apply(t, e, Command{Kind: "end"})
			hp, mp := 0, 0
			for _, event := range r.Events {
				if event.Actor != hero.ID {
					continue
				}
				switch event.Kind {
				case "recover-hp":
					hp += event.Amount
				case "recover-mp":
					mp += event.Amount
				}
			}
			if predicted.HP != hp || predicted.MP != mp {
				t.Fatalf("prediction %+v, actual HP +%d MP +%d", predicted, hp, mp)
			}
			if tc.missing == 0 && (predicted.HP != 0 || predicted.MP != 0) {
				t.Fatal("full unit predicts recovery")
			}
			if tc.name == "open ground" && (predicted.HP != 0 || predicted.MP != st.Recovery) {
				t.Fatalf("open ground prediction %+v, base MP recovery %d", predicted, st.Recovery)
			}
		})
	}
}

func TestNextTurnRecoveryForMissingOrDeadUnit(t *testing.T) {
	e, hero, _ := arena(t)
	hero.HP = 0
	for _, id := range []string{"unknown", hero.ID} {
		t.Run(id, func(t *testing.T) {
			if got := e.NextTurnRecovery(id); got != (RecoveryPreview{}) {
				t.Fatalf("unexpected recovery %+v", got)
			}
		})
	}
}
