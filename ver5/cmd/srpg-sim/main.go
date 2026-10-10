package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"srpg/internal/content"
	"srpg/internal/sim"
	"strings"
)

func main() {
	path := flag.String("content", "assets/data", "content directory")
	duels := flag.Bool("duels", true, "allow stage duels")
	party := flag.String("party", "", "comma-separated character IDs; default full roster")
	count := flag.Int("seeds", 10, "number of seeds")
	flag.Parse()
	d, err := content.Load(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i := 0; i < *count; i++ {
		var ids []string
		if *party != "" {
			ids = strings.Split(*party, ",")
		}
		r, err := sim.RunConfigured(d, uint64(i+1), ids, *duels)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = json.NewEncoder(os.Stdout).Encode(r)
	}
}
