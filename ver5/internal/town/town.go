// Package town reads a walkable town (assets/towns/*.json): its areas (the streets and the rooms behind the doors),
// each staged on a scene map, the people standing in them with what they say, the signs to read and the doors that
// lead from one area to another. It is GUI data, not rules content: the hero walks it outside the story.
package town

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Line is one line of a talk: [speaker, text]; an empty speaker is narration.
type Line [2]string

// Person stands in an area. The key in Area.People is the shown name; a key with "#" is a silent extra.
// Sprite and Portrait stand in for a character's own when the name is no character (a smith, a shopkeeper).
type Person struct {
	At       []float64
	Face     string
	Sprite   string
	Portrait string
	Faction  string
	Pose     string
	Hold     bool      // keeps facing its own way while others speak; turns only to whoever talks to it
	Patrol   []float64 // walks to and fro between At and here
	Chatter  string    // short bubbles now and then, "|" between them
	Talk     []Line
}

// Sign is something on the map to look at: walking up to it shows its lines.
type Sign struct {
	Name string
	At   []float64
	Talk []Line
}

// Door is a way into another area. The hero walks to At, then stands at the door of area To that leads back here,
// facing that door's Face. Prop is the anchor of the building the door belongs to: clicking its picture walks to
// the door, and its name is shown above it. A door without To is shut: walking up to it shows Talk.
type Door struct {
	Name string
	Prop []float64
	At   []float64
	Face string
	To   string
	Talk []Line
}

// Area is one map of the town.
type Area struct {
	Map    string
	Name   string
	Time   string
	Zoom   float64 // the camera's zoom on arrival, whole numbers; 2 when 0
	People map[string]Person
	Signs  []Sign
	Doors  []Door
}

// Town is the whole walk.
type Town struct {
	Name  string
	Hero  string // the walker: a character's id or shown name
	Start struct {
		Area string
		At   []float64
		Face string
	}
	Ground     []string              // prop name prefixes that lie flat: walked over, never in the way
	Footprints map[string][4]float64 // cells a prop covers round its anchor: [u0, v0, u1, v1]
	Areas      map[string]*Area
}

// Load reads one town file and checks that its doors lead somewhere.
func Load(path string) (*Town, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t := &Town{}
	if err := json.Unmarshal(raw, t); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if t.Areas[t.Start.Area] == nil || len(t.Start.At) != 2 {
		return nil, fmt.Errorf("%s: start area %q is missing", path, t.Start.Area)
	}
	for id, a := range t.Areas {
		for _, d := range a.Doors {
			if len(d.At) != 2 {
				return nil, fmt.Errorf("%s: area %s: door %q has no spot", path, id, d.Name)
			}
			if d.To == "" {
				continue
			}
			to := t.Areas[d.To]
			if to == nil {
				return nil, fmt.Errorf("%s: area %s: door %q leads to unknown area %q", path, id, d.Name, d.To)
			}
			if to.DoorTo(id) == nil {
				return nil, fmt.Errorf("%s: area %s has no door back to %s", path, d.To, id)
			}
		}
	}
	return t, nil
}

// DoorTo is the area's door that leads to area to.
func (a *Area) DoorTo(to string) *Door {
	for i := range a.Doors {
		if a.Doors[i].To == to {
			return &a.Doors[i]
		}
	}
	return nil
}

// Flat reports whether a prop lies on the ground (a floor, a paved way, a mat): nothing to walk round.
func (t *Town) Flat(prop string) bool {
	for _, p := range t.Ground {
		if strings.HasPrefix(prop, p) {
			return true
		}
	}
	return false
}
