package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"srpg/internal/content"
)

func TestOfficerPaletteOverridesCharacterColors(t *testing.T) {
	g := battleController(t)
	f := g.S.Data.Factions["wei"]
	f.OfficerPalette = map[string]content.Colors{"themeA": {"#2a2a30", "#18181c", "#0c0c10"}}
	g.S.Data.Factions["wei"] = f
	look := content.Look{Palette: map[string]content.Colors{"themeA": {"#ffffff"}}}
	assert.Equal(t, colorsOf(f.OfficerPalette["themeA"]), g.palette("wei*", look)["themeA"])
	assert.Equal(t, colorsOf(look.Palette["themeA"]), g.palette("wei", look)["themeA"])
}
