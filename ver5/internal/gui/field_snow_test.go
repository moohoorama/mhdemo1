package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The snow pictures stand in the places of the originals: same size and pivot, same frame count, so the plan and
// the shadows lay out as before.
func TestSnowAssetsMatchTheOriginals(t *testing.T) {
	d, err := NewDemo("../..", DemoOptions{Name: "mapfx"})
	require.NoError(t, err)
	a := d.g.assets
	require.NotNil(t, a.snowArt)
	s := a.snowAssets()
	require.NotSame(t, a, s)
	assert.Same(t, s, a.snowAssets())
	require.NotEmpty(t, a.snowArt.strips)
	for name, strip := range a.snowArt.strips {
		orig := a.Anims[name]
		require.Len(t, s.Anims[name].Frames, len(orig.Frames), name)
		for i, f := range orig.Frames {
			assert.Equal(t, f.Image.Bounds().Size(), s.Anims[name].Frames[i].Image.Bounds().Size(), name)
			assert.Equal(t, f.PivotX, s.Anims[name].Frames[i].PivotX, name)
		}
		assert.NotSame(t, orig.Frames[0].Image, strip.Frames[0].Image, name)
	}
	for name := range a.snowArt.sprites {
		assert.Equal(t, a.Props[name].Image.Bounds().Size(), s.Props[name].Image.Bounds().Size(), name)
	}
	assert.Equal(t, a.Props["well"].Image, s.Props["well"].Image, "a prop without a snow picture stays")
}
