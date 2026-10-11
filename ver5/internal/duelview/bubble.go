package duelview

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

type Bubble struct{ pieces map[string]*ebiten.Image }

func LoadBubble(dir string) (*Bubble, error) {
	b := &Bubble{pieces: map[string]*ebiten.Image{}}
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

func (b *Bubble) piece(dst *ebiten.Image, name string, x, y, w, h float64) {
	im := b.pieces[name]
	bounds := im.Bounds()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(w/float64(bounds.Dx()), h/float64(bounds.Dy()))
	op.GeoM.Translate(x, y)
	dst.DrawImage(im, op)
}

func (b *Bubble) Draw(dst *ebiten.Image, x, y, w, h float64) {
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
	// The tail replaces the border directly beside the lower-left corner.
	b.piece(dst, "tail", x+c, y+h-2, 24, 20)
}
