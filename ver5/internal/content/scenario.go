package content

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func strictDecode(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// ScenarioRoot is the scenario folder of the bundle at dir: assets/scenario beside assets/data,
// or a scenario folder inside dir.
func ScenarioRoot(dir string) (string, error) {
	for _, p := range []string{filepath.Join(dir, "..", "scenario"), filepath.Join(dir, "scenario")} {
		if _, err := os.Stat(filepath.Join(p, "scenario.json")); err == nil {
			return filepath.Clean(p), nil
		}
	}
	return "", fmt.Errorf("no scenario folder for %s", dir)
}

// FileNumber is the part of a stage id that names its files: B01 is stage01.json and maps/map01.json.
func FileNumber(id string) string {
	if n := strings.TrimLeft(id, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_-"); n != "" {
		return n
	}
	return id
}

func StageFileName(id string) string { return "stage" + FileNumber(id) + ".json" }
func MapFileName(id string) string   { return "map" + FileNumber(id) + ".json" }

// ScenarioDir is the folder of the scenario the data was loaded with, under the scenario root.
func (d *Data) ScenarioDir(root string) string {
	return filepath.Join(root, "assets", "scenario", d.ScenarioID)
}

func (d *Data) loadScenario(dir, want string, h hash.Hash) error {
	root, err := ScenarioRoot(dir)
	if err != nil {
		return err
	}
	read := func(path string) ([]byte, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		h.Write([]byte("scenario/" + filepath.ToSlash(path[len(root):])))
		h.Write(b)
		return b, nil
	}
	b, err := read(filepath.Join(root, "scenario.json"))
	if err != nil {
		return err
	}
	var index struct{ Scenarios []ScenarioRef }
	if err = strictDecode(b, &index); err != nil || len(index.Scenarios) == 0 {
		return fmt.Errorf("scenario.json: %v", err)
	}
	ref := index.Scenarios[0]
	if want != "" {
		found := false
		for _, r := range index.Scenarios {
			if r.ID == want {
				ref, found = r, true
			}
		}
		if !found {
			return fmt.Errorf("no scenario %q", want)
		}
	}
	scenarioDir := filepath.Join(root, filepath.Dir(ref.Path))
	if b, err = read(filepath.Join(root, ref.Path)); err != nil {
		return err
	}
	var file ScenarioFile
	if err = strictDecode(b, &file); err != nil {
		return fmt.Errorf("%s: %w", ref.Path, err)
	}
	d.ScenarioID, d.Campaign = ref.ID, file.Campaign
	for _, c := range file.Chapters {
		if b, err = read(filepath.Join(scenarioDir, c.File)); err != nil {
			return err
		}
		var story struct{ Nodes []Node }
		if err = strictDecode(b, &story); err != nil {
			return fmt.Errorf("%s: %w", c.File, err)
		}
		for _, n := range story.Nodes {
			c.Nodes = append(c.Nodes, n.ID)
		}
		d.Chapters = append(d.Chapters, c)
		d.Nodes = append(d.Nodes, story.Nodes...)
	}
	stages, err := filepath.Glob(filepath.Join(scenarioDir, "stage*.json"))
	if err != nil {
		return err
	}
	sort.Strings(stages)
	for _, p := range stages {
		if b, err = read(p); err != nil {
			return err
		}
		var s Stage
		if err = strictDecode(b, &s); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(p), err)
		}
		if s.Map == "" {
			d.Stages = append(d.Stages, s)
			continue
		}
		mb, err := read(filepath.Join(scenarioDir, "maps", s.Map))
		if err != nil {
			return err
		}
		doc, err := DecodeMapFile(mb)
		if err != nil {
			return fmt.Errorf("%s: %w", s.Map, err)
		}
		doc.Stage, doc.StageFile, doc.MapFile = s, filepath.Base(p), s.Map
		if d.Maps == nil {
			d.Maps = map[string]*MapDocument{}
		}
		if _, exists := d.Maps[s.ID]; exists {
			return fmt.Errorf("duplicate map %s", s.ID)
		}
		d.Maps[s.ID] = doc
		d.Stages = append(d.Stages, doc.Compile())
	}
	return nil
}

// After is the node that follows id: its own Next, else the next node of its chapter, else the
// first node of the next chapter. Empty when the story ends.
func (d *Data) After(id string) string {
	if n := d.Node(id); n.Next != "" {
		return n.Next
	}
	for i, c := range d.Chapters {
		for j, v := range c.Nodes {
			if v != id {
				continue
			}
			if j+1 < len(c.Nodes) {
				return c.Nodes[j+1]
			}
			for _, next := range d.Chapters[i+1:] {
				if len(next.Nodes) > 0 {
					return next.Nodes[0]
				}
			}
			return ""
		}
	}
	return ""
}

// mapFile is a map document as stored in maps/: everything but the Stage, which has its own file.
type mapFile struct {
	Decorations  [][4]any
	Props        [][]any
	Version      int
	Play, Visual MapRect
	Cells        map[string]string
	Default      string
}

func DecodeMapFile(b []byte) (*MapDocument, error) {
	var f mapFile
	if err := strictDecode(b, &f); err != nil {
		return nil, err
	}
	doc := &MapDocument{Decorations: f.Decorations, Props: f.Props, Version: f.Version, Play: f.Play, Visual: f.Visual, Cells: f.Cells, Default: f.Default}
	if err := doc.ValidateShape(); err != nil {
		return nil, err
	}
	return doc, nil
}

// EncodeFiles is the stage file and the map file a document is saved as.
func (d *MapDocument) EncodeFiles() (stage, mapDoc []byte, err error) {
	s := cloneMap(d.Stage)
	s.Tiles, s.Width, s.Height = nil, 0, 0
	s.Map = d.MapFile
	if s.Map == "" {
		s.Map = MapFileName(s.ID)
	}
	if stage, err = json.MarshalIndent(s, "", " "); err != nil {
		return
	}
	mapDoc, err = json.MarshalIndent(mapFile{d.Decorations, d.Props, d.Version, d.Play, d.Visual, d.Cells, d.Default}, "", " ")
	return
}
