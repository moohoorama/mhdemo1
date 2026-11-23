//go:build mobile

package gui

// touchUI is true in the build for phones (-tags mobile): no hover, fingers for pointers.
const touchUI = true

// The sortie card on a phone: the deputy's medal a third of the card's area and its swap mark 2x the desktop's, for a fingertip.
const (
	sortieLeadW = 120
	sortieLeadH = 124
	sortieMedal = 84
	sortieSwap  = 40
	sortieCols  = 3
)
