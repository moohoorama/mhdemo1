package gui

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
	"srpg/internal/autoplay"
	"srpg/internal/content"
	"srpg/internal/session"
)

// TestCampaignEndToEnd plays the whole story through the screen as srpg-gui does: a new game at 쉬움 (casual), the
// debug 자동실행 choosing every story choice, walk, preparation and battle command (the learned model when it was
// trained on this data, otherwise the search player), each frame updated and every few drawn, until the ending.
// The campaign's own waits (the result screen, windows) are skipped and the playback runs fast, so a full story
// takes minutes, not hours. It is long and uses the whole game: run it on its own, by hand,
//
//	SRPG_E2E=1 go test -p 1 ./internal/gui -run TestCampaignEndToEnd -timeout 0 -v
//
// The preparations are drawn from the seed (autoplay.Variety: random sorties and deputies, pieces never worn first,
// 부관임명 first and then random trait goals saved for); SRPG_E2E_PREP=search prepares as srpg-gui does.
// SRPG_E2E_MINUTES caps the wall clock (default 120), SRPG_E2E_SEED picks the game's seed (default 1). Without the
// learned model the search player takes its first choice for every command and preparation, with no playouts ahead
// (a second where it would think one to three); SRPG_E2E_PLAYER=search lets it search as srpg-gui does.
func TestCampaignEndToEnd(t *testing.T) {
	if os.Getenv("SRPG_E2E") != "1" {
		t.Skip("set SRPG_E2E=1 to play the whole story")
	}
	if !DebugBuild {
		t.Skip("자동실행 is a debug control")
	}
	minutes, seed := 120, uint64(1)
	if v, err := strconv.Atoi(os.Getenv("SRPG_E2E_MINUTES")); err == nil && v > 0 {
		minutes = v
	}
	if v, err := strconv.ParseUint(os.Getenv("SRPG_E2E_SEED"), 10, 64); err == nil {
		seed = v
	}
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	s, err := session.New(d, t.TempDir())
	require.NoError(t, err)
	g, err := New(s, "../../assets/fonts/NotoSansKR.ttf", 1, "")
	require.NoError(t, err)
	defer g.Close()
	cursorPos = func() (int, int) { return -1 << 20, -1 << 20 } // no pointer resting over the screen
	g.Debug, g.fastTest = true, true

	r := g.request(session.Request{Op: "new", Seed: seed, Difficulty: "casual"})
	require.True(t, r.OK, r.Error)
	g.toggleCampaignAuto()
	if !g.campaignActive() { // the learned model was trained on other data: the search player plays
		g.campaignRule = true
		g.toggleCampaignAuto()
	}
	require.True(t, g.campaignActive(), "자동실행 starts")
	player := "learned model"
	if g.campaign.model == nil && os.Getenv("SRPG_E2E_PLAYER") != "search" {
		// one candidate each: the tuned AI's first choice, with no playouts ahead of a command or a preparation
		g.campaign.player.K, g.campaign.player.P, g.campaign.player.Par = 1, 1, 1
		player = "fast (first choice)"
	} else if g.campaign.model == nil {
		player = "search"
	}
	t.Logf("player: %s", player)
	var variety *autoplay.Variety
	if os.Getenv("SRPG_E2E_PREP") != "search" {
		// preparations drawn from the seed, so that one run tries many traits, pieces, deputies and sorties; the fourth
		// and fifth tries of a battle prepare by the tuned rules instead (the strongest officers, the best pieces), at
		// once: the search's playouts would take a minute a battle
		variety = autoplay.NewVariety(g.campaign.player.D, g.campaign.player, seed, 4)
		g.campaign.prep = variety
		defer func() { logVariety(t, variety) }()
	}

	run := &campaignRun{t: t, g: g, deadline: time.Now().Add(time.Duration(minutes) * time.Minute), minutes: minutes}
	ebiten.SetWindowSize(Width, Height)
	ebiten.SetWindowTitle("SRPG end-to-end")
	ebiten.SetTPS(600) // many game frames to each drawn one
	ebiten.SetRunnableOnUnfocused(true)
	result := make(chan error, 1)
	rendererMain <- func() { result <- ebiten.RunGame(run) }
	require.NoError(t, <-result)
	if run.err != nil {
		t.Fatal(run.err)
	}
	t.Logf("the story ends after %d battles, %d frames", run.battles, run.frame)
}

// campaignRun is srpg-gui's game loop for the end-to-end test: the game's own Update and Draw, watched each frame.
type campaignRun struct {
	t        *testing.T
	g        *Game
	deadline time.Time
	minutes  int
	frame    int
	battles  int
	last     string
	err      error
}

func (r *campaignRun) Layout(w, h int) (int, int) { return r.g.Layout(w, h) }
func (r *campaignRun) Draw(dst *ebiten.Image)     { r.g.Draw(dst) }
func (r *campaignRun) Update() error {
	g := r.g
	g.speed = 32 // past the speeds a player picks: the playback keeps its order, only faster
	if err := g.Update(); err != nil {
		r.err = err
		return ebiten.Termination
	}
	r.frame++
	o := g.S.View()
	if o.Phase == "complete" {
		return ebiten.Termination
	}
	if o.Phase == "result" && o.Result == "defeat" && r.last != "result "+o.Node {
		st := g.S.Data.Stages[o.Stage]
		why := "a unit the battle needs fell"
		if st.TurnLimit > 0 && o.Round > st.TurnLimit {
			why = fmt.Sprintf("the turn limit (%d)", st.TurnLimit)
		}
		r.t.Logf("defeat at %s round %d: %s", st.ID, o.Round, why)
	}
	if key := fmt.Sprintf("%s %s", o.Phase, o.Node); key != r.last {
		if o.Phase == "battle" && r.last != "" {
			r.battles++
		}
		r.t.Logf("%s %s", time.Now().Format("15:04:05"), key)
		r.last = key
	}
	switch {
	case !g.campaignActive():
		r.err = fmt.Errorf("자동실행 stopped at %s %s (round %d, result %q): %v", o.Phase, o.Node, o.Round, o.Result, g.toastTexts())
	case time.Now().After(r.deadline):
		r.err = fmt.Errorf("still at %s %s after %d minutes", o.Phase, o.Node, r.minutes)
	}
	if r.err != nil {
		return ebiten.Termination
	}
	return nil
}

// logVariety reports what the drawn preparations tried over the run, and how the last ones went.
func logVariety(t *testing.T, v *autoplay.Variety) {
	t.Logf("traits taken %d: %v", len(v.Learned), v.Learned)
	t.Logf("pieces worn %d: %v", len(v.Equipped), v.Equipped)
	t.Logf("deputies %d: %v", len(v.Deputies), v.Deputies)
	t.Logf("sorties: %v", v.Sorties)
	for _, l := range v.Log[max(0, len(v.Log)-5):] {
		t.Log(l)
	}
}

// toastTexts are the messages on screen: why 자동실행 stopped, when it says.
func (g *Game) toastTexts() []string {
	var out []string
	for _, t := range g.toasts {
		out = append(out, t.text)
	}
	return out
}
