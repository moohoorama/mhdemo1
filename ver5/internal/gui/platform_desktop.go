//go:build !mobile

package gui

// touchUI is true in the build for phones (-tags mobile): no hover, fingers for pointers.
const touchUI = false

// The sortie card on a desktop: the officer's card and the deputy's medal that overhangs its corner.
const (
	sortieLeadW = 104
	sortieLeadH = 124
	sortieMedal = 56
	sortieSwap  = 20
)
