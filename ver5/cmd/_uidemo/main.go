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
	flag.Parse()
	if *out == "" {
		*out = "reports/uidemo/" + *name
	}
	d, err := gui.NewDemo(".", *name, *out, *stage, *duel, *frames, *every)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ebiten.SetWindowSize(gui.Width, gui.Height)
	ebiten.SetWindowTitle("uidemo " + *name)
	if err = ebiten.RunGame(d); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
