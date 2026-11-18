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
	"srpg/internal/core"
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
//
// SRPG_E2E_ROUTE picks the telling from 24장 융중대 on (make e2e E2E_ROUTE=virtual):
//   - yanyi (default): 자동실행 takes 「빠른 길」 whenever the 융중 choice comes; without the choice (고순 gone, 12장's
//     부목표 missed) the story is 연의 by itself. The run ends at c27-end.
//   - virtual: 자동실행 takes 「느린 길」, and the run makes sure the choice comes. The choice needs 고순 in the party,
//     flag-zhugexuan (12장's bonus objective) and flag-guanyu-mifang, which 22장 c22-mifang sets only while 고순 is
//     in the party. When the story reaches c22-start (and again at c24-start), the test sets each missing flag on the
//     engine's state (Snapshot → core.Restore, as a save edited by hand) and logs what it set; 22장 then plays its
//     관우·미방 scene by its own branch. A 고순 who is no longer in the party cannot be brought back this way: the run
//     fails there, since that is a seed where 12장 or 18장 lost him. The run ends at c27v-end.
//
// Either way the mode flags (flag-mode-yanyi, flag-mode-virtual) are logged at the end and checked against the route.
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
	route := os.Getenv("SRPG_E2E_ROUTE")
	switch route {
	case "", "yanyi":
		route = "yanyi"
		g.campaign.prefer = []string{"빠른 길"}
	case "virtual":
		g.campaign.prefer = []string{"느린 길"}
	default:
		t.Fatalf("SRPG_E2E_ROUTE %q: yanyi or virtual", route)
	}
	t.Logf("route: %s", route)
	var variety *autoplay.Variety
	if os.Getenv("SRPG_E2E_PREP") != "search" {
		// preparations drawn from the seed, so that one run tries many traits, pieces, deputies and sorties; the fourth
		// and fifth tries of a battle prepare by the tuned rules instead (the strongest officers, the best pieces), at
		// once: the search's playouts would take a minute a battle
		variety = autoplay.NewVariety(g.campaign.player.D, g.campaign.player, seed, 4)
		g.campaign.prep = variety
		defer func() { logVariety(t, variety) }()
	}

	run := &campaignRun{t: t, g: g, deadline: time.Now().Add(time.Duration(minutes) * time.Minute), minutes: minutes, route: route, opened: map[string]bool{}}
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
	flags := g.S.Engine.Snapshot().Flags
	t.Logf("the story ends after %d battles, %d frames; mode flags: flag-mode-yanyi=%d flag-mode-virtual=%d",
		run.battles, run.frame, flags["flag-mode-yanyi"], flags["flag-mode-virtual"])
	want := map[string]string{"yanyi": "flag-mode-yanyi", "virtual": "flag-mode-virtual"}[route]
	require.Equal(t, 1, flags[want], "the %s route was played", route)
}

// openVirtual makes the 융중 choice come on the virtual route: on the first scenario frame of 22장 and of 24장 it sets the flags the
// choice needs that the run has not earned (see TestCampaignEndToEnd), on the engine's state.
func (r *campaignRun) openVirtual(at string) error {
	g := r.g
	s := g.S.Engine.Snapshot()
	if !g.S.Engine.Holds([]string{"party:gaoshun"}) {
		return fmt.Errorf("고순 is not in the party at %s: the 가상 route cannot open on this seed", at)
	}
	need := []string{"flag-zhugexuan"}
	if at == "c24-start" {
		need = append(need, "flag-guanyu-mifang")
	}
	var set []string
	for _, f := range need {
		if s.Flags[f] == 0 {
			s.Flags[f] = 1
			set = append(set, f)
		}
	}
	if len(set) == 0 {
		r.t.Logf("route virtual: at %s (%s) the conditions already hold", at, g.S.View().Node)
		return nil
	}
	e, err := core.Restore(g.S.Data, s)
	if err != nil {
		return err
	}
	g.S.Engine, g.S.Dirty = e, true
	r.t.Logf("route virtual: at %s (%s) set %v", at, g.S.View().Node, set)
	return nil
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
	route    string
	opened   map[string]bool
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
	for _, at := range []string{"c22-start", "c24-start"} { // the first scenario frame in each chapter
		if r.route == "virtual" && !r.opened[at] && o.Phase == "scenario" && g.S.Data.ChapterIndex(o.Node) >= g.S.Data.ChapterIndex(at) {
			r.opened[at] = true
			if r.err = r.openVirtual(at); r.err != nil {
				return ebiten.Termination
			}
			o = g.S.View()
		}
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
