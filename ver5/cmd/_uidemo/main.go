// Temporary tool: plays one UI feature in the real game and saves its frames, to compare
// against docs/ui-mockup.
//
//	go run ./cmd/_uidemo -ui turn -out reports/uidemo/turn -frames 120 -every 6
//	go run ./cmd/_uidemo -ui duel -duel <stage duel id>
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
	size := flag.String("size", "1280x850", "window size in device pixels, e.g. 2400x1080 for a phone")
	uiScale := flag.Int("uiscale", 0, "UI size setting in percent (80-160); 0 keeps the default")
	flag.Parse()
	var w, h int
	if _, err := fmt.Sscanf(*size, "%dx%d", &w, &h); err != nil {
		fmt.Fprintln(os.Stderr, "bad -size:", err)
		os.Exit(2)
	}
	if *out == "" {
		*out = "reports/uidemo/" + *name
	}
	d, err := gui.NewDemo(".", gui.DemoOptions{Name: *name, Out: *out, Stage: *stage, Duel: *duel,
		Frames: *frames, Every: *every, UIScale: *uiScale})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("uidemo " + *name)
	if err = ebiten.RunGame(d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
