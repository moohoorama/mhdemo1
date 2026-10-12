package gui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"srpg/internal/duelview"
)

type duelGallery struct {
	view          *duelview.View
	id            string
	debug, paused bool
	captureDir    string
	shot          int
	drawn         bool
	err           error
}

// OpenDuel opens the independent viewer without issuing a campaign command.
// An empty ID opens the selection menu.
func (g *Game) OpenDuel(id, captureDir string, debug bool) error {
	gallery := &duelGallery{debug: debug, captureDir: captureDir}
	if id == "" {
		if captureDir != "" {
			return fmt.Errorf("duel capture requires a duel ID")
		}
		g.gallery = gallery
		return nil
	}
	for _, stage := range g.S.Data.Stages {
		for _, d := range stage.Duels {
			if d.ID != id {
				continue
			}
			v := &duelview.View{Font: g.font, Bubble: g.assets.Bubble, Story: g.assets.Duels.Story(id, d.Outcome)}
			ids := append([]string{d.Ally, d.Enemy}, g.assets.Duels[id].Supports...)
			for i, actor := range ids {
				c, ok := g.S.Data.Characters[actor]
				if !ok {
					return fmt.Errorf("unknown duel fighter %q", actor)
				}
				faction := c.Look.Faction
				if i == 1 {
					if faction == "" {
						faction = stage.Faction
					}
					for _, spawn := range stage.Enemies {
						if spawn.Character == actor {
							if spawn.Class != "" {
								c.Class = spawn.Class
							}
							if spawn.Faction != "" {
								faction = spawn.Faction
							}
						}
					}
				}
				v.Fighters[i] = g.duelFighter(actor, c.Class, faction)
			}
			gallery.id, gallery.view = id, v
			if captureDir != "" {
				if err := os.MkdirAll(captureDir, 0755); err != nil {
					return err
				}
				gallery.prepareCapture()
			}
			g.gallery = gallery
			return nil
		}
	}
	return fmt.Errorf("unknown duel %q", id)
}

func (p *duelGallery) prepareCapture() {
	b := p.view.Story.Beats[p.shot]
	local := b.Duration * .75
	switch b.Kind {
	case "hit", "block", "critical-block", "critical-dodge", "yielding-block", "critical", "backhit", "dodge":
		local = .57
	}
	p.view.Time = p.view.Story.Start(p.shot) + local
	p.drawn = false
}

func (g *Game) updateDuelGallery() error {
	p := g.gallery
	if p.err != nil {
		return p.err
	}
	if p.captureDir != "" {
		if !p.drawn {
			return nil
		}
		p.shot++
		if p.shot >= len(p.view.Story.Beats) {
			return ebiten.Termination
		}
		p.prepareCapture()
		return nil
	}
	if p.view != nil {
		if !p.paused {
			p.view.Time += duelview.PlaybackSpeed / 60
		}
		if p.debug {
			if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
				p.paused = !p.paused
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyR) {
				p.view.Time = 0
			}
			for key, delta := range map[ebiten.Key]int{ebiten.KeyArrowLeft: -1, ebiten.KeyArrowRight: 1} {
				if inpututil.IsKeyJustPressed(key) {
					index := p.view.Story.Sample(p.view.Time).Index + delta
					p.view.Time = p.view.Story.Start(max(0, min(index, len(p.view.Story.Beats)-1)))
				}
			}
		}
		if p.view.Time >= p.view.Story.Duration() || inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) || !p.debug && inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			p.view = nil
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.gallery = nil
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		g.capture = true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := cursorPos()
		for _, b := range g.buttons {
			if image.Pt(x, y).In(b.rect) {
				b.click()
				break
			}
		}
	}
	return nil
}

func (g *Game) drawDuelGallery(dst *ebiten.Image) {
	p := g.gallery
	if p.view == nil {
		g.label(dst, "일기토", 96, 70, 48, gold)
		n := 0
		for _, stage := range g.S.Data.Stages {
			for _, d := range stage.Duels {
				duelID := d.ID
				x, y := 65+(n%3)*390, 180+(n/3)*180
				label := stage.Name + "\n" + g.S.Data.Characters[d.Ally].Name + " · " + g.S.Data.Characters[d.Enemy].Name
				g.btn(dst, image.Rect(x, y, x+360, y+135), label, func() {
					if err := g.OpenDuel(duelID, "", p.debug); err != nil {
						g.toast(err.Error(), bad)
					}
				})
				n++
			}
		}
		g.btn(dst, image.Rect(1000, 805, 1254, 839), "시작 메뉴", func() { g.gallery = nil })
		return
	}
	p.view.Draw(dst)
	g.btn(dst, image.Rect(1000, 805, 1254, 839), "일기토 건너뛰기", func() { p.view = nil })
	if p.debug {
		state := p.view.Story.Sample(p.view.Time)
		g.label(dst, state.Stage, 400, 30, 20, gold)
		g.label(dst, "Space 정지 · R 처음 · ←/→ 장면 이동", 30, 805, 17, muted)
	}
	if p.captureDir != "" && !p.drawn {
		path := filepath.Join(p.captureDir, fmt.Sprintf("%s-%02d.png", p.id, p.shot))
		p.err = saveDuelFrame(dst, path)
		p.drawn = true
	}
}

func saveDuelFrame(dst *ebiten.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	im := image.NewRGBA(dst.Bounds())
	dst.ReadPixels(im.Pix)
	err = png.Encode(f, im)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
