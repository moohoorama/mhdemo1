package scenes

import (
	"encoding/json"
	"fmt"
	"os"
)

// Reel is the title screen's backdrop: shots of the story played one after another, without a word of dialogue,
// under the menu (assets/scenes/title/reel.json).
type Reel struct {
	Shots []*Shot
}

// Shot is one stretch of the reel: a scene's stage (map, time of day, cast, ambient effects) and the beats it
// plays, group after group, before it lingers for Hold seconds and fades to the next shot.
type Shot struct {
	ID string
	Scene
	Groups [][]Step
	Hold   float64
	Zoom   float64 // camera zoom, whole numbers (1 when 0)
	Radius float64 // the remembered circle's radius as a share of the screen height (.36 when 0)
	// End is when the shot starts darkening to the next, in seconds from the start of its last group, though its
	// people are still at it; 0 waits for its beats to be done and lingers Hold seconds
	End float64 `json:",omitempty"`
	// Lead is how many seconds the shot plays in the dark before it opens: it opens on people already on the run
	Lead float64 `json:",omitempty"`
}

// LoadReel reads the title reel.
func LoadReel(path string) (*Reel, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Reel
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(r.Shots) == 0 {
		return nil, fmt.Errorf("%s: no shots", path)
	}
	return &r, nil
}

// Node is the story node a shot is staged as: the scene player plays a node's lines, and a shot is one line.
func (s *Shot) Node() string { return "reel:" + s.ID }

// Def is the shot staged as a scene of one node whose one line plays the shot's groups.
func (s *Shot) Def() *Scene {
	def := s.Scene
	def.ID = s.Node()
	def.Nodes = []string{s.Node()}
	def.Beats = map[string][][][]Step{s.Node(): {s.Groups}}
	return &def
}
