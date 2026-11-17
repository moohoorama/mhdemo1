package gui

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/session"
)

func TestRenderedDuelGalleryMenu(t *testing.T) {
	if os.Getenv("SRPG_RENDER_TEST") != "1" {
		t.Skip("set SRPG_RENDER_TEST=1 for the native renderer")
	}
	old := cursorPos
	defer func() { cursorPos = old }()
	d, err := content.Load("../../assets/data")
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.New(d, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 17, "")
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	out := "../../reports/duel-gallery/menu"
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	p := &galleryMenuRenderTest{g: g, out: out}
	result := make(chan error, 1)
	ebiten.SetWindowSize(Width, Height)
	rendererMain <- func() { result <- ebiten.RunGame(p) }
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if p.err != nil {
		t.Fatal(p.err)
	}
}

type galleryMenuRenderTest struct {
	g     *Game
	out   string
	id    string
	step  int
	drawn bool
	err   error
}

func (p *galleryMenuRenderTest) Layout(int, int) (int, int) { return Width, Height }
func (p *galleryMenuRenderTest) Update() error {
	if p.err != nil {
		return ebiten.Termination
	}
	if !p.drawn {
		return nil
	}
	p.drawn = false
	labels := []string{"일기토", "", "일기토 건너뛰기", "닫기"}
	if p.step >= len(labels) {
		return ebiten.Termination
	}
	if p.step == 2 && len(p.g.buttons) != 1 {
		p.err = fmt.Errorf("duel viewer has unexpected controls")
		return ebiten.Termination
	}
	for i := len(p.g.buttons) - 1; i >= 0; i-- {
		b := p.g.buttons[i]
		if b.label != labels[p.step] || p.step == 1 && b.rect.Dx() < 100 {
			continue
		}
		if p.step == 1 {
			cursorPos = func() (int, int) { return b.rect.Min.X + 20, b.rect.Min.Y + 20 }
		}
		b.click()
		if p.step == 0 {
			index := 0
			for _, stage := range p.g.S.Data.Stages {
				for _, d := range stage.Duels {
					p.id = d.ID
					p.g.gallery.list.top = index
					index++
				}
			}
		}
		if p.step == 1 {
			p.g.updateTableDrag()
			if p.g.gallery.id != p.id {
				p.err = fmt.Errorf("wrong duel selected: %s", p.g.gallery.id)
			}
			cursorPos = func() (int, int) { return -100, -100 }
		}
		p.step++
		if p.step == 3 && p.g.gallery.view != nil {
			p.err = fmt.Errorf("skip did not return to selection")
		}
		if p.step == 4 && p.g.gallery != nil {
			p.err = fmt.Errorf("close did not return to title")
		}
		return nil
	}
	p.err = fmt.Errorf("missing button %q", labels[p.step])
	return ebiten.Termination
}
func (p *galleryMenuRenderTest) Draw(dst *ebiten.Image) {
	if p.drawn {
		return
	}
	p.g.fit(Width, Height)
	p.g.Draw(dst)
	if p.g.gallery != nil && p.g.gallery.view == nil {
		for _, b := range p.g.buttons {
			if !b.rect.In(image.Rect(0, 0, uiW, uiH)) {
				p.err = fmt.Errorf("button outside screen: %v", b.rect)
				return
			}
		}
	}
	p.err = saveDuelFrame(dst, filepath.Join(p.out, fmt.Sprintf("menu-%02d.png", p.step)))
	p.drawn = true
}
