package gui

import (
	"srpg/internal/core"
	"srpg/internal/duel"
	"srpg/internal/duelview"
	"srpg/internal/replay"
	"testing"
)

func TestDuelPlaybackAndSkip(t *testing.T) {
	for _, mode := range []string{"finish", "skip duel", "skip replay"} {
		t.Run(mode, func(t *testing.T) {
			g := &Game{vis: map[string]*unitVis{"enemy": {id: "enemy", shownHP: 20, dying: -1}}}
			p := &g.player
			p.g = g
			view := &duelview.View{Story: duel.Story{Beats: []duel.Beat{{Kind: "win", Duration: 2}}}}
			aftermath := false
			p.actions = []*replay.Action{{Actor: "ally"}}
			p.done = make([]bool, 1)
			p.running = []*running{{index: 0, i: -1, beats: []beat{
				{view.Story.Duration() / duelview.PlaybackSpeed, func() { p.Duel = view }},
				{blinkTime, func() { p.Duel = nil; aftermath = true; g.strike(nil, []core.Event{{Kind: "death", Actor: "enemy"}}) }},
			}}}
			p.Update(.01)
			if p.Duel == nil || aftermath {
				t.Fatal("aftermath ran before presentation")
			}
			switch mode {
			case "skip replay":
				p.Skip()
				if p.Busy() || p.Duel != nil {
					t.Fatal("replay skip retained duel")
				}
				return
			case "skip duel":
				p.SkipDuel()
			case "finish":
				p.Update(.5)
				if view.Time != 1 || aftermath {
					t.Fatal("wrong presentation clock")
				}
				p.Update(.5)
			}
			p.Update(.01)
			if p.Duel != nil || !aftermath || !p.Busy() || g.vis["enemy"].shownHP != 0 || g.vis["enemy"].dying != 0 {
				t.Fatal("aftermath was dropped or replay ended too soon")
			}
			p.Update(blinkTime)
			if p.Busy() {
				t.Fatal("replay did not finish after aftermath")
			}
		})
	}
}

func TestDuelAxis(t *testing.T) {
	for _, c := range []struct {
		dx, dy int
		want   string
	}{{1, 0, "SE"}, {0, 1, "SW"}, {-1, 0, "NW"}, {0, -1, "NE"}, {2, 0, "SE"}} {
		if got := duelAxis(c.dx, c.dy); got != c.want {
			t.Errorf("duelAxis(%d,%d) = %s, want %s", c.dx, c.dy, got, c.want)
		}
	}
}
