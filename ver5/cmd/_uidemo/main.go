// Temporary tool: plays one UI feature in the real game and saves its frames, to compare
// against docs/ui-mockup.
//
//	go run ./cmd/_uidemo -ui turn -out reports/uidemo/turn -frames 120 -every 6
//	go run ./cmd/_uidemo -ui duel -duel <stage duel id>
//	go run ./cmd/_uidemo -ui sceneskip -node <story node id>
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/gui"
)

func main() {
	name := flag.String("ui", "", "demo to play: "+strings.Join(gui.DemoNames(), ", "))
	out := flag.String("out", "", "frame directory (default reports/uidemo/<ui>)")
	frames := flag.Int("frames", 120, "frames to play (60 per second)")
	every := flag.Int("every", 6, "save every n-th frame; 0 saves none (vmhost guest)")
	stage := flag.String("stage", "", "battle stage the demos play (default: the first)")
	duel := flag.String("duel", "", "stage duel the duel demo plays (default: the first)")
	node := flag.String("node", "", "story node the scene demos jump to")
	size := flag.String("size", "1280x850", "window size in device pixels, e.g. 2400x1080 for a phone")
	uiScale := flag.Int("uiscale", 0, "UI size setting in percent (80-160); 0 keeps the default")
	debug := flag.Bool("debug", false, "show what srpg-gui -debug shows")
	cpuProfile := flag.String("cpuprofile", "", "write a CPU profile of the whole run (loading included) to this file")
	flag.Parse()
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}
	if err := run(*name, *out, *stage, *duel, *node, *size, *frames, *every, *uiScale, *debug); err != nil {
		fmt.Fprintln(os.Stderr, err)
		pprof.StopCPUProfile()
		os.Exit(1)
	}
}

func run(name, out, stage, duel, node, size string, frames, every, uiScale int, debug bool) error {
	var w, h int
	if _, err := fmt.Sscanf(size, "%dx%d", &w, &h); err != nil {
		return fmt.Errorf("bad -size: %w", err)
	}
	if out == "" {
		out = "reports/uidemo/" + name
	}
	start := time.Now()
	d, err := gui.NewDemo(".", gui.DemoOptions{Name: name, Out: out, Stage: stage, Duel: duel, Node: node,
		Frames: frames, Every: every, UIScale: uiScale, Debug: debug})
	if err != nil {
		return err
	}
	loaded := time.Now()
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("uidemo " + name)
	err = ebiten.RunGame(d)
	fmt.Fprintf(os.Stderr, "guest: loading %.1fs, playing %d frames %.1fs\n", loaded.Sub(start).Seconds(), frames, time.Since(loaded).Seconds())
	return err
}
