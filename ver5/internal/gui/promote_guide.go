package gui

import (
	"slices"

	"srpg/internal/content"
)

// The promotion guide leads the eye in orange, as the gear guide does in green, to a seal an officer can take now:
// 관리 (or the sortie preparation's 소지품), the 소지품 view, the 소모품 filter, the seal's row, 승급하기, then in the
// picker the officers it can promote, 미리 보기 and the preview's 승급하기. It shows wherever a seal can be used (the walk and a
// battle's sortie preparation) for as long as one can, and the 길 안내 setting hides it.

// promoteGuide is the seal the guide leads to, lowest tier first; ok is false when none can be used now.
func (g *Game) promoteGuide() (item string, ok bool) {
	if g.S.Engine == nil || g.S.Settings.NoGuide {
		return "", false
	}
	o := g.S.View()
	if g.guidePlace(o) == "" {
		return "", false
	}
	var seals []string
	for id, it := range g.S.Data.Items {
		if it.Effect == content.ItemPromote {
			seals = append(seals, id)
		}
	}
	slices.SortFunc(seals, func(a, b string) int { return g.S.Data.Items[a].Tier - g.S.Data.Items[b].Tier })
	for _, id := range seals {
		if g.insuUsable(o, id) {
			return id, true
		}
	}
	return "", false
}
