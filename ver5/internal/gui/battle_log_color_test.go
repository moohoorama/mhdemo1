package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogSpans(t *testing.T) {
	g := &Game{names: map[string]string{"장비": "zhangfei", "유비": "liubei"}}
	join := func(spans []logSpan) (text string, kinds []string) {
		for _, s := range spans {
			text += s.text
			switch s.c {
			case logHurt:
				kinds = append(kinds, "hurt:"+s.text)
			case logGain:
				kinds = append(kinds, "gain:"+s.text)
			case logPerson:
				kinds = append(kinds, "person:"+s.text)
			}
		}
		return
	}
	t.Run("damage", func(t *testing.T) {
		text, kinds := join(g.logSpans("장비 → 황건 도적2: 피해 402"))
		assert.Equal(t, "장비 → 황건 도적2: 피해 402", text)
		assert.Equal(t, []string{"person:장비", "hurt:피해 402"}, kinds)
	})
	t.Run("heal", func(t *testing.T) {
		text, kinds := join(g.logSpans("유비 → 유비: 병력 +220"))
		assert.Equal(t, "유비 → 유비: 병력 +220", text)
		assert.Equal(t, []string{"person:유비", "person:유비", "gain:병력 +220"}, kinds)
	})
	t.Run("plain", func(t *testing.T) {
		text, kinds := join(g.logSpans("7턴 · 적군 진영"))
		assert.Equal(t, "7턴 · 적군 진영", text)
		assert.Empty(t, kinds)
	})
}
