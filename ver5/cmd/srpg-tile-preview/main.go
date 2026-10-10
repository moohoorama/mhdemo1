package main

import (
	"flag"
	"fmt"
	"os"

	"srpg/internal/gui"
)

func main() {
	out := flag.String("out", "reports/tile-preview", "PNG output directory")
	flag.Parse()
	if err := gui.RenderTilePreview(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
