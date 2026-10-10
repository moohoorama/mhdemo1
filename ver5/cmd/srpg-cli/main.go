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
		err = cli.RunHuman(s, os.Stdin, os.Stdout, *seed)
	}
	if err != nil {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
