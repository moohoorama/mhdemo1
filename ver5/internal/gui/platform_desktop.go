//go:build !mobile

package gui

// touchUI is true in the build for phones (-tags mobile): no hover, fingers for pointers.
const touchUI = false

// The sortie card on a desktop: a 70×84 UI px card (about 160% of the old 44×52), three a row beside the detail panel
// (sortie_unified.go), the deputy's medal 48 px overhanging its corner.
const (
	sortieLeadW = 70
	sortieLeadH = 84
	sortieMedal = 48
	sortieSwap  = 24
	sortieCols  = 3
)
