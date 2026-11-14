package ai

import (
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"srpg/internal/content"
	"srpg/internal/core"
)

// TestCampaignStatesRestore plays the whole story with the AI on both sides, the party undying so it reaches
// the end (a battle lost five times ends the run there), choosing the first option of every choice and promoting
// whoever a seal fits. Under test every Apply restores the state it leaves (core's Apply), so a command whose
// state a save or the move preview could not load again fails here with what broke. It takes minutes: set
// SRPG_CAMPAIGN=1 to run it.
func TestCampaignStatesRestore(t *testing.T) {
	if os.Getenv("SRPG_CAMPAIGN") == "" {
		t.Skip("set SRPG_CAMPAIGN=1 to play the campaign")
	}
	d, err := content.Load("../../assets/data")
	require.NoError(t, err)
	for _, difficulty := range []string{"normal", "hard"} {
		t.Run(difficulty, func(t *testing.T) {
			e, err := core.NewGame(d, 1, difficulty)
			require.NoError(t, err)
			e.SetImmortal("ally")
			apply := func(c core.Command) {
				t.Helper()
				_, err := e.Apply(c)
				require.NoError(t, err, "stage %s round %d: %+v", d.Stages[e.Observe().Stage].ID, e.Observe().Round, c)
			}
			battles := 0
			for step := 0; step < 20000; step++ {
				o := e.Observe()
				switch {
				case o.Phase == "complete":
					t.Logf("%d battles to the end", battles)
					return
				case o.Phase == "scenario" && (o.Dialogue.Kind == "say" || o.Dialogue.Kind == "narrate"):
					apply(core.Command{Kind: "next"})
				case e.Walking():
					for _, c := range e.ExploreNext() {
						apply(c)
					}
				case o.Phase == "scenario" && o.Dialogue.Kind == "choice":
					options := []string{}
					for k := range o.Dialogue.Choices {
						options = append(options, k)
					}
					sort.Strings(options)
					apply(core.Command{Kind: "choose", Option: options[0]})
				case o.Phase == "scenario" && o.Dialogue.Kind == "battle":
					promoteAll(t, d, e, apply)
					for c, ok := e.OwnerPiece(); ok; c, ok = e.OwnerPiece() {
						apply(c)
					}
					before := e.Snapshot()
					for try := 1; ; try++ {
						apply(core.Command{Kind: "start"})
						for i := 0; i < 20000 && e.Observe().Phase != "result"; i++ {
							c, err := Next(e)
							require.NoError(t, err)
							apply(c)
						}
						if e.Observe().Result == "victory" {
							break
						}
						if try == 5 {
							t.Logf("the AI lost %s five times: the states up to it are checked, the rest of the story is not", d.Stages[e.Observe().Stage].ID)
							return
						}
						before.RNG += 0x9e3779b97f4a7c15 // a retry plays out differently
						e, err = core.Restore(d, before)
						require.NoError(t, err)
						e.SetImmortal("ally")
					}
					battles++
					for c, ok := Learning(e); ok; c, ok = Learning(e) {
						apply(c)
					}
					apply(core.Command{Kind: "continue"})
				default:
					t.Fatalf("stuck in phase %s at node %s (%s)", o.Phase, o.Node, o.Dialogue.Kind)
				}
			}
			t.Fatal("the story did not end")
		})
	}
}

// promoteAll spends the party's seals on the officers they fit, highest level first, into the first class offered.
func promoteAll(t *testing.T, d *content.Data, e *core.Engine, apply func(core.Command)) {
	t.Helper()
	officers := append([]core.Character(nil), e.Observe().Characters...)
	sort.SliceStable(officers, func(i, j int) bool { return officers[i].Level > officers[j].Level })
	items := []string{}
	for id, it := range d.Items {
		if it.Effect == content.ItemPromote {
			items = append(items, id)
		}
	}
	sort.Strings(items)
	for _, item := range items {
		for _, c := range officers {
			if e.Observe().Inventory[item] < 1 || !d.Characters[c.Def].Playable {
				continue
			}
			if classes, err := e.PromotionsAway(item, c.ID); err == nil {
				apply(core.Command{Kind: "item", Target: c.ID, Item: item, Option: classes[0]})
			}
		}
	}
}
