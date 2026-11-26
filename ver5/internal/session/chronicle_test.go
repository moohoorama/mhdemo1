package session

import (
	"os"
	"path/filepath"
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
	lines := s.Chronicle()
	require.Len(t, lines, 1)
	assert.Equal(t, []string{"intro"}, nodes(lines[0].Entries), "the chapter start is kept at the new game")
	assert.True(t, lines[0].Current)
	reachBattle(t, s)
	lines = s.Chronicle()
	require.Len(t, lines, 1)
	assert.Equal(t, []string{"intro", "prep1"}, nodes(lines[0].Entries))
	es := lines[0].Entries
	assert.True(t, es[1].At.After(es[0].At), "each point has the time it was reached")
	assert.Positive(t, es[0].Level, "the lord's level then")
	assert.Empty(t, s.Store.Slots(), "a chronicle point is not a slot")

	point, err := s.Store.ChronicleLoad(lines[0].ID, 2)
	require.NoError(t, err)
	assert.Equal(t, "prep1", point.Node)
	assert.Equal(t, 2, point.Steps)
	assert.Equal(t, lines[0].ID, point.Run)

	t.Run("manual and automatic saves neither touch it nor are touched", func(t *testing.T) {
		before := chronicleBytes(t, s.Store.Dir)
		request(t, s, Request{Op: "save", Slot: "1"})
		for i := 0; i < 12; i++ {
			require.NoError(t, s.Store.Autosave(s.stamp(s.Engine.Snapshot())))
		}
		assert.Equal(t, before, chronicleBytes(t, s.Store.Dir), "slots never write the chronicle")
		saved, err := s.Store.Load("1")
		require.NoError(t, err)
		assert.Equal(t, lines[0].ID, saved.Run, "a save says which line it belongs to")
		assert.Equal(t, 2, saved.Steps)
		require.NoError(t, os.Remove(filepath.Join(s.Store.Dir, "slot-1.json")))
		assert.Len(t, s.Chronicle()[0].Entries, 2, "deleting a save leaves the chronicle")
	})
	t.Run("a new session on the same folder lists the same points", func(t *testing.T) {
		again, err := New(s.Data, s.Store.Dir)
		require.NoError(t, err)
		got := again.Chronicle()
		require.Len(t, got, 1)
		assert.Equal(t, nodes(lines[0].Entries), nodes(got[0].Entries))
		assert.False(t, got[0].Current, "no game is being played yet")
	})
}

func TestChronicleGoingBackStartsAnotherLineAndKeepsTheFirst(t *testing.T) {
	s := chronicleSession(t)
	request(t, s, Request{Op: "new", Seed: 1})
	reachBattle(t, s)
	first := s.Chronicle()[0].ID
	before := chronicleBytes(t, s.Store.Dir)

	r := s.Handle(Request{Op: "chronicle_load", Line: first, Seq: 1})
	assert.Equal(t, "DiscardRequired", r.Error, "progress not saved is asked about")
	request(t, s, Request{Op: "chronicle_load", Line: first, Seq: 1, Discard: true})
	assert.Equal(t, "intro", s.Engine.Observe().Node)
	assert.Equal(t, before, chronicleBytes(t, s.Store.Dir), "loading changes nothing in the chronicle")
	assert.Len(t, s.Chronicle(), 1, "a loaded point is not a new line until the story goes on")

	t.Run("playing the same point another way makes a second line", func(t *testing.T) {
		reachBattle(t, s)
		lines := s.Chronicle()
		require.Len(t, lines, 2)
		assert.Equal(t, first, lines[0].ID)
		assert.Equal(t, []string{"intro", "prep1"}, nodes(lines[0].Entries))
		assert.Equal(t, []string{"intro", "prep1"}, nodes(lines[1].Entries))
		require.NotNil(t, lines[1].From)
		assert.Equal(t, 1, lines[1].From.Number)
		assert.Equal(t, "intro", lines[1].From.Entry.Node)
		assert.True(t, lines[1].Current)
		assert.False(t, lines[0].Current)
		assert.Equal(t, 2, lines[1].Entries[1].Seq)
		assert.NotEqual(t, lines[0].Entries[1].At, lines[1].Entries[1].At)
		for name, content := range before {
			assert.Equal(t, content, chronicleBytes(t, s.Store.Dir)[name], "the first line's file %s is unchanged", name)
		}
	})
	t.Run("going on from the latest point of a line keeps the line", func(t *testing.T) {
		request(t, s, Request{Op: "chronicle_load", Line: s.Chronicle()[1].ID, Seq: 2, Discard: true})
		assert.Equal(t, "prep1", s.Engine.Observe().Node)
		request(t, s, Request{Op: "command", Command: core.Command{Kind: "start"}})
		assert.Len(t, s.Chronicle(), 2)
	})
	t.Run("a manual save of the old line, loaded later, forks as well", func(t *testing.T) {
		request(t, s, Request{Op: "chronicle_load", Line: first, Seq: 1, Discard: true})
		request(t, s, Request{Op: "save", Slot: "3"})
		request(t, s, Request{Op: "chronicle_load", Line: first, Seq: 2, Discard: true})
		request(t, s, Request{Op: "load", Slot: "3", Discard: true})
		assert.Equal(t, "intro", s.Engine.Observe().Node)
		reachBattle(t, s)
		assert.Len(t, s.Chronicle(), 3)
		assert.Equal(t, []string{"intro", "prep1"}, nodes(s.Chronicle()[0].Entries), "the first line still has its two")
	})
}

