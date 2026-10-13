package content

import (
	"slices"
	"testing"
)

// classPools is trait-design-v2 section 8 as the data bundle teaches it.
var classPools = map[string][]string{
	"light_infantry":     {"guard_prep", "fire", "first_aid"},
	"heavy_infantry":     {"deputy_appoint", "brace_prep", "assist"},
	"guards":             {"castle_defense", "ironwall", "siegecraft"},
	"light_cavalry":      {"dash", "disrupt_prep", "provoke_prep"},
	"heavy_cavalry":      {"deputy_appoint", "thrust_art", "breakthrough"},
	"royal_guard":        {"gale", "charge", "chain_thrust_art"},
	"archer":             {"relief", "snipe_prep", "aim_prep"},
	"crossbow":           {"deputy_appoint", "water", "fire_arrow_art", "reload_prep"},
	"repeating_crossbow": {"exhort", "rapid_art", "penetrate_art"},
}

func TestClassPoolsFollowTraitDesign(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	for class, want := range classPools {
		got := slices.Clone(d.Classes[class].Pool)
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s teaches %v, want %v", class, got, want)
		}
	}
	for _, class := range []string{"staff_officer", "sorcerer"} {
		if !slices.Contains(d.Classes[class].Pool, "deputy_appoint") {
			t.Errorf("%s cannot learn deputy_appoint", class)
		}
	}
}

func TestEveryLearnableSkillTraitOpensASkill(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	opens := map[string]bool{}
	for _, s := range d.Skills {
		for _, trait := range s.Requires {
			opens[trait] = true
		}
	}
	for _, class := range classPools {
		for _, trait := range class {
			if d.Traits[trait].Cost < 1 {
				t.Errorf("%s is not learnable", trait)
			}
		}
	}
	for _, trait := range []string{"guard_prep", "brace_prep", "first_aid", "disrupt_prep", "provoke_prep", "thrust_art", "chain_thrust_art",
		"snipe_prep", "aim_prep", "fire_arrow_art", "reload_prep", "rapid_art", "penetrate_art"} {
		if !opens[trait] {
			t.Errorf("trait %s opens no skill", trait)
		}
	}
}
