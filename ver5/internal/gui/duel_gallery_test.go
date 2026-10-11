package gui

import (
	"math"
	"reflect"
	"testing"

	"srpg/internal/content"
	"srpg/internal/core"
	"srpg/internal/session"
)

func TestDuelGalleryPreservesCampaign(t *testing.T) {
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
	s.Engine = core.New(d, 17)
	before := s.Engine.Snapshot()
	screen := s.Screen
	clock := g.clockMS
	if err := g.OpenDuel("", "", false); err != nil {
		t.Fatal(err)
	}
	if g.gallery.view != nil {
		t.Fatal("menu started a duel")
	}
	for _, st := range d.Stages {
		for _, pair := range st.Duels {
			t.Run(pair.ID, func(t *testing.T) {
				if err := g.OpenDuel(pair.ID, "", false); err != nil {
					t.Fatal(err)
				}
				view := g.gallery.view
				for i := 0; i < 2+len(g.assets.Duels[pair.ID].Supports); i++ {
					if view.Fighters[i].Art == nil || view.Fighters[i].Name == "" {
						t.Fatal("missing cast")
					}
				}
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
				if math.Abs(view.Time-2.0/60) > 1e-12 {
					t.Fatal("gallery is not playing at exactly twice its previous speed")
				}
				for frame := 0; frame < 4000 && g.gallery.view != nil; frame++ {
					if err := g.Update(); err != nil {
						t.Fatal(err)
					}
				}
				if g.gallery.view != nil {
					t.Fatal("viewer never returned to selection")
				}
				if !reflect.DeepEqual(before, s.Engine.Snapshot()) || s.Screen != screen || g.clockMS != clock || g.player.Busy() {
					t.Fatal("gallery advanced the campaign")
				}
			})
		}
	}
	gallery := g.gallery
	if err := g.OpenDuel("missing", "", false); err == nil || g.gallery != gallery {
		t.Fatal("invalid ID should leave the viewer unchanged")
	}
	if err := g.OpenDuel("", t.TempDir(), false); err == nil {
		t.Fatal("capture requires a story")
	}
}
