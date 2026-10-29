package gui

import (
	"image"
	"testing"

	"srpg/internal/content"
)

func TestLooksAreTheCellsWithAStateThatShows(t *testing.T) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	st := content.Stage{Width: 5, Height: 1, Tiles: []string{"g-eg-Fg-sg--wbd"}}
	looks := Looks(d, &st)
	want := map[image.Point]string{{X: 0, Y: 0}: "ember", {X: 1, Y: 0}: "flame", {X: 2, Y: 0}: "soot", {X: 4, Y: 0}: "damaged"}
	if len(looks) != len(want) {
		t.Fatalf("looks %v", looks)
	}
	for p, l := range want {
		if looks[p] != l {
			t.Errorf("%v: %q, want %q", p, looks[p], l)
		}
	}
	if Looks(d, nil) != nil {
		t.Error("a missing map has no looks")
	}
}

func TestMapFXDemoPaintsAValidMapAndTheArtIsLoaded(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	if err != nil {
		t.Fatal(err)
	}
	a := d.g.assets
	for _, name := range []string{"wall_clean_00", "wall_clean_15", "gatehouse_closed_u_0", "gatehouse_closed_v_3", "fence_00", "fence_15", "ash_2_0", "burnt_village_0", "burnt_village_3", "burnt_camp", "burnt_forest_3", "bridge_u_damaged", "bridge_v_broken"} {
		if _, ok := a.Props[name]; !ok {
			t.Errorf("no picture %q", name)
		}
	}
	for _, name := range []string{"flame_t", "flame_t_small", "smoke_t"} {
		if s := a.Anims[name]; len(s.Frames) == 0 {
			t.Errorf("no flame strip %q", name)
		}
	}
	looks := LooksOf(d.g.S.Data, d.g.S.Engine.Observe())
	seen := map[string]bool{}
	for _, l := range looks {
		seen[l] = true
	}
	for _, l := range []string{"ember", "flame", "soot", "damaged", "broken"} {
		if !seen[l] {
			t.Errorf("the demo shows no %s cell", l)
		}
	}
}
