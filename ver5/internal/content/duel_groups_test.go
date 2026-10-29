package content

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDuelGroups(t *testing.T) {
	t.Run("separate pairs", func(t *testing.T) {
		s := Stage{Duels: []Duel{{Ally: "zhangfei", Enemy: "dengmao"}, {Ally: "guanyu", Enemy: "chengyuanzhi"}}}
		assert.Equal(t, []int{0, 1}, s.DuelGroups())
	})
	t.Run("one enemy met by several", func(t *testing.T) {
		s := Stage{Duels: []Duel{{Ally: "zhangfei", Enemy: "lubu"}, {Ally: "mushun", Enemy: "lubu"}, {Ally: "guanyu", Enemy: "huaxiong"}}}
		assert.Equal(t, []int{0, 0, 1}, s.DuelGroups())
	})
	t.Run("joined by a support", func(t *testing.T) {
		s := Stage{Duels: []Duel{{Ally: "a", Enemy: "x"}, {Ally: "b", Enemy: "y"}, {Ally: "c", Enemy: "z", Supports: []string{"a", "b"}}}}
		assert.Equal(t, []int{0, 0, 0}, s.DuelGroups())
	})
}
