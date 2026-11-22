package content

import (
	"slices"
	"testing"
)

// spellRoots are the spell roots classes teach.
var spellRoots = []string{"fire", "water", "earth", "relief", "exhort", "stratagem"}

func TestSpellRootsAreLearnableAndTaught(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range spellRoots {
		if d.Traits[root].Cost < 1 {
			t.Errorf("%s is not learnable", root)
		}
	}
	for _, root := range spellRoots {
		taught := false
		for _, c := range d.Classes {
			taught = taught || slices.Contains(c.Pool, root)
		}
		if !taught {
			t.Errorf("no class teaches %s", root)
		}
	}
}

// Officers start with no trait learned, so every scholar class must teach a spell root on its line.
func TestScholarClassesTeachASpellRoot(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range []string{"scholar", "staff_officer", "strategist", "orator", "diplomat", "sorcerer", "mystic"} {
		has := false
		for _, trait := range d.Classes[class].AllPool() {
			has = has || slices.Contains(spellRoots, trait)
		}
		if !has {
			t.Errorf("%s teaches no spell root on its line", class)
		}
	}
}
