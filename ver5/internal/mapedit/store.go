package mapedit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"srpg/internal/content"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var saveMu sync.Mutex

// Load reads a stage file and the map document it points to.
func Load(stagePath string) (*Document, error) {
	b, err := os.ReadFile(stagePath)
	if err != nil {
		return nil, err
	}
	var st content.Stage
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&st); err != nil {
		return nil, err
	}
	mb, err := os.ReadFile(filepath.Join(filepath.Dir(stagePath), "maps", st.Map))
	if err != nil {
		return nil, err
	}
	d, err := content.DecodeMapFile(mb)
	if err != nil {
		return nil, err
	}
	d.Stage, d.StageFile, d.MapFile = st, filepath.Base(stagePath), st.Map
	return d, nil
}

// Save replaces the stage file and the map file used by both the editor and the game.
func Save(root string, d *Document, data *content.Data) error {
	saveMu.Lock()
	defer saveMu.Unlock()
	if !idPattern.MatchString(d.Stage.ID) {
		return fmt.Errorf("맵 ID는 영문/숫자/-/_만 사용할 수 있습니다")
	}
	if err := d.Validate(data); err != nil {
		return err
	}
	source := Clone(*d)
	if source.StageFile == "" {
		source.StageFile = content.StageFileName(d.Stage.ID)
	}
	if source.MapFile == "" {
		source.MapFile = content.MapFileName(d.Stage.ID)
	}
	stage, mapDoc, err := source.EncodeFiles()
	if err != nil {
		return err
	}
	dir := data.ScenarioDir(root)
	if err := writeFile(filepath.Join(dir, "maps", source.MapFile), mapDoc); err != nil {
		return err
	}
	return writeFile(filepath.Join(dir, source.StageFile), stage)
}

func writeFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".map-save-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	var err2 error
	if err2 = f.Chmod(0644); err2 == nil {
		_, err2 = f.Write(append(b, '\n'))
	}
	if err2 == nil {
		err2 = f.Sync()
	}
	closeErr := f.Close()
	if err2 != nil {
		return err2
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
