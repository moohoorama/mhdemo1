package session

import (
	"os"
	"path/filepath"
	"sort"
	"srpg/internal/core"
	"srpg/internal/storage"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reachBattle plays the mini story's first lines up to its first battle: the chapter start (intro) then the battle (prep1).
func reachBattle(t *testing.T, s *Session) {
	t.Helper()
	for _, c := range []core.Command{{Kind: "next"}, {Kind: "choose", Option: "go"}} {
		request(t, s, Request{Op: "command", Command: c})
	}
	require.Equal(t, "prep1", s.Engine.Observe().Node)
}

func chronicleSession(t *testing.T) *Session {
	s := setup(t)
	clock := time.Date(2026, 11, 25, 9, 0, 0, 0, time.Local)
	s.Clock = func() time.Time { clock = clock.Add(time.Hour); return clock }
	return s
}

func nodes(es []storage.ChronicleEntry) (out []string) {
	for _, e := range es {
		out = append(out, e.Node)
	}
	sort.Strings(out)
	return
}

func chronicleBytes(t *testing.T, dir string) map[string]string {
	out := map[string]string{}
	require.NoError(t, filepath.Walk(filepath.Join(dir, "chronicle"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			b, _ := os.ReadFile(p)
			rel, _ := filepath.Rel(dir, p)
			out[rel] = string(b)
		}
		return err
	}))
	return out
}

func TestChronicleKeepsReachedPointsApartFromSlots(t *testing.T) {
	s := chronicleSession(t)
	assert.Empty(t, s.Chronicle(), "nothing before a game")
	request(t, s, Request{Op: "new", Seed: 1})
	assert.Equal(t, []string{"intro"}, nodes(s.Chronicle()), "the chapter start is kept at the new game")
	reachBattle(t, s)
	es := s.Chronicle()
	assert.Equal(t, []string{"intro", "prep1"}, nodes(es))
	assert.Positive(t, es[0].Level, "the lord's level then")
	assert.False(t, es[0].At.IsZero(), "each point has the time it was reached")
	assert.Empty(t, s.Store.Slots(), "a chronicle point is not a slot")

	point, err := s.Store.ChronicleLoad("prep1", "")
	require.NoError(t, err)
	assert.Equal(t, "prep1", point.Node)

	t.Run("manual and automatic saves neither touch it nor are touched", func(t *testing.T) {
		before := chronicleBytes(t, s.Store.Dir)
		request(t, s, Request{Op: "save", Slot: "1"})
		for i := 0; i < 12; i++ {
			require.NoError(t, s.Store.Autosave(s.Engine.Snapshot()))
		}
		assert.Equal(t, before, chronicleBytes(t, s.Store.Dir), "slots never write the chronicle")
		require.NoError(t, os.Remove(filepath.Join(s.Store.Dir, "slot-1.json")))
		assert.Len(t, s.Chronicle(), 2, "deleting a save leaves the chronicle")
	})
	t.Run("a new session on the same folder lists the same points", func(t *testing.T) {
		again, err := New(s.Data, s.Store.Dir)
		require.NoError(t, err)
		assert.Equal(t, nodes(es), nodes(again.Chronicle()))
	})
}

func TestChronicleReplayingAPointOverwritesIt(t *testing.T) {
	s := chronicleSession(t)
	request(t, s, Request{Op: "new", Seed: 1})
	reachBattle(t, s)
	first := map[string]time.Time{}
	for _, e := range s.Chronicle() {
		first[e.Node] = e.At
	}

	r := s.Handle(Request{Op: "chronicle_load", Node: "intro"})
	assert.Equal(t, "DiscardRequired", r.Error, "progress not saved is asked about")
	before := chronicleBytes(t, s.Store.Dir)
	request(t, s, Request{Op: "chronicle_load", Node: "intro", Discard: true})
	assert.Equal(t, "intro", s.Engine.Observe().Node)
	assert.Equal(t, before, chronicleBytes(t, s.Store.Dir), "loading changes nothing in the chronicle")

	reachBattle(t, s)
	got := map[string]time.Time{}
	for _, e := range s.Chronicle() {
		got[e.Node] = e.At
	}
	assert.Len(t, got, 2, "one point a place, however often it is reached")
	assert.Equal(t, first["intro"], got["intro"], "a point the replay did not reach again is as it was")
	assert.True(t, got["prep1"].After(first["prep1"]), "the point reached again is the newest play's")
}

func TestChronicleSkipsADebugJump(t *testing.T) {
	s := chronicleSession(t)
	request(t, s, Request{Op: "jump", Node: "prep1", Seed: 1})
	assert.Empty(t, s.Chronicle())
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "start"}})
	assert.Empty(t, s.Chronicle(), "a jumped game is not kept")
}

