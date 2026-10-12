package gui

import (
	"fmt"
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
	labels := []string{"일기토", "", "일기토 건너뛰기", "시작 메뉴"}
	// Select by content identity rather than assuming a particular stage name.
	if p.step == 1 {
		for _, stage := range p.g.S.Data.Stages {
			for _, d := range stage.Duels {
				if d.ID == "hulao-guan-yu" {
					labels[1] = stage.Name + "\n" + p.g.S.Data.Characters[d.Ally].Name + " · " + p.g.S.Data.Characters[d.Enemy].Name
				}
			}
		}
	}
	if p.step >= len(labels) {
		return ebiten.Termination
	}
	if p.step == 1 && len(p.g.buttons) != 10 {
		p.err = fmt.Errorf("expected nine duels and menu, got %d buttons", len(p.g.buttons))
		return ebiten.Termination
	}
	if p.step == 2 && len(p.g.buttons) != 1 {
		p.err = fmt.Errorf("duel viewer has unexpected controls")
		return ebiten.Termination
	}
	for _, b := range p.g.buttons {
		if b.label == labels[p.step] {
			b.click()
			p.step++
			if p.step == 3 && p.g.gallery.view != nil {
				p.err = fmt.Errorf("skip did not return to selection")
			}
			if p.step == 4 && p.g.gallery != nil {
				p.err = fmt.Errorf("menu did not return to title")
			}
			return nil
		}
	}
	p.err = fmt.Errorf("missing button %q", labels[p.step])
	return ebiten.Termination
}
func (p *galleryMenuRenderTest) Draw(dst *ebiten.Image) {
	if p.drawn {
		return
	}
	p.g.Draw(dst)
	p.err = saveDuelFrame(dst, filepath.Join(p.out, fmt.Sprintf("menu-%02d.png", p.step)))
	p.drawn = true
}
