package mapedit

import (
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

func Load(path string) (*Document, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d Document
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	if err := d.ValidateShape(); err != nil {
		return nil, err
	}
	return &d, nil
}

// Save replaces the one document used by both the editor and the game.
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
	source.Stage.Tiles = nil
	source.Stage.Width, source.Stage.Height = 0, 0
	b, err := json.MarshalIndent(source, "", " ")
	if err != nil {
		return err
	}
	path := filepath.Join(root, "assets/data/maps", d.Stage.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".map-save-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0644); err == nil {
		_, err = f.Write(append(b, '\n'))
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
