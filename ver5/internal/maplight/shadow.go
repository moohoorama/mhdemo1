package maplight

import (
	"image"
	"math"

	"srpg/internal/sprite"
)

// shadowOf is the ground shadow of a prop (cast from its picture, or from the mask its model gave), nil for one
// that casts none.
func (a *Atlas) shadowOf(name string) *Sprite {
	a.cache.mu.Lock()
	defer a.cache.mu.Unlock()
	if s, ok := a.cache.shadows[name]; ok {
		return s
	}
	var out *Sprite
	if src, ok := a.shadowSrc[name]; ok {
		var sd *sprite.Shadow
		if src.mask != nil {
			sd = sprite.MaskShadow(src.sheet.SubImage(*src.mask).(*image.RGBA), src.maskP)
		} else {
			sd = sprite.ImageShadow(src.sheet.SubImage(src.rect).(*image.RGBA), [2]int{int(math.Round(src.pivot[0])), int(math.Round(src.pivot[1]))})
		}
		out = &Sprite{sprite.Tint(sd.Image, a.Shade), float64(sd.Pivot[0]), float64(sd.Pivot[1])}
	}
	a.cache.shadows[name] = out
	return out
}

// maskedShadow reports whether the picture's shadow comes from a mask its model gave: a wall line piece casts a
// ground shadow only then (the old wall pieces cast none).
func (a *Atlas) maskedShadow(name string) bool {
	src, ok := a.shadowSrc[name]
	return ok && src.mask != nil
}

// maxBlit lays s over the shadow layer taking the larger of each channel, so shadows that overlap do not darken
// each other.
func maxBlit(dst *image.RGBA, s *image.RGBA, x, y int) {
	b := s.Bounds()
	for sy := 0; sy < b.Dy(); sy++ {
		for sx := 0; sx < b.Dx(); sx++ {
			px, py := x+sx, y+sy
			if !image.Pt(px, py).In(dst.Rect) {
				continue
			}
			si, di := s.PixOffset(b.Min.X+sx, b.Min.Y+sy), dst.PixOffset(px, py)
			for k := 0; k < 4; k++ {
				dst.Pix[di+k] = max(dst.Pix[di+k], s.Pix[si+k])
			}
		}
	}
}

// laySourceAtop draws the shadow layer over the canvas where the canvas is drawn, leaving the canvas alpha as it is.
func laySourceAtop(canvas, shadows *image.RGBA) {
	for i := 0; i < len(canvas.Pix); i += 4 {
		sa, da := uint32(shadows.Pix[i+3]), uint32(canvas.Pix[i+3])
		if sa == 0 || da == 0 {
			continue
		}
		for k := 0; k < 3; k++ {
			canvas.Pix[i+k] = uint8((uint32(shadows.Pix[i+k])*da + uint32(canvas.Pix[i+k])*(255-sa)) / 255)
		}
	}
}