func TestChronicleDamageAndFailureLeaveTheGameAndSavesAlone(t *testing.T) {
	s := chronicleSession(t)
	request(t, s, Request{Op: "new", Seed: 1})
	reachBattle(t, s)
	byNode := func(node string) storage.ChronicleEntry {
		for _, e := range s.Chronicle() {
			if e.Node == node {
				return e
			}
		}
		return storage.ChronicleEntry{}
	}

	t.Run("a damaged point is shown unreadable and will not load", func(t *testing.T) {
		request(t, s, Request{Op: "save", Slot: "1"})
		before := snapshot(s)
		p := filepath.Join(s.Store.Dir, "chronicle", "intro.json")
		good, _ := os.ReadFile(p)
		require.NoError(t, os.WriteFile(p, append(append([]byte(nil), good[:len(good)/2]...), []byte("garbage")...), 0600))
		assert.True(t, byNode("intro").Bad)
		assert.False(t, byNode("prep1").Bad)
		r := s.Handle(Request{Op: "chronicle_load", Node: "intro", Discard: true})
		assert.False(t, r.OK)
		assert.Contains(t, r.Error, "CorruptSave")
		assert.Equal(t, before, snapshot(s), "a failed load leaves the game as it was")
		assert.True(t, s.Store.Exists("1"), "the manual save stays")
		require.NoError(t, os.WriteFile(p, good, 0600))
		assert.False(t, byNode("intro").Bad)
	})
	t.Run("a point whose checksum fails is damaged", func(t *testing.T) {
		p := filepath.Join(s.Store.Dir, "chronicle", "prep1.json")
		good, _ := os.ReadFile(p)
		bad := append([]byte(nil), good...)
		for i := len(bad) / 2; i < len(bad)/2+4; i++ {
			if bad[i] == '1' {
				bad[i] = '2'
			} else {
				bad[i] = '1'
			}
		}
		require.NoError(t, os.WriteFile(p, bad, 0600))
		assert.True(t, byNode("prep1").Bad)
		require.NoError(t, os.WriteFile(p, good, 0600))
	})
	t.Run("a point from before a data change asks first, as a save does", func(t *testing.T) {
		old := s.Engine.Snapshot()
		old.ContentHash = "before-an-edit"
		require.NoError(t, s.Store.ChroniclePut(storage.ChronicleMeta{Node: "prep2", At: time.Now(), Level: 1}, old))
		r := s.Handle(Request{Op: "chronicle_load", Node: "prep2", Discard: true})
		assert.Equal(t, "ChangedContent", r.Error)
		assert.True(t, s.Handle(Request{Op: "chronicle_load", Node: "prep2", Discard: true, Changed: true}).OK)
	})
	t.Run("an unwritable chronicle does not stop the game", func(t *testing.T) {
		d := t.TempDir()
		s2, err := New(s.Data, d)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(d, "chronicle"), []byte("a file where the folder should be"), 0600))
		r := s2.Handle(Request{Op: "new", Seed: 1})
		assert.True(t, r.OK)
		failed := false
		for _, e := range r.Events {
			failed = failed || e.Kind == "chronicle-failed"
		}
		assert.True(t, failed, "the failure is told")
		request(t, s2, Request{Op: "command", Command: core.Command{Kind: "next"}})
		request(t, s2, Request{Op: "save", Slot: "1"})
		assert.True(t, s2.Store.Exists("1"), "manual saves are not affected")
	})
}

func TestChronicleStoreKeepsOnePointPerPlaceAndArm(t *testing.T) {
	st := storage.Store{Dir: t.TempDir()}
	require.NoError(t, st.ChroniclePut(storage.ChronicleMeta{Node: "x", Route: "a"}, core.State{Node: "1"}))
	require.NoError(t, st.ChroniclePut(storage.ChronicleMeta{Node: "x", Route: "b"}, core.State{Node: "2"}))
	require.NoError(t, st.ChroniclePut(storage.ChronicleMeta{Node: "x", Route: "a"}, core.State{Node: "3"}))
	got, err := st.ChronicleLoad("x", "a")
	require.NoError(t, err)
	assert.Equal(t, "3", got.Node, "a place reached again is written again")
	got, err = st.ChronicleLoad("x", "b")
	require.NoError(t, err)
	assert.Equal(t, "2", got.Node, "the other arm keeps its own")
	assert.Len(t, st.ChronicleEntries(), 2)
	_, err = st.ChronicleLoad("x", "")
	assert.Error(t, err)
	_, err = st.ChronicleLoad("../etc", "")
	assert.Error(t, err)
	assert.Error(t, st.ChroniclePut(storage.ChronicleMeta{Node: "a/b"}, core.State{}))
}
