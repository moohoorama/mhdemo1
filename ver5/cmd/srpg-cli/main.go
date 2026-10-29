package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"srpg/internal/cli"
	"srpg/internal/content"
	"srpg/internal/session"
)

func main() {
	path := flag.String("content", "assets/data", "content directory")
	data := flag.String("data", "", "save directory")
	seed := flag.Uint64("seed", 1, "new campaign seed")
	jsonMode := flag.Bool("json", false, "JSON Lines protocol")
	slot := flag.String("slot", "", "manual save slot (1~10): loaded at start when it exists, saved when the input ends; lets a script play a few moves per run")
	flag.Parse()
	d, err := content.Load(*path)
	if err != nil {
		fatal(err)
	}
	if *data == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			fatal(err)
		}
		*data = filepath.Join(dir, "srpg-cli")
	}
	s, err := session.New(d, *data)
	if err != nil {
		fatal(err)
	}
	if *jsonMode {
		err = cli.RunJSON(s, os.Stdin, os.Stdout)
	} else {
		if *slot != "" && s.Store.Exists(*slot) {
			r := s.Handle(session.Request{Op: "load", Slot: *slot})
			if r.Error == "ChangedContent" {
				fmt.Fprintf(os.Stderr, "warning: slot %s was saved before the game data changed; loading it anyway\n", *slot)
				r = s.Handle(session.Request{Op: "load", Slot: *slot, Changed: true})
			}
			if !r.OK {
				fatal(fmt.Errorf("load %s: %s", *slot, r.Error))
			}
		}
		err = cli.RunHuman(s, *path, os.Stdin, os.Stdout, *seed)
		if *slot != "" && s.Engine != nil {
			if r := s.Handle(session.Request{Op: "save", Slot: *slot, Overwrite: true}); !r.OK {
				fatal(fmt.Errorf("save %s: %s", *slot, r.Error))
			}
		}
	}
	if err != nil {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
