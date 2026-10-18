package duel

import "srpg/internal/content"

// loadScripts is the staging of every duel in the campaign data.
func loadScripts() (Scripts, error) {
	d, err := content.Load("../../assets/data")
	if err != nil {
		return nil, err
	}
	return FromData(d)
}
