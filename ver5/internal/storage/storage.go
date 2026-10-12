package storage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"srpg/internal/core"
)

type Store struct{ Dir string }

func ManualSlots() []string { return []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"} }
func AutoSlots() []string   { return []string{"auto1", "auto2", "auto3", "auto4"} }
func IsManualSlot(slot string) bool {
	for _, id := range ManualSlots() {
		if slot == id {
			return true
		}
	}
	return false
}

type Settings struct {
	AI          string `json:"ai"`
	Color       bool   `json:"color"`
	Detail      bool   `json:"detail"`
	TextDelayMS int    `json:"text_delay_ms"`
}

func DefaultSettings() Settings { return Settings{AI: "auto", Detail: true} }
func (s Settings) Validate() error {
	if s.AI != "auto" && s.AI != "step" || s.TextDelayMS < 0 || s.TextDelayMS > 1000 {
		return fmt.Errorf("InvalidSettings")
	}
	return nil
}
func (s Store) path(slot string) (string, error) {
	valid := slot == "auto" || IsManualSlot(slot)
	for _, id := range AutoSlots() {
		valid = valid || slot == id
	}
	if !valid {
		return "", fmt.Errorf("InvalidSlot")
	}
	return filepath.Join(s.Dir, "slot-"+slot+".json"), nil
}
func (s Store) Exists(slot string) bool {
	p, err := s.path(slot)
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}
func (s Store) Slots() []string {
	out := []string{}
	for _, id := range append(append(ManualSlots(), AutoSlots()...), "auto") {
		if s.Exists(id) {
			out = append(out, id)
		}
	}
	return out
}

type envelope struct {
	Sequence uint64          `json:"sequence,omitempty"`
	Format   int             `json:"format"`
	Checksum string          `json:"checksum"`
	State    json.RawMessage `json:"state"`
}

func (s Store) Save(slot string, state core.State, overwrite bool) error {
	return s.save(slot, state, overwrite, 0)
}
func (s Store) save(slot string, state core.State, overwrite bool, sequence uint64) error {
	p, err := s.path(slot)
	if err != nil {
		return err
	}
	if s.Exists(slot) && !overwrite {
		return fmt.Errorf("OverwriteRequired")
	}
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	v := envelope{Sequence: sequence, Format: 1, Checksum: fmt.Sprintf("%x", sha256.Sum256(b)), State: b}
	b, err = json.Marshal(v)
	if err != nil {
		return err
	}
	return atomic(p, b)
}
func (s Store) Load(slot string) (core.State, error) {
	p, err := s.path(slot)
	if err != nil {
		return core.State{}, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return core.State{}, err
	}
	var v envelope
	if err = json.Unmarshal(b, &v); err != nil {
		return core.State{}, fmt.Errorf("CorruptSave: %w", err)
	}
	if v.Format != 1 {
		return core.State{}, fmt.Errorf("IncompatibleSave")
	}
	if v.Checksum != fmt.Sprintf("%x", sha256.Sum256(v.State)) {
		return core.State{}, fmt.Errorf("CorruptSave: checksum")
	}
	var state core.State
	err = json.Unmarshal(v.State, &state)
	return state, err
}
func atomic(path string, b []byte) (err error) {
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".srpg-*")
	if err != nil {
		return
	}
	defer func() { _ = f.Close(); _ = os.Remove(f.Name()) }()
	if _, err = f.Write(b); err != nil {
		return
	}
	if err = f.Sync(); err != nil {
		return
	}
	if err = f.Close(); err != nil {
		return
	}
	return os.Rename(f.Name(), path)
}
func (s Store) SaveSettings(v Settings) error {
	if err := v.Validate(); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return atomic(filepath.Join(s.Dir, "settings.json"), b)
}
func (s Store) LoadSettings() (Settings, error) {
	v := DefaultSettings()
	b, err := os.ReadFile(filepath.Join(s.Dir, "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return v, nil
	}
	if err != nil {
		return v, err
	}
	if err = json.Unmarshal(b, &v); err != nil {
		return DefaultSettings(), err
	}
	return v, v.Validate()
}

// Autosave stores the next turn snapshot and the rotation sequence in the same atomic write.
func (s Store) Autosave(state core.State) error {
	var latest uint64
	slots := AutoSlots()
	for _, slot := range slots {
		p, _ := s.path(slot)
		b, err := os.ReadFile(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		var v envelope
		if json.Unmarshal(b, &v) == nil {
			latest = max(latest, v.Sequence)
		}
	}
	return s.save(slots[latest%uint64(len(slots))], state, true, latest+1)
}
