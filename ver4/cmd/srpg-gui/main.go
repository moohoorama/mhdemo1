package main

import (
	"flag"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"os"
	"path/filepath"
	"srpg/internal/content"
	"srpg/internal/gui"
	"srpg/internal/session"
	"strconv"
	"strings"
)

func main() {
	path := flag.String("content", "assets/content/campaign.json", "content JSON")
	font := flag.String("font", "assets/fonts/NotoSansKR.ttf", "Korean font")
	dir := flag.String("data", "", "save directory")
	verify := flag.Bool("verify", false, "rendered controller and save/load audit; no native input synthesis")
	audit := flag.String("audit", "", "write GUI input trace and F12 captures")
	seed := flag.Uint64("seed", 1, "new-game seed")
	shots := flag.String("shots", "", "autoplay a new game and save screens at these ticks (60/s), e.g. 600,1200; needs -audit")
	flag.Parse()
	// Finder launches apps with an arbitrary working directory. Resolve bundled
	// runtime assets beside the executable while respecting explicit flags.
	if exe, err := os.Executable(); err == nil {
		resources := filepath.Join(filepath.Dir(exe), "..", "Resources")
		if _, err := os.Stat(*path); err != nil {
			if _, err := os.Stat(filepath.Join(resources, *path)); err == nil {
				*path = filepath.Join(resources, *path)
			}
		}
		if _, err := os.Stat(*font); err != nil {
			if _, err := os.Stat(filepath.Join(resources, *font)); err == nil {
				*font = filepath.Join(resources, *font)
			}
		}
	}
	if *dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			fail(err)
		}
		*dir = filepath.Join(base, "srpg-cli")
	}
	d, err := content.Load(*path)
	if err != nil {
		fail(err)
	}
	s, err := session.New(d, *dir)
	if err != nil {
		fail(err)
	}
	g, err := gui.New(s, *font, *seed, *audit)
	if err != nil {
		fail(err)
	}
	defer g.Close()
	g.Verify = *verify
	for _, t := range strings.Split(*shots, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			g.Shots = append(g.Shots, n)
		}
	}
	if len(g.Shots) > 0 && *audit == "" {
		fail(fmt.Errorf("-shots requires -audit directory"))
	}
	if *verify && *audit == "" {
		fail(fmt.Errorf("-verify requires -audit directory"))
	}
	ebiten.SetWindowSize(gui.Width, gui.Height)
	ebiten.SetWindowTitle("도원결의 · 삼국지 SRPG (ver4)")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err = ebiten.RunGame(g); err != nil {
		fail(err)
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
