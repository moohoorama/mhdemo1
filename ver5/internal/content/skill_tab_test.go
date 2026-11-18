package content

import "testing"

func TestSkillTab(t *testing.T) {
	d, err := Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("martial skills in data", func(t *testing.T) {
		for _, id := range []string{"aim", "reload", "counter"} {
			if d.Skills[id].Tab != TabMartial {
				t.Errorf("%s is not in the martial tab", id)
			}
		}
	})
	t.Run("unknown tab is rejected", func(t *testing.T) {
		s := d.Skills["aim"]
		s.Tab = "nowhere"
		d.Skills["aim"] = s
		if err := d.Validate(); err == nil {
			t.Error("a skill with an unknown tab passed validation")
		}
	})
}
