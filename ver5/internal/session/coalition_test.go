package session

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
	"testing"
)

func TestSupportPhaseIsAlwaysAutomaticAndRejectsPlayerCommands(t *testing.T) {
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, n := range d.Nodes {
		if n.Kind == "preparation" && n.Stage == "B03" {
			d.Campaign.Start = n.ID
			break
		}
	}
	s, err := New(d, t.TempDir())
	require.NoError(t, err)
	s.Engine = core.New(d, 17)
	s.Screen = "playing"
	s.Settings.AI = "step"
	_, err = s.Engine.Apply(core.Command{Kind: "start"})
	require.NoError(t, err)
	plan, err := s.Engine.DeploymentPlan()
	require.NoError(t, err)
	for _, c := range plan {
		_, err = s.Engine.Apply(c)
		require.NoError(t, err)
	}
	snap := s.Engine.Snapshot()
	for i := range snap.Units {
		if snap.Units[i].Faction == "enemy" && snap.Units[i].ID != "lubu" {
			snap.Units[i].HP = 0
		}
	}
	s.Engine, err = core.Restore(d, snap)
	require.NoError(t, err)
	_, err = s.Engine.Apply(core.Command{Kind: "wait", Actor: "liubei"})
	require.NoError(t, err)
	_, err = s.Engine.Apply(core.Command{Kind: "end"})
	require.NoError(t, err)
	before := snapshot(s)
	rev := s.Engine.Observe().Revision
	r := s.Handle(Request{Op: "command", Revision: &rev, Command: core.Command{Kind: "wait", Actor: "fangyue"}})
	assert.False(t, r.OK)
	assert.Equal(t, "NotYourTurn", r.Error)
	assert.Equal(t, before, snapshot(s))
	events, err := s.AdvanceEnemies()
	require.NoError(t, err)
	assert.NotEmpty(t, events)
	assert.Equal(t, "enemy", s.Engine.Observe().Turn)
}
