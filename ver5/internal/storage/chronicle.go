package storage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"srpg/internal/core"
	"strings"
	"time"
)

// The chronicle (지난 이야기) keeps the state the player really had on reaching a point of the story, apart from the
// manual and automatic slots, so the point can be played again with everything it had. A point is the waypoint
// node and the arm of its fork the game was on (content.Fork; none off the forks): chronicle/<node>.json, or
// chronicle/<node>@<arm>.json on a fork. Reaching it again writes it again, so there is one game's point per place
// and the arms the player did not take keep what they had.

const (
	chronicleDir    = "chronicle"
	chronicleFormat = 2
)

// ChronicleMeta says what a point is, apart from its state.
type ChronicleMeta struct {
	Node  string    // the waypoint node it was reached at
	Route string    // the arm of its fork; empty off the forks
	At    time.Time // when the player reached it
	Level int       // the lord's level then
}

// ChronicleEntry is a point as a list shows it; Bad is a file that could not be read or fails its checksum.
type ChronicleEntry struct {
	ChronicleMeta
	Bad bool
}

type chronicleFile struct {
	Format   int
	Checksum string
	Meta     ChronicleMeta
	State    json.RawMessage
}

func validChronicleName(s string) bool {
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return s != ""
}

func (s Store) chroniclePath(node, route string) (string, error) {
	if !validChronicleName(node) || route != "" && !validChronicleName(route) {
		return "", fmt.Errorf("InvalidChronicle")
	}
	name := node
	if route != "" {
		name += "@" + route
	}
	return filepath.Join(s.Dir, chronicleDir, name+".json"), nil
}

// ChroniclePut keeps the state as the point of the node on the arm, replacing the point the place had.
func (s Store) ChroniclePut(meta ChronicleMeta, state core.State) error {
	p, err := s.chroniclePath(meta.Node, meta.Route)
	if err != nil {
		return err
	}
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	f, err := json.Marshal(chronicleFile{Format: chronicleFormat, Checksum: fmt.Sprintf("%x", sha256.Sum256(b)), Meta: meta, State: b})
	if err != nil {
		return err
	}
	return atomic(p, f)
}

// readPoint reads and verifies the point of a place.
func (s Store) readPoint(node, route string) (ChronicleMeta, json.RawMessage, error) {
	p, err := s.chroniclePath(node, route)
	if err != nil {
		return ChronicleMeta{}, nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return ChronicleMeta{}, nil, err
	}
	var f chronicleFile
	if err = json.Unmarshal(b, &f); err != nil {
		return ChronicleMeta{}, nil, fmt.Errorf("CorruptSave: %w", err)
	}
	if f.Format != chronicleFormat {
		return ChronicleMeta{}, nil, fmt.Errorf("IncompatibleSave")
	}
	if f.Checksum != fmt.Sprintf("%x", sha256.Sum256(f.State)) || f.Meta.Node != node || f.Meta.Route != route {
		return ChronicleMeta{}, nil, fmt.Errorf("CorruptSave: checksum")
	}
	return f.Meta, f.State, nil
}

// ChronicleLoad reads the state of the point of the node on the arm.
func (s Store) ChronicleLoad(node, route string) (core.State, error) {
	_, raw, err := s.readPoint(node, route)
	if errors.Is(err, os.ErrNotExist) {
		return core.State{}, fmt.Errorf("UnknownChronicle")
	}
	if err != nil {
		return core.State{}, err
	}
	var state core.State
	if err = json.Unmarshal(raw, &state); err != nil {
		return core.State{}, fmt.Errorf("CorruptSave: %w", err)
	}
	return state, nil
}

// ChronicleEntries lists every point kept, in no order. A file that cannot be read is listed Bad, its place
// taken from its name.
func (s Store) ChronicleEntries() []ChronicleEntry {
	files, _ := os.ReadDir(filepath.Join(s.Dir, chronicleDir))
	var out []ChronicleEntry
	for _, f := range files {
		name, ok := strings.CutSuffix(f.Name(), ".json")
		if !ok || f.IsDir() {
			continue
		}
		node, route, _ := strings.Cut(name, "@")
		meta, _, err := s.readPoint(node, route)
		if err != nil {
			out = append(out, ChronicleEntry{ChronicleMeta: ChronicleMeta{Node: node, Route: route}, Bad: true})
			continue
		}
		out = append(out, ChronicleEntry{ChronicleMeta: meta})
	}
	return out
}
