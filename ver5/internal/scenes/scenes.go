// Package scenes reads the story scenes (assets/scenes/*.json): for each stretch of story nodes, the scene map,
// who stands where, and what they do before each line. It is GUI data, not rules content; the GUI plays it and the
// CLI prints it in letters.
package scenes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"srpg/internal/content"
)

// Step is one thing a beat does. Steps of a group start together; the group ends when every walk, motion, bubble
// and wait of it has ended.
type Step struct {
	Actor   string
	Move    []float64   // walk to [u, v], along u first, then v
	Via     [][]float64 // corners [u, v] the walk to Move passes first, in order, each reached along u first
	Enter   []float64   // appear at [u, v] first (an actor that starts off the stage), then walk to Move
	Exit    bool        // leave the stage when the walk ends
	Back    bool        // the walk keeps the way the actor faces: giving ground to a blow, backing away
	Strike  string      // the actor strikes this cast member: its motion, then the target's React, sparks, a flash and a shake as the blow lands
	Motion  string      // the striker's motion for a Strike ("attack" when empty or not on the sheet)
	React   string      // the target's motion as a Strike lands ("block" when empty): block, hit or a sheet's own
	Push    float64     // cells a Strike drives its target back along the blow, still bracing
	Crit    bool        // a Strike is a critical blow: a burst of sparks and a double flash as it lands
	Gather  float64     // seconds a Strike's striker gathers itself, whitening, before the motion
	Through float64     // a Strike dealt dashing through: the striker comes out this many cells beyond the target as it lands
	Speed   float64     // walking speed, 1 when 0 (2 runs, .6 drags)
	After   float64     // seconds after the group starts
	Face    string
	Look    string // turn this way for a moment, then back
	Play    string
	Emote   string
	Say     string // a speech bubble on the map, without the dialogue scroll
	Wait    float64
	Intro   string // an officer's first appearance
	Effect  string // petals, fire, smallfire, smoke, steam, birds, glint, flash, halberd, halberd_throw, none
	At      []float64
	Shake   float64 // seconds of screen shake
	Cam     string  // look at this actor
	CamAt   []float64
	Zoom    float64 // camera zoom, whole numbers only (pixel art)
	Time    string  // day (""), dusk, night or dawn from now on
	Sprite  string  // the actor stands as this sprite from now on
	Prop    []any   // [name, u, v] puts a map prop down, [name] takes it away, [name, actor] makes it move with that actor; "name#2" is one more of that prop
	Pose    string  // the motion the actor rests in from now on (sit, lie, kneel …); "idle" stands again
	Ease    string  // with Move: one stroke straight to the end at Speed on average, "line" even, "in" checked then gathering speed (a rider closing for the next blow), "out" fast then slowing to a stop
	Dead    bool    // the actor is killed where it is: its colours drain to grey and stay so
	Patrol  []float64
	// walk to [u, v] and back, over and over, without holding up the beats (a sentry); a Move ends it
	Chatter   string    // short bubbles the actor keeps saying now and then, "|" between them; "-" stops
	Lag       float64   // with Chatter: seconds before its first bubble (staggers a murmur across a crowd); an empty entry between the "|" is a turn of silence
	If        []string  // Ambient only: the step runs when these story conditions hold (carts there only once won back)
	Shoot     *Shoot    // the actor looses an arrow at the stage's halberd and the camera chases it
	Arrow     []float64 // with Play "attack": the actor's arrow leaves as the bow is let go and lands on this [u, v] (volleys from a wall)
	Free      bool      // the walk goes on without holding up the beats: people talking as they ride
	Stop      bool      // the actor stops walking at the next whole cell
	Follow    string    // the camera keeps on this actor from now on ("-" stops)
	Place     []float64 // the actor is at [u, v] at once (in the dark of a fade: time has passed)
	IfWalking bool      // the step runs only while its actor is still on its way (a line for players who read fast)
	Bound     *bool     // false unties a bound prisoner (the rope comes off and he may gesture again), true binds one
}

// Shoot is an arrow loosed at the halberd standing on the stage (the scene's halberd effect), flown over the map with
// the camera scrolling after it.
type Shoot struct {
	Time float64 // seconds in the air (1 when 0)
	Peak float64 // how high the arc climbs above its ends, in canvas px (60 when 0)
	Say  string  // a bubble over the shooter while the bow is held drawn
}

// Actor is where a cast member starts. A cast key with "#" ("병사#2") is an extra: no name plate, no portrait;
// the part before "#" may still be a character.
type Actor struct {
	At      []float64
	Face    string
	Faction string
	Sprite  string
	Mount   string   // an equipped horse (equipment id) whose colours a mounted sprite takes, as in battle
	Off     bool     // starts off the stage, until a step Enters it
	Hold    bool     // never turns by itself toward whoever speaks
	Pose    string   // the motion it rests in (sit, lie, kneel …) instead of standing idle
	Stay    bool     // on an explore walk's own stage: stands where the walk left it, with no At or Face of its own
	If      []string // stands only when these story conditions hold (an officer who lived through an earlier battle)
	Bound   bool     // a prisoner with a rope round his arms: walks and turns, but never gestures, bows or plays a motion
}

// Scene is one stretch of story nodes staged on one map.
type Scene struct {
	ID       string `json:"-"`
	Map      string
	Time     string
	Backdrop string             // the far picture behind an outdoor map (assets/graphics/backdrops)
	Glow     string             // a far light washing over the screen from one corner: "fire_nw" (a city burning beyond the north-west)
	Music    string             // the theme while it plays (assets/music/themes, no extension): a light-hearted scene's own; empty goes by the map
	Grass    *content.GrassTint `json:",omitempty"` // the grass here when the map stands for another land; nil keeps the map's
	Snow     float64            `json:",omitempty"` // 0..1: snow lies on the scene; its trees and bushes are the snow pictures and the grass white unless Grass says otherwise
	Steps    float64            `json:",omitempty"` // the walkers' footsteps and hoof beats at this share of their volume; 0 keeps them whole
	Nodes    []string
	NoSkip   []string `json:",omitempty"` // nodes whose beats play out whole: a click waits for them (a duel whose blows and pushes land in turn)
	Mute     []string `json:",omitempty"` // nodes whose speakers get no talking bubble on the map: the dialogue scroll alone carries the line
	NoFocus  []string `json:",omitempty"` // nodes whose beats and lines do not draw the camera to their actor: it keeps the shot it had
	Ambient  []Step   // run once as the scene opens (fires, braziers, smoke, sentries, chatter)
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
		if err := b.Add(p); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// Add reads one more scene file into the book, for a trial chapter kept outside assets/scenes.
func (b *Book) Add(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f file
	if err := json.Unmarshal(raw, &f); err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	for id, s := range f.Scenes {
		if _, dup := b.Scenes[id]; dup {
			return fmt.Errorf("%s: scene %s is defined twice", filepath.Base(path), id)
		}
		s.ID = id
		b.Scenes[id] = s
		for _, n := range s.Nodes {
			if other := b.byNode[n]; other != nil {
				return fmt.Errorf("node %s is in scenes %s and %s", n, other.ID, id)
			}
			b.byNode[n] = s
		}
		for _, n := range slices.Concat(s.NoSkip, s.Mute, s.NoFocus) {
			if !slices.Contains(s.Nodes, n) {
				return fmt.Errorf("%s: scene %s names node %s that it does not stage", filepath.Base(path), id, n)
			}
		}
	}
	return nil
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
