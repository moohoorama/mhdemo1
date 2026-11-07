// Temporary tool: plays chapters 14-15 of the story — from 여포 at the gate of 서주 to his shot at the halberd
// (원문사극) — in the real story screen, with a panel for the shot's timing. The scenes are the game's own
// (assets/scenes/ch14.json, ch15.json, staged by tools/scene_maps/ch14_15_scenes.py).
//
//	go run ./cmd/_scene15                    # from the field: 관우·장비·유비 facing 기령 (c15-clash)
//	go run ./cmd/_scene15 -node c14-envoy    # from the edict
//	go run ./cmd/_scene15 -node c15-r19      # from "활을 가져오너라", just before the shot
//	go run ./cmd/_scene15 -c .85 -freeze 6 -aim 1.2 -flight 1   # the shot's camera and timing
//	go run ./cmd/_scene15 -anims             # the motions of every sprite the cast stands in
//
// Click, Enter or Space moves on; F12 saves a screenshot to reports/gui. The panel on the left changes the shot's
// values while it runs ([ ] pick, - = change) and jumps to the field scene's moments (a g s d f); h hides it. Headless capture of the same chapter:
//
//	go run ./cmd/_uidemo-vm -ui transition -node c15-clash -frames 3000 -mp4 reports/uidemo/scene15.mp4
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"srpg/internal/content"
	"srpg/internal/gui"
	"srpg/internal/scenes"
	"srpg/internal/session"
	"srpg/internal/sprite"
)


func main() {
	node := flag.String("node", "c15-clash", "story node to start from")
	t := &gui.ShotTune
	flag.Float64Var(&t.Chase, "c", t.Chase, "arrow camera: S = S*c + A*(1-c) every frame as the arrow leaves; higher stays further behind")
	flag.Float64Var(&t.ChaseEnd, "c2", t.ChaseEnd, "c as the arrow arrives (lower: the camera hurries after it)")
	flag.Float64Var(&t.Damping, "bounce", t.Damping, "landing spring damping: under 1 the camera runs past the halberd and swings back")
	flag.Float64Var(&t.Stiff, "stiff", t.Stiff, "landing spring strength")
	flag.IntVar(&t.Freeze, "freeze", t.Freeze, "frames the camera does not move after the arrow leaves")
	flag.Float64Var(&t.Aim, "aim", t.Aim, "seconds the bow is held drawn while the camera glides")
	flag.Float64Var(&t.Flight, "flight", t.Flight, "seconds the arrow is in the air (0: the scene's own, 1)")
	flag.Float64Var(&t.Peak, "peak", t.Peak, "arc height in canvas px (0: the scene's own, 60)")
	flag.Float64Var(&t.ThrowPeak, "throw", t.ThrowPeak, "the thrown halberd's arc height in canvas px")
	flag.Float64Var(&t.ThrowTime, "throwtime", t.ThrowTime, "the thrown halberd's seconds in the air")
	flag.Float64Var(&t.Hold, "hold", t.Hold, "seconds on the struck halberd before going back")
	flag.Float64Var(&t.Arrow[0], "ax", t.Arrow[0], "screen x (0-1) the camera keeps the arrow at")
	flag.Float64Var(&t.Arrow[1], "ay", t.Arrow[1], "screen y (0-1) the camera keeps the arrow at")
	flag.Float64Var(&t.Shooter[0], "sx", t.Shooter[0], "screen x (0-1) the shooter ends the glide at")
	flag.Float64Var(&t.Shooter[1], "sy", t.Shooter[1], "screen y (0-1) the shooter ends the glide at")
	anims := flag.Bool("anims", false, "print the motions of the cast's sprites and exit")
	flag.Parse()
	d, err := content.Load("assets/data")
	check(err)
	first := "c14-gate"
	if strings.HasPrefix(*node, "c15-") {
		first = "c15-grain"
	}
	if *anims {
		check(printAnims(d))
		return
	}
	if *node == "" {
		*node = first
	}
	s, err := session.New(d, filepath.Join(os.TempDir(), "scene15"))
	check(err)
	s.Settings.DamagePct, s.Settings.XPPct = 0, 0
	check(gui.StartTrial(s, first, *node))
	g, err := gui.New(s, "assets/fonts/NotoSansKR.ttf", 1, "")
	check(err)
	defer g.Close()
	ebiten.SetWindowSize(gui.Width, gui.Height)
	ebiten.SetWindowTitle("원문사극 미리보기")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if t.Flight <= 0 {
		t.Flight = 1
	}
	if t.Peak <= 0 {
		t.Peak = 60
	}
	tn, err := newTuner(g, s, first)
	check(err)
	check(ebiten.RunGame(tn))
}

func readJSON(name string, v any) error {
	raw, err := os.ReadFile(filepath.Join("assets/scenes", name))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// printAnims lists, for every cast member of the trial's scenes, the sprite it stands in and that sprite's motions.
func printAnims(d *content.Data) error {
	var f, f15 struct{ Scenes map[string]*scenes.Scene }
	if err := readJSON("ch14.json", &f); err != nil {
		return err
	}
	if err := readJSON("ch15.json", &f15); err != nil {
		return err
	}
	maps.Copy(f.Scenes, f15.Scenes)
	names := map[string]string{}
	for id, c := range d.Characters {
		names[c.Name] = id
	}
	seen := map[string]bool{}
	for _, sc := range f.Scenes {
		for key, a := range sc.Cast {
			c := d.Characters[names[scenes.Base(key)]]
			ids := []string{a.Sprite, c.Look.Scene, c.Look.Sprite}
			for _, st := range allSteps(sc) {
				if st.Actor == key && st.Sprite != "" {
					ids = append(ids, st.Sprite)
				}
			}
			for _, id := range ids {
				if id == "" || seen[id] {
					continue
				}
				seen[id] = true
				sh, err := sprite.Load(filepath.Join("assets/sprites", id+".spr"))
				if err != nil {
					fmt.Printf("%-24s %s: %v\n", id, key, err)
					continue
				}
				motions := []string{}
				for m := range sh.Animations {
					motions = append(motions, m)
				}
				sort.Strings(motions)
				fmt.Printf("%-24s %-8s %v\n", id, key, motions)
			}
		}
	}
	return nil
}

func allSteps(sc *scenes.Scene) []scenes.Step {
	out := slices.Clone(sc.Ambient)
	for _, lines := range sc.Beats {
		for _, groups := range lines {
			for _, g := range groups {
				out = append(out, g...)
			}
		}
	}
	return out
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
