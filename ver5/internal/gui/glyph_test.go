package gui

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const glyphChars = "0123456789+-%!:EXPMH 제턴치명막음상태회복학습이동력감거격경계고공교기독동란략러리물박반발방보복봉사소속수순신약어정중찔책팔피필해혼화"

func testGlyphAtlas() *glyphAtlas {
	a := &glyphAtlas{Height: 100, Space: 30, Glyphs: map[string]glyphEntry{}}
	for _, r := range glyphChars {
		if r != ' ' {
			a.Glyphs[string(r)] = glyphEntry{R: [4]int{0, 0, 80, 100}, Adv: 90}
		}
	}
	return a
}

func TestGlyphLayout(t *testing.T) {
	a := testGlyphAtlas()
	t.Run("advance scales with height", func(t *testing.T) {
		places, w, ok := a.layout("+12", 50)
		require.True(t, ok)
		assert.Len(t, places, 3)
		assert.InDelta(t, 135, w, 1e-9)
		assert.InDelta(t, 45, places[1].x, 1e-9)
	})
	t.Run("space moves the pen without a glyph", func(t *testing.T) {
		places, w, ok := a.layout("HP +9", 100)
		require.True(t, ok)
		assert.Len(t, places, 4)
		assert.InDelta(t, 90*4+30, w, 1e-9)
	})
	t.Run("missing character falls back", func(t *testing.T) {
		assert.False(t, a.has("가"))
		assert.Nil(t, a.text("가", 40, color.NRGBA{}, color.NRGBA{}))
	})
}

func TestGlyphAtlasCoversScreenText(t *testing.T) {
	a, err := loadGlyphAtlas("../../assets/graphics")
	require.NoError(t, err)
	for _, s := range []string{"0123456789", "+", "제1턴", "치명", "막음!", "EXP +12", "MP -3", "HP +40", "상태 회복", "학습:", "이동력 -1", "사거리 +1"} {
		assert.True(t, a.has(s), s)
	}
	for id, label := range statusLabels {
		assert.True(t, a.has(label), id)
	}
	for _, s := range a.Glyphs {
		assert.Positive(t, s.R[2]*s.R[3])
		assert.Positive(t, s.Adv)
	}
}
