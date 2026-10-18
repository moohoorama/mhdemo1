package duel

import (
	"encoding/json"
	"os"

	"srpg/internal/content"
)

// brotherStory is the three brothers' duel against Lu Bu as it plays when who moved first.
func brotherStory(s Scripts, who string) Story { return s.StoryFor("hulao-brothers", "victory", who) }

// loadScripts is the staging of every duel in the campaign data.
func loadScripts() (Scripts, error) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		return nil, err
	}
	return FromData(d)
}

// earlyScripts are two short duels (a bandit officer against Zhang Fei and against Guan Yu) kept as a
// fixture: they pin down how brief fights are staged, whatever the campaign holds.
func earlyScripts() (Scripts, error) {
	b, err := os.ReadFile("testdata/early_duels.json")
	if err != nil {
		return nil, err
	}
	var duels []content.Duel
	if err := json.Unmarshal(b, &duels); err != nil {
		return nil, err
	}
	return FromData(&content.Data{Stages: []content.Stage{{Duels: duels}}})
}
