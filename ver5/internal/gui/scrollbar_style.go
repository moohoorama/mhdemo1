package gui

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// scrollBarStyle picks how the common scrollbar (tableBar) is drawn: an index into scrollBarStyles. 0 is the
// bar the game has always had; the others are the dot-art options compared in the scrollbar demo.
var scrollBarStyle = 0

// scrollBarStyles draw a track and its thumb; c is the thumb's ink, which lights up when hovered or dragged.
var scrollBarStyles = [...]func(g *Game, dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA){
	barFlat,
	barArt("scroll_a"),
	barArt("scroll_b"),
	barArt("scroll_c"),
	barArt("scroll_d"),
	barArt("scroll_e"),
	barArt("scroll_a_shade"),
	barArt("scroll_b_shade"),
	barArt("scroll_c_shade"),
	barArt("scroll_d_shade"),
	barArt("scroll_e_shade"),
}

// scrollBarNames are the labels of the styles in the scrollbar demo, in the order of scrollBarStyles.
var scrollBarNames = [...]string{"현행", "A 금테 쪽매", "B 구슬 줄", "C 두루마리 축", "D 옥 장식", "E 점무늬", "A 금테 음영", "B 구슬 음영", "C 두루마리 음영", "D 옥 음영", "E 점무늬 음영"}

// lit is the thumb's hovered or dragged ink (table.go's tableBar).
func lit(c color.NRGBA) bool { return c.R >= 225 }

// barFlat is the bar as it has been: a translucent track and a plain thumb.
func barFlat(g *Game, dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
	rect(dst, float32(track.Min.X), float32(track.Min.Y), float32(track.Dx()), float32(track.Dy()), color.NRGBA{R: 90, G: 90, B: 90, A: 110})
	rect(dst, float32(thumb.Min.X), float32(thumb.Min.Y), float32(thumb.Dx()), float32(thumb.Dy()), c)
}

// barArt draws a dot-art option: the track and the thumb are the pieces named prefix_track and prefix_thumb in the UI
// sheet (tools/spritetool/assets/ver5-ui/seed.py), the thumb's second frame when lit. Each is nine-sliced with 4 px
// corners, so any height keeps its top and foot.
func barArt(prefix string) func(g *Game, dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
	return func(g *Game, dst *ebiten.Image, track, thumb image.Rectangle, c color.NRGBA) {
		nineSlice(dst, g.assets.UI[prefix+"_track"].Frames[0], 4, track)
		frame := 0
		if lit(c) {
			frame = 1
		}
		nineSlice(dst, g.assets.UI[prefix+"_thumb"].Frames[frame], 4, thumb)
	}
}
