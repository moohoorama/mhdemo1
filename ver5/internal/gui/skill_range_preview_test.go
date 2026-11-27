package gui

import (
	"image/color"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

func TestSkillRangePreview(t *testing.T) {
	g := battleController(t)
	g.field = &Field{Map: &MapArt{W: 20, H: 20, Tiles: strings.Split(strings.Repeat(strings.Repeat(".", 20)+"\n", 20), "\n")[:20]}}
	hero := g.unitView(g.S.View(), "hero")
	require.NotNil(t, hero)
	g.ord = order{actor: "hero", stage: "sub", sub: "skill"}
	marked := func() map[content.Point]bool {
		out := map[content.Point]bool{}
		for _, h := range g.orderMarks(g.S.View()) {
			if h.Color == (color.RGBA{230, 90, 60, 60}) {
				out[content.Point{X: h.U, Y: h.V}] = true
			}
		}
		return out
	}
	t.Run("nothing is painted until a row is hovered", func(t *testing.T) {
		assert.Empty(t, marked())
	})
	t.Run("a hovered skill row paints the cells it reaches", func(t *testing.T) {
		g.ord.preview = "volley"
		want := g.S.Engine.ActionCells(core.Command{Kind: "skill", Skill: "volley", Actor: "hero"})
		require.NotEmpty(t, want)
		got := marked()
		assert.Len(t, got, len(want))
		for _, p := range want {
			assert.True(t, got[p], "%v", p)
		}
	})
	t.Run("a self skill paints only the actor's own cell", func(t *testing.T) {
		for id, s := range g.S.Data.Skills {
			if s.Mode == content.ModeSelf {
				g.ord.preview = id
				assert.Equal(t, map[content.Point]bool{{X: hero.X, Y: hero.Y}: true}, marked())
				return
			}
		}
		t.Skip("no self skill")
	})
	t.Run("the preview is for the list only", func(t *testing.T) {
		g.ord.stage = "menu"
		assert.Empty(t, marked())
	})
}
