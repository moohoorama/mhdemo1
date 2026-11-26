package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGuojiaDebateFollowsConfusion(t *testing.T) {
	d, err := Load("../../assets/data")
	require.NoError(t, err)
	assert.Contains(t, d.Traits["confusion_debate"].Requires, "stratagem")
	assert.NotContains(t, d.Characters["guojia"].Traits, "confusion_debate")
	for _, tc := range []struct {
		level             int
		confusion, debate bool
	}{
		{1, false, false}, {39, false, false}, {40, true, true}, {41, true, true},
	} {
		class := d.PromotedAt("guojia", d.Characters["guojia"].Class, tc.level)
		traits := d.PlannedTraits("guojia", class, tc.level, nil)
		assert.Equal(t, tc.confusion, containsTrait(traits, "stratagem"), "level %d", tc.level)
		assert.Equal(t, tc.debate, containsTrait(traits, "confusion_debate"), "level %d", tc.level)
	}
}

func containsTrait(traits []string, want string) bool {
	for _, trait := range traits {
		if trait == want {
			return true
		}
	}
	return false
}
