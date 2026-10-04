package content

import (
	"testing"
)

func TestOpeningProfilesAndValidation(t *testing.T) {
	d, err := Load("../../assets/content/campaign.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		id    string
		level int
	}{{"정원지", 4}, {"등무", 2}, {"장각", 2}, {"화웅", 6}, {"호진", 4}, {"여포", 9}, {"이유", 7}, {"이각", 7}} {
		if d.Officers[c.id].Level != c.level {
			t.Fatal(c.id)
		}
	}
	if len(d.Stages[0].Enemies) != 7 || len(d.Stages[1].Enemies) != 8 || len(d.Stages[2].Enemies) != 9 {
		t.Fatal("enemy counts")
	}
	d.Stages[0].Tiles[0] = "bad"
	if d.Validate() == nil {
		t.Fatal("bad map accepted")
	}
}
