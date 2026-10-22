package storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"reflect"
	"srpg/internal/content"
	"srpg/internal/core"
	"testing"
)

func TestSlotsChecksumAndAtomicFailure(t *testing.T) {
	d, err := content.Load("../testdata/mini")
	if err != nil {
		t.Fatal(err)
	}
	s := Store{t.TempDir()}
	a := core.New(d, 2).Snapshot()
	if err = s.Save("1", a, false); err != nil {
		t.Fatal(err)
	}
	b, err := s.Load("1")
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("roundtrip", err)
	}
	if err = s.Save("1", a, false); err == nil {
		t.Fatal("overwrite lacks intent")
	}
	a.Revision = 2
	if err = s.Save("1", a, true); err != nil {
		t.Fatal(err)
	}
	b, _ = s.Load("1")
	if b.Revision != 2 {
		t.Fatal("overwrite")
	}
	if err = s.Save("11", a, true); err == nil {
		t.Fatal("11th slot")
	}
	for i := 2; i <= 10; i++ {
		if err = s.Save(fmt.Sprint(i), a, true); err != nil {
			t.Fatal(err)
		}
	}
	if err = s.Save("10", a, true); err != nil {
		t.Fatal(err)
	}
	if err = s.Save("auto", a, true); err != nil {
		t.Fatal(err)
	}
	if len(s.Slots()) != 11 {
		t.Fatal("slot count")
	}
	p := filepath.Join(s.Dir, "slot-1.json")
	bytes, _ := os.ReadFile(p)
	if err = os.WriteFile(p, append(bytes, []byte("broken")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Load("1"); err == nil {
		t.Fatal("corrupt save")
	}
	if err = os.WriteFile(p, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	blocked := filepath.Join(s.Dir, "blocked")
	_ = os.WriteFile(blocked, []byte("file"), 0600)
	if err = (Store{Dir: blocked}).Save("1", a, true); err == nil {
		t.Fatal("expected write failure")
	}
	after, _ := os.ReadFile(p)
	if !reflect.DeepEqual(bytes, after) {
		t.Fatal("failed write damaged existing slot")
	}
}
func TestSettingsIndependent(t *testing.T) {
	s := Store{t.TempDir()}
	v := DefaultSettings()
	v.AI = "step"
	v.Color = true
	v.TextDelayMS = 5
	if err := s.SaveSettings(v); err != nil {
		t.Fatal(err)
	}
	read, err := s.LoadSettings()
	if err != nil || read != v {
		t.Fatal(read, err)
	}
	bad := v
	bad.AI = "unknown"
	if err = s.SaveSettings(bad); err == nil {
		t.Fatal("invalid setting saved")
	}
	read, _ = s.LoadSettings()
	if read != v {
		t.Fatal("invalid setting replaced current")
	}
}

func TestDamagePctSetting(t *testing.T) {
	v := DefaultSettings()
	assert.Equal(t, 80, v.DamageScale())
	for _, p := range ScalePcts {
		v.DamagePct = p
		assert.NoError(t, v.Validate())
		assert.Equal(t, p, v.DamageScale())
	}
	v.DamagePct = 40
	assert.Error(t, v.Validate())

	x := DefaultSettings()
	assert.Equal(t, 80, x.XPScale())
	x.XPPct = 90
	assert.NoError(t, x.Validate())
	x.XPPct = 50
	assert.Error(t, x.Validate())
}

func TestCounterAllIsOnUntilTurnedOff(t *testing.T) {
	s := Store{t.TempDir()}
	t.Run("older settings without the key", func(t *testing.T) {
		assert.NoError(t, os.WriteFile(filepath.Join(s.Dir, "settings.json"), []byte(`{"ai":"auto","detail":true,"damage_pct":70}`), 0644))
		v, err := s.LoadSettings()
		assert.NoError(t, err)
		assert.True(t, v.CounterAll)
	})
	t.Run("turned off stays off", func(t *testing.T) {
		v := DefaultSettings()
		v.CounterAll = false
		assert.NoError(t, s.SaveSettings(v))
		read, err := s.LoadSettings()
		assert.NoError(t, err)
		assert.False(t, read.CounterAll)
	})
}
