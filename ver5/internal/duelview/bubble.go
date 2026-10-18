package duelview

import (
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	bubbleCorner   = 16.0
	bubbleTailH    = 20.0
	bubblePadX     = 20.0
	bubblePadTop   = 14.0
	bubblePadEnd   = 18.0
	bubbleNameSize = 16.0
	bubbleBodySize = 17.0
	bubbleSpacing  = bubbleBodySize * 1.45
	bubbleMaxText  = 340.0
	bubbleMinW     = 64.0
	bubbleMinH     = 44.0
	bubbleCacheMax = 64
)

// Tail is where a balloon's pointer sits: X is the tip's distance from the balloon's left
// edge in design pixels (0 is the default beside the lower-left corner), and Up puts the
// pointer on the top edge for a balloon that hangs below its speaker.
type Tail struct {
	X  float64
	Up bool
}

const tailHalf = 12.0

func (t Tail) tipX(w float64) float64 {
	if t.X == 0 {
		return bubbleCorner + tailHalf
	}
	return max(bubbleCorner+tailHalf, min(w-bubbleCorner-tailHalf, math.Round(t.X/4)*4))
}

type bubbleKey struct {
	name, body string
	r          int
	tail       Tail
}

// Bubble draws speech balloons: a 9-slice frame of PNG pieces with a bold name line above
// the wrapped body. Rendered balloons are cached by text, so a balloon that stays on
// screen is laid out once.
type Bubble struct {
	pieces map[string]*ebiten.Image
	font   *text.GoTextFaceSource
	cache  map[bubbleKey]*ebiten.Image
}

func LoadBubble(dir string, font *text.GoTextFaceSource) (*Bubble, error) {
	b := &Bubble{pieces: map[string]*ebiten.Image{}, font: font, cache: map[bubbleKey]*ebiten.Image{}}
	for _, name := range []string{"top-left", "top-right", "bottom-left", "bottom-right", "top", "bottom", "left", "right", "center", "tail"} {
		path := filepath.Join(dir, name+".png")
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		im, err := png.Decode(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		b.pieces[name] = ebiten.NewImageFromImage(im)
	}
	return b, nil
}

// Size is the balloon's pixel size including its tail, which adds bubbleTailH above or below.
func (b *Bubble) Size(name, body string) (w, h float64) {
	im := b.image(name, body)
	return float64(im.Bounds().Dx()), float64(im.Bounds().Dy())
}

// DrawText draws the balloon with its top-left corner at (x, y), enlarged by scale. A blank
// name leaves out the name line.
func (b *Bubble) DrawText(dst *ebiten.Image, name, body string, tail Tail, x, y, scale float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	op.GeoM.Scale(scale, scale)
	dst.DrawImage(b.imageAt(name, body, 1, tail), op)
}

// DrawSharp draws the balloon with its top-left corner at (x, y) on a screen where one design
// pixel is scale device pixels: the text is rasterised at that size instead of being
// stretched from the design size.
func (b *Bubble) DrawSharp(dst *ebiten.Image, name, body string, tail Tail, x, y, scale, alpha float64) {
	r := max(1, min(4, int(math.Ceil(scale))))
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(scale/float64(r), scale/float64(r))
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleAlpha(float32(alpha))
	dst.DrawImage(b.imageAt(name, body, r, tail), op)
}

func (b *Bubble) image(name, body string) *ebiten.Image { return b.imageAt(name, body, 1, Tail{}) }

func (b *Bubble) imageAt(name, body string, r int, tail Tail) *ebiten.Image {
	key := bubbleKey{name, body, r, tail}
	if im, ok := b.cache[key]; ok {
		return im
	}
	if len(b.cache) >= bubbleCacheMax {
		for k, im := range b.cache {
			im.Deallocate()
			delete(b.cache, k)
		}
	}
	im := b.render(name, body, r, tail)
	b.cache[key] = im
	return im
}

func (b *Bubble) face(size float64, weight float32) *text.GoTextFace {
	f := &text.GoTextFace{Source: b.font, Size: size}
	f.SetVariation(text.MustParseTag("wght"), weight)
	return f
}

func (b *Bubble) render(name, body string, r int, tail Tail) *ebiten.Image {
	bodyFace, nameFace := b.face(bubbleBodySize, 400), b.face(bubbleNameSize, 800)
	body = wrapText(body, bodyFace, bubbleMaxText)
	textW, textH := text.Measure(body, bodyFace, bubbleSpacing)
	textTop, textLeft := bubblePadTop, bubblePadX
	var top float64 // the body's offset below the image top, where an upward tail sits
	if tail.Up {
		top = bubbleTailH
	}
	if name != "" {
		nameW, _ := text.Measure(name, nameFace, bubbleNameSize*1.45)
		textW = max(textW, nameW)
		textTop += 27
	}
	if body == "" {
		textH = 0
	}
	w := max(bubbleMinW, textW+2*bubblePadX)
	h := max(bubbleMinH, textTop+textH+bubblePadEnd)
	if name == "" { // a mark or dots sits in the middle
		textLeft = (w - textW) / 2
	}
	textTop += top
	tipX := tail.tipX(w)
	iw, ih := int(w+0.999), int(h+bubbleTailH+0.999)
	im := ebiten.NewImage(iw*r, ih*r)
	if r == 1 {
		b.draw(im, 0, top, w, h, tipX, tail.Up)
	} else {
		frame := ebiten.NewImage(iw, ih)
		b.draw(frame, 0, top, w, h, tipX, tail.Up)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Scale(float64(r), float64(r))
		im.DrawImage(frame, op)
		frame.Deallocate()
		bodyFace, nameFace = b.face(bubbleBodySize*float64(r), 400), b.face(bubbleNameSize*float64(r), 800)
	}
	k := float64(r)
	ink := color.NRGBA{40, 37, 34, 255}
	if name != "" {
		op := &text.DrawOptions{}
		op.GeoM.Translate(bubblePadX*k, (bubblePadTop+top)*k)
		op.ColorScale.ScaleWithColor(ink)
		text.Draw(im, name, nameFace, op)
	}
	op := &text.DrawOptions{}
	op.LineSpacing = bubbleSpacing * k
	op.GeoM.Translate(textLeft*k, textTop*k)
	op.ColorScale.ScaleWithColor(ink)
	text.Draw(im, body, bodyFace, op)
	return im
}

// wrapText breaks s into lines no wider than maxW, preferring spaces; explicit newlines are kept.
func wrapText(s string, f *text.GoTextFace, maxW float64) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		r := []rune(line)
		for {
			n := len(r)
			for n > 0 {
				if w, _ := text.Measure(string(r[:n]), f, 0); w <= maxW {
					break
				}
				n--
			}
			if n >= len(r) || n == 0 {
				break
			}
			cut := n
			for i := n; i > n/2; i-- {
				if r[i] == ' ' {
					cut = i
					break
				}
			}
			out = append(out, strings.TrimRight(string(r[:cut]), " "))
			r = []rune(strings.TrimLeft(string(r[cut:]), " "))
		}
		out = append(out, string(r))
	}
	return strings.Join(out, "\n")
}

