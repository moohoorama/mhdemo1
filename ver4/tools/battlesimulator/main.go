// Command battlesimulator plays a configured skirmish between two sides: a setup window
// for the sides, terrain and class numbers, then the match with both sides on autoplay;
// or -n matches without a screen for win rates.
//
//	go run ./tools/battlesimulator                 # 설정 창 → 전투 (R 다시, S 설정, F 배속)
//	go run ./tools/battlesimulator -n 200          # 화면 없이 200판 통계
package main

import (
	"flag"
	"fmt"
	"os"
	"srpg/internal/content"
	"srpg/internal/gui"
	"srpg/internal/session"
	"srpg/internal/skirmish"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	config := flag.String("config", "tools/battlesimulator/battle.json", "match settings (설정 저장 writes here)")
	path := flag.String("content", "assets/content/campaign.json", "content JSON")
	maps := flag.String("maps", "assets/content/simulator-maps.json", "simulator maps")
	font := flag.String("font", "assets/fonts/NotoSansKR.ttf", "Korean font")
	n := flag.Int("n", 0, "play this many matches without a screen and print statistics")
	seed := flag.Uint64("seed", 0, "first seed (default: the config's)")
	audit := flag.String("audit", "", "save screens here at the -shots ticks")
	shots := flag.String("shots", "", "ticks (60/s) to capture, e.g. 30,300,900, then quit; the first shows the setup; needs -audit")
	flag.Parse()
	c, err := skirmish.Load(*config)
	if err != nil {
		fail(err)
	}
	if *seed > 0 {
		c.Seed = *seed
	}
	base, err := content.Load(*path)
	if err != nil {
		fail(err)
	}
	terrains, err := skirmish.LoadMaps(*maps)
	if err != nil {
		fail(err)
	}
	if *n > 0 {
		d, err := skirmish.Build(base, terrains, c)
		if err != nil {
			fail(err)
		}
		if err = batch(os.Stdout, d, c, *n); err != nil {
			fail(err)
		}
		return
	}
	dir, err := os.MkdirTemp("", "battlesimulator")
	if err != nil {
		fail(err)
	}
	defer os.RemoveAll(dir)
	s, err := session.New(base, dir)
	if err != nil {
		fail(err)
	}
	s.Screen = "playing"
	g, err := gui.New(s, *font, c.Seed, *audit)
	if err != nil {
		fail(err)
	}
	defer g.Close()
	if g.Skirmish, err = gui.NewSkirmish(base, terrains, c, *config); err != nil {
		fail(err)
	}
	for _, t := range strings.Split(*shots, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			g.Shots = append(g.Shots, n)
		}
	}
	if len(g.Shots) > 0 && *audit == "" {
		fail(fmt.Errorf("-shots requires -audit directory"))
	}
	ebiten.SetWindowSize(gui.Width, gui.Height)
	ebiten.SetWindowTitle("전투 시뮬레이터")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err = ebiten.RunGame(g); err != nil {
		fail(err)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
