package main

import (
	"expvar"
	"flag"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	_ "image/png"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"srpg/internal/content"
	"srpg/internal/gui"
	"srpg/internal/session"
	"strconv"
	"strings"
)

func main() {
	path := flag.String("content", "assets/data", "content directory")
	font := flag.String("font", "assets/fonts/NotoSansKR.ttf", "Korean font")
	dir := flag.String("data", "", "save directory")
	verify := flag.Bool("verify", false, "rendered controller and save/load audit; no native input synthesis")
	audit := flag.String("audit", "", "write GUI input trace and F12 captures")
	seed := flag.Uint64("seed", 1, "new-game seed")
	shots := flag.String("shots", "", "autoplay a new game and save screens at these ticks (60/s), e.g. 600,1200; needs -audit")
	duelID := flag.String("duel", "", "open a duel directly by its scenario ID")
	duelCapture := flag.String("duel-capture", "", "capture each beat of -duel and exit")
	debug := flag.Bool("debug", gui.DebugBuild, "checking tools (non-release builds): scene jumps, duels, map editor, damage/XP scales, autoplay, pprof on "+debugAddr)
	flag.Parse()
	*debug = gui.DebugBuild && *debug
	if err := portableResources(path, font); err != nil {
		fail(err)
	}
	// Launchers may choose an arbitrary working directory. Look beside the executable
	// and in a macOS app bundle, while keeping an existing explicit path.
	if exe, err := os.Executable(); err == nil {
		*path = resourcePath(*path, exe)
		*font = resourcePath(*font, exe)
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
	if *debug {
		serveDebug()
	} else {
		// the scales are chosen only in debug mode; a choice made there is not in force outside it
		s.Settings.DamagePct, s.Settings.XPPct = 0, 0
	}
	g, err := gui.New(s, *font, *seed, *audit)
	if err != nil {
		fail(err)
	}
	defer g.Close()
	if *duelID != "" || *duelCapture != "" {
		if err := g.OpenDuel(*duelID, *duelCapture, *debug); err != nil {
			fail(err)
		}
	}
	g.Verify, g.Debug = *verify, *debug
	if !*verify && *shots == "" && *duelCapture == "" {
		if err := g.EnableSound(); err != nil {
			fmt.Fprintln(os.Stderr, "sound:", err) // the game plays on without it
		}
	}
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
	ebiten.SetWindowTitle(d.Title)
	if icon, err := loadIcon(filepath.Join(*path, "..", "graphics/ui/app_icon.png")); err == nil {
		ebiten.SetWindowIcon([]image.Image{icon})
	}
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err = ebiten.RunGame(g); err != nil {
		fail(err)
	}
}

// resourcePath keeps an existing path and otherwise searches the installed layout.
func resourcePath(path, exe string) string {
	if _, err := os.Stat(path); err == nil || filepath.IsAbs(path) {
		return path
	}
	base := filepath.Dir(exe)
	for _, root := range []string{base, filepath.Join(base, "..", "Resources")} {
		candidate := filepath.Join(root, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return path
}

func loadIcon(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	icon, _, err := image.Decode(f)
	return icon, err
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }

const debugAddr = "127.0.0.1:6060"

// serveDebug serves net/http/pprof (heap, CPU, goroutines) and expvar, which carries the Go memory statistics and
// gpu_image_bytes: the textures ebiten holds, which the Go heap does not show.
func serveDebug() {
	expvar.Publish("gpu_image_bytes", expvar.Func(func() any {
		var d ebiten.DebugInfo
		ebiten.ReadDebugInfo(&d)
		return d.TotalGPUImageMemoryUsageInBytes
	}))
	l, err := net.Listen("tcp", debugAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "debug server:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "debug: http://%s/debug/pprof/ · http://%s/debug/vars\n", debugAddr, debugAddr)
	go http.Serve(l, nil)
}
