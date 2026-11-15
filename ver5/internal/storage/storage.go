package storage

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"srpg/internal/core"
	"time"
)

type Store struct{ Dir string }

// SlotCount is how many slots each of the manual and the automatic saves has.
const SlotCount = 10

// ManualSlots are the player's slots "1".."10"; AutoSlots are "auto1".."auto10", written only by Autosave.
func ManualSlots() []string { return slotIDs("") }
func AutoSlots() []string   { return slotIDs("auto") }
func slotIDs(prefix string) []string {
	out := make([]string, SlotCount)
	for i := range out {
		out[i] = fmt.Sprint(prefix, i+1)
	}
	return out
}
func IsManualSlot(slot string) bool { return slices.Contains(ManualSlots(), slot) }
func IsAutoSlot(slot string) bool   { return slices.Contains(AutoSlots(), slot) }

type Settings struct {
	Replay      string `json:"replay"` // how an AI army's turn plays: "together", "spells" (a stratagem plays alone) or "one" action at a time
	Color       bool   `json:"color"`
	Detail      bool   `json:"detail"`
	TextDelayMS int    `json:"text_delay_ms"`           // the CLI's pause after each letter
	InstantText bool   `json:"instant_text,omitempty"`  // the GUI shows lines whole instead of letter by letter
	AutoText    bool   `json:"auto_text,omitempty"`     // the GUI turns to the next line on its own a while after one is out
	UIScale     int    `json:"ui_scale,omitempty"`      // percent; 0 is the platform default
	DamagePct   int    `json:"damage_pct,omitempty"`    // damage dealt in percent; 0 is DefaultScalePct
	XPPct       int    `json:"xp_pct,omitempty"`        // experience gained in percent; 0 is DefaultScalePct
	NoGuide     bool   `json:"no_guide,omitempty"`      // the walk does not mark where the story goes on
	NoGoalGuide bool   `json:"no_goal_guide,omitempty"` // the battle does not outline what it is won and lost by
	Sound       string `json:"sound,omitempty"`         // the GUI's sound: "" all, "sfx" effects only, "music" music only, "off"
	Speed       int    `json:"speed,omitempty"`         // the GUI's replay speed (×1, 2, 4, 8); 0 is ×1
	ExploreZoom int    `json:"explore_zoom,omitempty"`  // the zoom the player last chose on the walk and in story scenes; 0 is the area's own
	BattleZoom  int    `json:"battle_zoom,omitempty"`   // the zoom the player last chose on a battle map; 0 is the default
}

// ScalePcts are the percents the player may choose for the damage and experience scales.
var ScalePcts = []int{60, 70, 80, 90, 100}

// DefaultScalePct is the percent of both scales until the player chooses one.
const DefaultScalePct = 80

// DamageScale is the damage percent in force.
func (s Settings) DamageScale() int {
	if s.DamagePct == 0 {
		return DefaultScalePct
	}
	return s.DamagePct
}

// XPScale is the experience percent in force.
func (s Settings) XPScale() int {
	if s.XPPct == 0 {
		return DefaultScalePct
	}
	return s.XPPct
}

func DefaultSettings() Settings { return Settings{Replay: "spells", Detail: true} }
func (s Settings) Validate() error {
	if !slices.Contains([]string{"together", "spells", "one"}, s.Replay) || s.TextDelayMS < 0 || s.TextDelayMS > 1000 || s.UIScale != 0 && (s.UIScale < 80 || s.UIScale > 160) || s.DamagePct != 0 && !slices.Contains(ScalePcts, s.DamagePct) || s.XPPct != 0 && !slices.Contains(ScalePcts, s.XPPct) || !slices.Contains([]string{"", "sfx", "music", "off"}, s.Sound) || s.ExploreZoom < 0 || s.ExploreZoom > 4 || s.BattleZoom < 0 || s.BattleZoom > 4 {
		return fmt.Errorf("InvalidSettings")
	}
	return nil
}
func (s Store) path(slot string) (string, error) {
	if !IsManualSlot(slot) && !IsAutoSlot(slot) {
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
	for _, id := range append(ManualSlots(), AutoSlots()...) {
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

// SavedAt is when the slot was last written; the zero time when it is empty.
func (s Store) SavedAt(slot string) time.Time {
	p, err := s.path(slot)
	if err != nil {
		return time.Time{}
	}
	fi, err := os.Stat(p)
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

// Latest is the most recently written slot, manual or automatic; false when nothing is saved.
func (s Store) Latest() (string, bool) {
	var best string
	var at time.Time
	for _, slot := range s.Slots() {
		if t := s.SavedAt(slot); best == "" || t.After(at) {
			best, at = slot, t
		}
	}
	return best, best != ""
}

// Autosave stores the next snapshot and the rotation sequence in the same atomic write.
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
