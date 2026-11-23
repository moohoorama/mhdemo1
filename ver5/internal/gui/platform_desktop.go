//go:build !mobile

package gui

// touchUI is true in the build for phones (-tags mobile): no hover, fingers for pointers.
const touchUI = false

// The sortie card on a desktop: a 44×52 UI px card, five a row beside the detail panel (sortie_unified.go), the deputy's
// medal 26 px overhanging its corner.
const (
	sortieLeadW = 44
	sortieLeadH = 52
	sortieMedal = 26
	sortieSwap  = 14
	sortieCols  = 5
)