func TestChronicleLegacySaveStartsALineWithNothingBefore(t *testing.T) {
	s := chronicleSession(t)
	request(t, s, Request{Op: "new", Seed: 1})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "next"}})
	legacy := s.Engine.Snapshot()
	require.Empty(t, legacy.Run)
	require.NoError(t, s.Store.Save("5", legacy, false))
	require.NoError(t, os.RemoveAll(filepath.Join(s.Store.Dir, "chronicle")))
	request(t, s, Request{Op: "title", Discard: true})

	request(t, s, Request{Op: "load", Slot: "5"})
	assert.Empty(t, s.Chronicle(), "loading puts nothing into the chronicle")
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "choose", Option: "go"}})
	lines := s.Chronicle()
	require.Len(t, lines, 1)
	assert.Equal(t, []string{"prep1"}, nodes(lines[0].Entries), "no earlier point is invented")
	assert.Nil(t, lines[0].From)
}

func TestChronicleSkipsADebugJump(t *testing.T) {
	s := chronicleSession(t)
	request(t, s, Request{Op: "jump", Node: "prep1", Seed: 1})
	assert.Empty(t, s.Chronicle())
	request(t, s, Request{Op: "save", Slot: "2"})
	saved, err := s.Store.Load("2")
	require.NoError(t, err)
	assert.Equal(t, noChronicle, saved.Run)
	request(t, s, Request{Op: "title", Discard: true})
	request(t, s, Request{Op: "load", Slot: "2"})
	request(t, s, Request{Op: "command", Command: core.Command{Kind: "start"}})
	assert.Empty(t, s.Chronicle(), "a jumped game stays out of the chronicle after a load too")
}

func TestChronicleDamageAndFailureLeaveTheGameAndSavesAlone(t *testing.T) {
	s := chronicleSession(t)
	request(t, s, Request{Op: "new", Seed: 1})
	reachBattle(t, s)
	line := s.Chronicle()[0]

	t.Run("a damaged point is shown unreadable and will not load", func(t *testing.T) {
		request(t, s, Request{Op: "save", Slot: "1"})
		before := snapshot(s)
		p := filepath.Join(s.Store.Dir, "chronicle", line.ID, "0001.json")
		good, _ := os.ReadFile(p)
		require.NoError(t, os.WriteFile(p, append(append([]byte(nil), good[:len(good)/2]...), []byte("garbage")...), 0600))
		es := s.Chronicle()[0].Entries
		require.Len(t, es, 2)
		assert.True(t, es[0].Bad)
		assert.False(t, es[1].Bad)
		r := s.Handle(Request{Op: "chronicle_load", Line: line.ID, Seq: 1, Discard: true})
		assert.False(t, r.OK)
		assert.Contains(t, r.Error, "CorruptSave")
		assert.Equal(t, before, snapshot(s), "a failed load leaves the game as it was")
		assert.True(t, s.Store.Exists("1"), "the manual save stays")
		require.NoError(t, os.WriteFile(p, good, 0600))
		assert.False(t, s.Chronicle()[0].Entries[0].Bad)
	})
	t.Run("a point whose checksum fails is damaged", func(t *testing.T) {
		p := filepath.Join(s.Store.Dir, "chronicle", line.ID, "0002.json")
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
		assert.True(t, s.Chronicle()[0].Entries[1].Bad)
		require.NoError(t, os.WriteFile(p, good, 0600))
	})
	t.Run("a point from before a data change asks first, as a save does", func(t *testing.T) {
		old := s.Engine.Snapshot()
		old.ContentHash = "before-an-edit"
		old.Run, old.Steps = line.ID, 3
		require.NoError(t, s.Store.ChronicleAdd(storage.ChronicleMeta{Line: line.ID, Seq: 3, Node: "prep2", At: time.Now(), Level: 1}, old))
		r := s.Handle(Request{Op: "chronicle_load", Line: line.ID, Seq: 3, Discard: true})
		assert.Equal(t, "ChangedContent", r.Error)
		assert.True(t, s.Handle(Request{Op: "chronicle_load", Line: line.ID, Seq: 3, Discard: true, Changed: true}).OK)
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

func TestChronicleStoreNeverOverwritesAPoint(t *testing.T) {
	st := storage.Store{Dir: t.TempDir()}
	id, err := st.ChronicleBegin("", 0, time.Now())
	require.NoError(t, err)
	meta := storage.ChronicleMeta{Line: id, Seq: 1, Node: "x"}
	require.NoError(t, st.ChronicleAdd(meta, core.State{Node: "a"}))
	assert.Error(t, st.ChronicleAdd(meta, core.State{Node: "b"}))
	got, err := st.ChronicleLoad(id, 1)
	require.NoError(t, err)
	assert.Equal(t, "a", got.Node)
	_, err = st.ChronicleLoad(id, 2)
	assert.Error(t, err)
	_, err = st.ChronicleLoad("../etc", 1)
	assert.Error(t, err)
	assert.Equal(t, 1, st.ChronicleLen(id))
	assert.Zero(t, st.ChronicleLen("ffffff"))
}
