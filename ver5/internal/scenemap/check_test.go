package scenemap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	m, c := load(t, "S_castle_town")
	assert.Empty(t, Check(m, c), "the map as it is")

	t.Run("an object in a gate's way", func(t *testing.T) {
		m, c := load(t, "S_castle_town")
		m.Objects = append(m.Objects, Object{Name: "stree_bush_0", U: -1, V: 8})
		assert.Equal(t, []Problem{{-1, 8, "stree_bush_0가 성벽 줄 위에 섬"}}, Check(m, c))
	})
	t.Run("the same object twice at one point", func(t *testing.T) {
		m, c := load(t, "S_castle_town")
		m.Objects = append(m.Objects, m.Objects[len(m.Objects)-1])
		assert.Len(t, Check(m, c), 1)
	})
	t.Run("wide objects overlapping, small ones close together left alone", func(t *testing.T) {
		m, c := load(t, "S_castle_town")
		m.Objects = append(m.Objects, Object{Name: "house_tile", U: 20, V: 20}, Object{Name: "house_thatch", U: 21, V: 21},
			Object{Name: "wine_jar", U: 24, V: 24}, Object{Name: "wine_jar", U: 24, V: 24, DX: 6})
		assert.Equal(t, []Problem{{20, 20, "house_tile와 house_thatch가 겹침"}}, Check(m, c))
	})
	t.Run("a wide object close to the wall in front of it", func(t *testing.T) {
		m, c := load(t, "S_xiangyang")
		assert.Empty(t, Check(m, c), "the map as it is")
		m.Objects = append(m.Objects, Object{Name: "house_tile_x", U: 44, V: 39}, Object{Name: "house_tile", U: 44, V: 37},
			Object{Name: "house_tile", U: 25, V: 36}, Object{Name: "stree_plum_0", U: 30, V: 39})
		assert.Equal(t, []Problem{
			{44, 39, "house_tile_x가 앞쪽 성벽과 2칸 이내"},
			{44, 37, "house_tile가 앞쪽 성벽과 2칸 이내"},
		}, Check(m, c))
	})
}
