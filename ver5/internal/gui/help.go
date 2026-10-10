package gui

import (
	"fmt"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
)

// skillHelp explains a skill: its description (generated from its numbers when the data has
// none) and its cost, reach, area and odds, with the caster's own numbers where known.
func (g *Game) skillHelp(id string, caster *core.UnitView) string {
	s := g.S.Data.Skills[id]
	what := s.Desc
	if what == "" {
		what = skillSummary(s)
	}
	facts := []string{fmt.Sprintf("병법치 %d", s.Cost)}
	switch {
	case s.Mode == content.ModeSelf:
		facts = append(facts, "대상 자신")
	case s.Max == 0:
		facts = append(facts, "기본 사거리")
	case s.Min == s.Max:
		facts = append(facts, fmt.Sprintf("사거리 %d", s.Max))
	default:
		facts = append(facts, fmt.Sprintf("사거리 %d–%d", max(1, s.Min), s.Max))
	}
	switch s.Shape {
	case content.ShapeCross:
		facts = append(facts, "대상과 상하좌우 칸")
	case content.ShapeSquare:
		facts = append(facts, "대상 둘레 3×3")
	case content.ShapeLine:
		facts = append(facts, fmt.Sprintf("앞으로 %d칸 일직선", s.Length))
	case content.ShapeAll:
		facts = append(facts, "사거리 안 전체")
	}
	if s.Hit < 100 && s.Mode != content.ModeSelf {
		facts = append(facts, fmt.Sprintf("기본 명중 %d%%", s.Hit))
	}
	if s.Duration > 0 {
		facts = append(facts, fmt.Sprintf("%d턴 지속", s.Duration))
	}
	return what + "\n" + strings.Join(facts, " · ")
}

// skillSummary describes a skill from its kind, effect and coefficient alone.
func skillSummary(s content.Skill) string {
	switch s.Kind {
	case content.KindPhysical:
		return fmt.Sprintf("무기 공격. 피해 ×%.2f.", s.Coeff)
	case content.KindMagic:
		return fmt.Sprintf("정신으로 피해를 주는 공격. 피해 ×%.2f.", s.Coeff)
	case content.KindHeal:
		return "아군을 회복합니다."
	case content.KindBuff:
		return "아군을 강화합니다."
	}
	return "적에게 불리한 효과를 겁니다."
}

// itemHelp explains a consumable.
func (g *Game) itemHelp(id string) string {
	it := g.S.Data.Items[id]
	return it.Desc + "\n자신 또는 인접 아군"
}

// traitDesc explains a trait.
func (g *Game) traitDesc(id string) string {
	if d := g.S.Data.Traits[id].Desc; d != "" {
		return d
	}
	return "효과 설명이 없습니다."
}
