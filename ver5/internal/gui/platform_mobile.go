//go:build mobile

package gui

// touchUI is true in the build for phones (-tags mobile): no hover, fingers for pointers.
const touchUI = true

// The sortie card on a phone: a card of 70×84 UI px, three a row beside the detail panel (sortie_unified.go), the medal
// 48 px and its swap mark 24 px, whose touch target is touchH square.
const (
	sortieLeadW = 70
	sortieLeadH = 84
	sortieMedal = 48
	sortieSwap  = 24
	sortieCols  = 3
)