func (b *Bubble) piece(dst *ebiten.Image, name string, x, y, w, h float64) {
	im := b.pieces[name]
	bounds := im.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/float64(bounds.Dx()), h/float64(bounds.Dy()))
	op.GeoM.Translate(x, y)
	dst.DrawImage(im, op)
}

func (b *Bubble) pieceFlipped(dst *ebiten.Image, name string, x, y, w, h float64) {
	im := b.pieces[name]
	bounds := im.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/float64(bounds.Dx()), -h/float64(bounds.Dy()))
	op.GeoM.Translate(x, y+h)
	dst.DrawImage(im, op)
}

func (b *Bubble) draw(dst *ebiten.Image, x, y, w, h, tipX float64, up bool) {
	const c = 16.0
	b.piece(dst, "center", x+c, y+c, w-2*c, h-2*c)
	b.piece(dst, "top", x+c, y, w-2*c, c)
	b.piece(dst, "bottom", x+c, y+h-c, w-2*c, c)
	b.piece(dst, "left", x, y+c, c, h-2*c)
	b.piece(dst, "right", x+w-c, y+c, c, h-2*c)
	b.piece(dst, "top-left", x, y, c, c)
	b.piece(dst, "top-right", x+w-c, y, c, c)
	b.piece(dst, "bottom-left", x, y+h-c, c, c)
	b.piece(dst, "bottom-right", x+w-c, y+h-c, c, c)
	// The tail replaces the border where the tip points at the speaker.
	if up {
		b.pieceFlipped(dst, "tail", x+tipX-tailHalf, y-bubbleTailH+2, 2*tailHalf, bubbleTailH)
		return
	}
	b.piece(dst, "tail", x+tipX-tailHalf, y+h-2, 2*tailHalf, bubbleTailH)
}
