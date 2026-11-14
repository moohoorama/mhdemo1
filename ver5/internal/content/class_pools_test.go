package content

import (
	"slices"
	"testing"
)

// classPools is the classified trait draft (_newfeature.md section 2) as the data bundle teaches it.
var classPools = map[string][]string{
	"light_infantry":     {"diagonal", "fire", "brace_prep"},
	"heavy_infantry":     {"deputy_appoint", "iron_armor", "ranged_ward"},
	"guards":             {"castle_defense", "siegecraft", "composure"},
	"pikeman":            {"deputy_appoint", "long_spear", "pierce", "assist"},
	"light_cavalry":      {"dash", "accuracy_up", "grass_footing"},
	"heavy_cavalry":      {"deputy_appoint", "counter_ward", "magic_ward"},
	"royal_guard":        {"diagonal", "charge_strike", "impact", "vital_strike"},
	"horse_archer":       {"deputy_appoint", "snipe_prep", "aim_prep"},
	"horse_crossbow":     {"reload_prep", "fire_arrow_art", "rapid_art"},
	"archer":             {"relief", "accuracy_up", "aim_prep"},
	"crossbow":           {"deputy_appoint", "water", "fire_arrow_art", "reload_prep"},
	"repeating_crossbow": {"exhort", "rapid_art", "penetrate_art"},
	"catapult":           {"siegecraft", "cross_strike", "far_sight"},
	"bandit":             {"counter_art", "earth", "provoke_prep", "disrupt_prep", "plunder"},
	"turban":             {"counter_art", "earth", "provoke_prep", "disrupt_prep"},
	"scholar":            {"fire", "relief", "focus", "foresight"},
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
	for _, class := range []string{"staff_officer", "orator", "sorcerer"} {
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
	for _, trait := range []string{"guard_prep", "brace_prep", "first_aid", "disrupt_prep", "provoke_prep", "press_prep",
		"slander_prep", "focus", "spirit_prep", "ambition", "inspire_prep", "extend_prep", "stance_prep",
		"snipe_prep", "aim_prep", "fire_arrow_art", "reload_prep", "rapid_art", "penetrate_art"} {
		if !opens[trait] {
			t.Errorf("trait %s opens no skill", trait)
		}
	}
}
