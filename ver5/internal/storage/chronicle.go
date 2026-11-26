package storage

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"srpg/internal/core"
	"strconv"
	"strings"
	"time"
)

// The chronicle (지난 이야기) keeps the state the player really had on reaching a point of the story, apart from the
// manual and automatic slots, so the point can be played again with everything it had. A line is one run of the
// story: its own points are files chronicle/<line>/NNNN.json (NNNN is the point's Seq, counted along the line),
// and a line that was started from an earlier point of another line (a point loaded and played on in another
// way) names that line and the Seq it left it at: it shares that line's points up to there and adds its own.
// A file is written once and never changed, and no line is ever deleted or rewritten by the game.

const (
	chronicleDir    = "chronicle"
	chronicleFormat = 1
	maxLineage      = 64 // a line's chain of parents; longer is taken as a loop
)

// ChronicleLine is the file run.json of a line.
type ChronicleLine struct {
	ID        string
	Created   time.Time
	Parent    string `json:",omitempty"` // the line this one was started from
	ParentSeq int    `json:",omitempty"` // the last point of the parent it shares
}

// ChronicleMeta says what a point is, apart from its state.
type ChronicleMeta struct {
	Line  string    // the line that wrote the point
	Seq   int       // its place along the line, from 1
	Node  string    // the waypoint node it was reached at
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

func (s Store) chroniclePath(parts ...string) string {
	return filepath.Join(append([]string{s.Dir, chronicleDir}, parts...)...)
}

func validLineID(id string) bool {
	if len(id) == 0 || len(id) > 32 {
		return false
	}
	for _, r := range id {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func chronicleFileName(seq int) string { return fmt.Sprintf("%04d.json", seq) }

// ChronicleBegin makes a new line, started from parent's point parentSeq or from nothing when parent is empty.
func (s Store) ChronicleBegin(parent string, parentSeq int, at time.Time) (string, error) {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	line := ChronicleLine{ID: hex.EncodeToString(raw[:]), Created: at, Parent: parent, ParentSeq: parentSeq}
	if parent != "" && (!validLineID(parent) || parentSeq < 1) {
		return "", fmt.Errorf("InvalidChronicle")
	}
	b, err := json.Marshal(line)
	if err != nil {
		return "", err
	}
	if err := atomic(s.chroniclePath(line.ID, "run.json"), b); err != nil {
		return "", err
	}
	return line.ID, nil
}

// ChronicleAdd writes the point that follows the line's last one: meta.Seq must be that next place. A point is
// never overwritten.
func (s Store) ChronicleAdd(meta ChronicleMeta, state core.State) error {
	if !validLineID(meta.Line) || meta.Seq < 1 {
		return fmt.Errorf("InvalidChronicle")
	}
	p := s.chroniclePath(meta.Line, chronicleFileName(meta.Seq))
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("OverwriteRequired")
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

func (s Store) readLine(id string) (ChronicleLine, error) {
	var line ChronicleLine
	if !validLineID(id) {
		return line, fmt.Errorf("InvalidChronicle")
	}
	b, err := os.ReadFile(s.chroniclePath(id, "run.json"))
	if err != nil {
		return line, err
	}
	if err = json.Unmarshal(b, &line); err != nil || line.ID != id {
		return line, fmt.Errorf("CorruptChronicle")
	}
	return line, nil
}

// ownSeqs lists the Seqs of the points a line wrote itself, ascending.
func (s Store) ownSeqs(id string) []int {
	files, _ := os.ReadDir(s.chroniclePath(id))
	var out []int
	for _, f := range files {
		name, ok := strings.CutSuffix(f.Name(), ".json")
		if n, err := strconv.Atoi(name); ok && err == nil && n > 0 && f.Name() == chronicleFileName(n) {
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

// lineage is the points a line shows: its parent's up to where it was started from, then its own. A point whose
// file is damaged is returned with Bad set, its Seq taken from its name.
func (s Store) lineage(id string, depth int) ([]ChronicleEntry, error) {
	if depth > maxLineage {
		return nil, fmt.Errorf("CorruptChronicle")
	}
	line, err := s.readLine(id)
	if err != nil {
		return nil, err
	}
	var out []ChronicleEntry
	if line.Parent != "" {
		up, err := s.lineage(line.Parent, depth+1)
		if err != nil {
			return nil, err
		}
		for _, e := range up {
			if e.Seq <= line.ParentSeq {
				out = append(out, e)
			}
		}
	}
	for _, n := range s.ownSeqs(id) {
		meta, _, err := s.readPoint(id, n)
		if err != nil {
			out = append(out, ChronicleEntry{ChronicleMeta: ChronicleMeta{Line: id, Seq: n}, Bad: true})
			continue
		}
		out = append(out, ChronicleEntry{ChronicleMeta: meta})
	}
	return out, nil
}

// readPoint reads and verifies one point file of the line that wrote it.
func (s Store) readPoint(id string, seq int) (ChronicleMeta, json.RawMessage, error) {
	b, err := os.ReadFile(s.chroniclePath(id, chronicleFileName(seq)))
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
	if f.Checksum != fmt.Sprintf("%x", sha256.Sum256(f.State)) || f.Meta.Line != id || f.Meta.Seq != seq {
		return ChronicleMeta{}, nil, fmt.Errorf("CorruptSave: checksum")
	}
	return f.Meta, f.State, nil
}

// ChronicleEntries lists the points a line shows, in the order they were reached.
func (s Store) ChronicleEntries(line string) ([]ChronicleEntry, error) { return s.lineage(line, 0) }

// ChronicleLen is how many points a line has reached, its parent's shared ones included; the Seq of its last.
// A line the store does not know has none.
func (s Store) ChronicleLen(line string) int {
	es, err := s.lineage(line, 0)
	if err != nil || len(es) == 0 {
		return 0
	}
	return es[len(es)-1].Seq
}

// ChronicleLoad reads the state of the point seq in the line's list.
func (s Store) ChronicleLoad(line string, seq int) (core.State, error) {
	es, err := s.lineage(line, 0)
	if err != nil {
		return core.State{}, err
	}
	i := slices.IndexFunc(es, func(e ChronicleEntry) bool { return e.Seq == seq })
	if i < 0 {
		return core.State{}, fmt.Errorf("UnknownChronicle")
	}
	_, raw, err := s.readPoint(es[i].Line, seq)
	if err != nil {
		return core.State{}, err
	}
	var state core.State
	if err = json.Unmarshal(raw, &state); err != nil {
		return core.State{}, fmt.Errorf("CorruptSave: %w", err)
	}
	return state, nil
}

// ChronicleLines lists the lines that have a point of their own, oldest first; a line the store cannot read is
// left out.
func (s Store) ChronicleLines() []ChronicleLine {
	dirs, err := os.ReadDir(s.chroniclePath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil
	}
	var out []ChronicleLine
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		line, err := s.readLine(d.Name())
		if err != nil || len(s.ownSeqs(line.ID)) == 0 {
			continue
		}
		out = append(out, line)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Created.Equal(out[j].Created) {
			return out[i].Created.Before(out[j].Created)
		}
		return out[i].ID < out[j].ID
	})
	return out
}
