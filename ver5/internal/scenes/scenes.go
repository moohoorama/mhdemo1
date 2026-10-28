// Package scenes reads the story scenes (assets/scenes/*.json): for each stretch of story nodes, the scene map,
// who stands where, and what they do before each line. It is GUI data, not rules content; the GUI plays it and the
// CLI prints it in letters.
package scenes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Step is one thing a beat does. Steps of a group start together; the group ends when every walk, motion, bubble
// and wait of it has ended.
type Step struct {
	Actor  string
	Move   []float64   // walk to [u, v], along u first, then v
	Via    [][]float64 // corners [u, v] the walk to Move passes first, in order, each reached along u first
	Enter  []float64   // appear at [u, v] first (an actor that starts off the stage), then walk to Move
	Exit   bool        // leave the stage when the walk ends
	Speed  float64     // walking speed, 1 when 0 (2 runs, .6 drags)
	After  float64     // seconds after the group starts
	Face   string
	Look   string // turn this way for a moment, then back
	Play   string
	Emote  string
	Say    string // a speech bubble on the map, without the dialogue scroll
	Wait   float64
	Intro  string // an officer's first appearance
	Effect string // petals, fire, smallfire, smoke, steam, birds, glint, none
	At     []float64
	Shake  float64 // seconds of screen shake
	Cam    string  // look at this actor
	CamAt  []float64
	Zoom   float64 // camera zoom, whole numbers only (pixel art)
	Time   string  // day (""), dusk, night or dawn from now on
	Sprite string  // the actor stands as this sprite from now on
	Prop   []any   // [name, u, v] puts a map prop down, [name] takes it away, [name, actor] makes it move with that actor; "name#2" is one more of that prop
	Pose   string  // the motion the actor rests in from now on (sit, lie, kneel …); "idle" stands again
	Patrol []float64
	// walk to [u, v] and back, over and over, without holding up the beats (a sentry); a Move ends it
	Chatter string // short bubbles the actor keeps saying now and then, "|" between them; "-" stops
}

// Actor is where a cast member starts. A cast key with "#" ("병사#2") is an extra: no name plate, no portrait;
// the part before "#" may still be a character.
type Actor struct {
	At      []float64
	Face    string
	Faction string
	Sprite  string
	Off     bool   // starts off the stage, until a step Enters it
	Hold    bool   // never turns by itself toward whoever speaks
	Pose    string // the motion it rests in (sit, lie, kneel …) instead of standing idle
	Stay    bool   // on an explore walk's own stage: stands where the walk left it, with no At or Face of its own
}

// Scene is one stretch of story nodes staged on one map.
type Scene struct {
	ID       string `json:"-"`
	Map      string
	Time     string
	Backdrop string // the far picture behind an outdoor map (assets/graphics/backdrops)
	Nodes    []string
	Ambient  []Step // run once as the scene opens (fires, braziers, smoke, sentries, chatter)
	Cast     map[string]Actor
	Beats    map[string][][][]Step // node -> per speaker line -> groups of steps
}

type file struct {
	Scenes map[string]*Scene
}

// Book is every scene, and which scene each node is staged in.
type Book struct {
	Scenes map[string]*Scene
	byNode map[string]*Scene
}

// Load reads every assets/scenes/*.json. A node staged in two scenes is an error.
func Load(dir string) (*Book, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no scene files in %s", dir)
	}
	sort.Strings(paths)
	b := &Book{Scenes: map[string]*Scene{}, byNode: map[string]*Scene{}}
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var f file
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(p), err)
		}
		for id, s := range f.Scenes {
			if _, dup := b.Scenes[id]; dup {
				return nil, fmt.Errorf("%s: scene %s is defined twice", filepath.Base(p), id)
			}
			s.ID = id
			b.Scenes[id] = s
			for _, n := range s.Nodes {
				if other := b.byNode[n]; other != nil {
					return nil, fmt.Errorf("node %s is in scenes %s and %s", n, other.ID, id)
				}
				b.byNode[n] = s
			}
		}
	}
	return b, nil
}

// For is the scene node is staged in and the node's place in it; nil when it has none.
func (b *Book) For(node string) (*Scene, int) {
	if b == nil {
		return nil, -1
	}
	s := b.byNode[node]
	if s == nil {
		return nil, -1
	}
	for i, n := range s.Nodes {
		if n == node {
			return s, i
		}
	}
	return nil, -1
}

// Line is the groups that play before speaker line i of node.
func (s *Scene) Line(node string, i int) [][]Step {
	if lines := s.Beats[node]; i < len(lines) {
		return lines[i]
	}
	return nil
}

// Extra reports whether a cast key names an extra ("병사#2").
func Extra(key string) bool { return strings.Contains(key, "#") }

// Base is the character name of a cast key: the part before "#".
func Base(key string) string {
	base, _, _ := strings.Cut(key, "#")
	return base
}
