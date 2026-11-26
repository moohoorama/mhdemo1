package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWaypointActs checks the groups the waypoint list is shown in: 1~16장 are the 1막, 17~24장 the 2막 both
// tellings share, and the 2막 closes in two tellings that the walk at 24장 splits — 연의 25~27장 and 가상 25v~27v장.
func TestWaypointActs(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	want := func(c Chapter) string {
		switch n := c.Number(); {
		case c.ID != "" && c.ID[len(c.ID)-1] == 'v':
			return "2막-가상"
		case n <= 16:
			return "1막"
		case n <= 24:
			return "2막"
		}
		return "2막-연의"
	}

	t.Run("the acts come in story order", func(t *testing.T) {
		assert.Equal(t, []string{"1막", "2막", "2막-연의", "2막-가상"}, d.WaypointActs())
	})
	t.Run("every chapter and battle waypoint is in the act of its chapter", func(t *testing.T) {
		ways := d.Waypoints()
		chapterStarts, battles := 0, 0
		for _, w := range ways {
			c := d.Chapters[d.ChapterIndex(w.Node)]
			assert.Equal(t, want(c), w.Act, "%s (chapter %s)", w.Node, c.ID)
			if w.Stage == "" {
				chapterStarts++
			} else {
				battles++
			}
		}
		assert.Equal(t, len(d.Chapters), chapterStarts, "no chapter is left out of the list")
		assert.Equal(t, len(ways), chapterStarts+battles)
	})
	t.Run("the groups together are the whole list, each waypoint once", func(t *testing.T) {
		seen := map[string]string{}
		for _, w := range d.Waypoints() {
			assert.Empty(t, seen[w.Node], "%s is in two groups", w.Node)
			seen[w.Node] = w.Act
			assert.Contains(t, d.WaypointActs(), w.Act)
		}
		assert.Len(t, seen, len(d.Waypoints()))
	})
	t.Run("the borders", func(t *testing.T) {
		act := func(node string) string {
			for _, w := range d.Waypoints() {
				if w.Node == node {
					return w.Act
				}
			}
			return "(none)"
		}
		assert.Equal(t, "1막", act("c16-battle"), "하비 is the last battle of the 1막")
		assert.Equal(t, "2막", act("c17-walk"), "허도 opens the 2막")
		assert.Equal(t, "2막", act("c24-battle"), "박망파 is still shared")
		assert.Equal(t, "2막-연의", act("c25-battle"))
		assert.Equal(t, "2막-연의", act("c27-redcliff-battle"))
		assert.Equal(t, "2막-가상", act("c25v-battle"))
		assert.Equal(t, "2막-가상", act("c27v-battle"))
		assert.Equal(t, "2막-가상", act("c25v-start"), "a 가상 chapter's start is in its telling, not in the shared 2막")
	})
}
