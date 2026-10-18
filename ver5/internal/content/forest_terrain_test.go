package content

import (
	"slices"
	"testing"
)

// Forests can be entered: foot classes pay one step, mounted and siege classes pay two.
func TestForestStepCost(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for id, c := range d.Classes {
		want := 1
		if slices.Contains(c.Tags, "mounted") || slices.Contains(c.Tags, "siege") {
			want = 2
		}
		if got := d.Move(c.Terrain, Cell{Ground: 'g', Structure: 't', State: NoLayer}).Cost; got != want {
			t.Errorf("%s pays %d to enter a forest, want %d", id, got, want)
		}
	}
}
