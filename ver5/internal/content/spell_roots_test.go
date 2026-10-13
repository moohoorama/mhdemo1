package content

import "testing"

var spellRoots = []string{"fire", "water", "earth", "relief", "exhort", "hinder", "stratagem"}

func TestSpellRootsAreLearnableAndTaught(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range spellRoots {
		if d.Traits[root].Cost < 1 {
			t.Errorf("%s is not learnable", root)
		}
		taught := false
		for _, c := range d.Classes {
			for _, p := range c.Pool {
				taught = taught || p == root
			}
		}
		if !taught {
			t.Errorf("no class teaches %s", root)
		}
	}
}

func TestScholarsStartWithASpellRoot(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for id, c := range d.Characters {
		if !isScholarClass(c.Class) {
			continue
		}
		has := false
		for _, trait := range c.Traits {
			for _, root := range spellRoots {
				has = has || trait == root
			}
		}
		if !has {
			t.Errorf("%s is a %s without any spell root", id, c.Class)
		}
	}
}

func isScholarClass(class string) bool {
	switch class {
	case "scholar", "staff_officer", "strategist", "sorcerer", "mystic":
		return true
	}
	return false
}
