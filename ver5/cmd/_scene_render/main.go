// _scene_render draws a script's scenes (plan/script/chNN.json "Renders": each scene, its shots and its places A, B …)
// as one PNG per shot next to the script — the whole map with the cast and the places marked — by playing the
// scenespots demo headlessly through cmd/_uidemo-vm.
// Run from ver5: go run ./cmd/_scene_render plan/script/ch10.json   → plan/script/ch10.<scene>.<shot>.png
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/_scene_render plan/script/chNN.json")
		os.Exit(2)
	}
	script := os.Args[1]
	cmd := exec.Command("go", "run", "./cmd/_uidemo-vm", "-ui", "scenespots", "-stage", script, "-frames", "40")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	shots, _ := filepath.Glob(strings.TrimSuffix(script, ".json") + ".*.png")
	for _, s := range shots {
		fmt.Println(s)
	}
}
