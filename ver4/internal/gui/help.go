package gui

import (
	"fmt"
	"strings"

	"srpg/internal/content"
	"srpg/internal/core"
)

// skillHelp explains what a skill does, with the caster's own numbers where the rules
// scale with them (core/rules.go: useSkill, stats, reach, strike).
func (g *Game) skillHelp(name string, caster *core.UnitView) string {
	s := g.S.Data.Skills[name]
	mind := 0
	class := content.Class{}
	if caster != nil {
		mind = caster.Stats.Mind
		class = g.S.Data.Classes[caster.Class]
	}
	turns := fmt.Sprintf("%d턴", s.Duration)
	if s.Duration == 1 {
		turns = "이번 턴"
	}
	value := int(float64(mind) * s.Coeff)
	var what string
	switch s.Effect {
	case "speed":
		what = "이번 턴 이동력 +2. 쓴 뒤에도 이동할 수 있습니다."
	case "charge":
		what = "이번 턴 적 옆을 지나가도 멈추지 않습니다(포위 무시). 쓴 뒤에도 이동할 수 있습니다."
	case "counter":
		what = turns + " 동안 공격을 받으면 사거리 안의 공격자에게 반격합니다(피해 70%)."
	case "range":
		what = "이번 턴 사거리 +1."
	case "morale":
		what = fmt.Sprintf("%s 동안 사기 +%d. 사기가 높으면 치명타가 잘 나고 적의 치명타를 덜 받습니다.", turns, value)
	case "attack":
		what = fmt.Sprintf("%s 동안 공격 +%d.", turns, value)
	case "defense":
		what = fmt.Sprintf("%s 동안 방어 +%d.", turns, value)
	case "confusion":
		what = turns + " 동안 혼란: 이동·공격·병법을 쓸 수 없습니다."
	case "seal":
		what = turns + " 동안 봉책: 병법을 쓸 수 없습니다."
	case "sealroot":
		what = turns + " 동안 병법을 쓸 수 없고 이동도 할 수 없습니다."
	case "heal":
		what = fmt.Sprintf("병력을 약 %d 회복합니다(정신 ×%.0f).", value, s.Coeff)
	case "curepoison":
		what = "중독·화상을 치료합니다."
	case "curemental":
		what = "혼란·봉책·속박을 풉니다."
	case "cure":
		what = "모든 상태 이상을 치료합니다."
	}
	switch {
	case name == "탈취":
		what = "피해 ×0.7의 공격. 30% 확률로 적의 군량을 빼앗습니다."
	case s.Kind == "physical" && s.Shape == "all":
		what = "사거리 안의 모든 적을 무기로 공격합니다."
	case s.Kind == "physical":
		what = fmt.Sprintf("강한 일격. 피해 ×%.2f.", s.Coeff)
	case s.Mode == "화계":
		what = "불로 공격합니다. 피해는 정신으로 정해지고, 방어 대신 적의 정신이 피해를 줄입니다."
	case s.Mode == "수계":
		what = "물로 공격합니다. 피해는 정신으로 정해지고, 방어 대신 적의 정신이 피해를 줄입니다. 시전자나 대상이 물가에 있어야 합니다."
	}
	facts := []string{fmt.Sprintf("병법치 %d", s.Cost)}
	lo, hi := s.Min, s.Max
	if hi == 0 {
		hi = class.Range
	}
	switch {
	case s.Mode == "자기":
		facts = append(facts, "대상 자신")
	case lo == hi:
		facts = append(facts, fmt.Sprintf("사거리 %d", hi))
	default:
		facts = append(facts, fmt.Sprintf("사거리 %d–%d", lo, hi))
	}
	switch s.Shape {
	case "cross":
		facts = append(facts, "대상과 상하좌우 칸")
	case "all":
		facts = append(facts, "사거리 안 전체")
	}
	if s.Hit < 100 && s.Mode != "자기" {
		facts = append(facts, fmt.Sprintf("기본 명중 %d%%", s.Hit))
	}
	return what + "\n" + strings.Join(facts, " · ")
}

// itemHelp explains a consumable (core/rules.go: useItem, promote).
func (g *Game) itemHelp(id string) string {
	if class, ok := strings.CutPrefix(id, "승급:"); ok {
		return fmt.Sprintf("병종을 %s(으)로 승급합니다. 승급 조건을 만족해야 합니다.\n자신 또는 인접 아군", class)
	}
	n := g.S.Data.Items[id]
	if id == "소병법단" {
		return fmt.Sprintf("병법치를 %d 회복합니다.\n자신 또는 인접 아군", n)
	}
	return fmt.Sprintf("병력을 %d 회복합니다.\n자신 또는 인접 부대", n)
}

// traitHelp lists what learning a trait gives.
func (g *Game) traitHelp(id string) string {
	t := g.S.Data.Traits[id]
	s := fmt.Sprintf("특성치 %d를 써서 특성 '%s'을(를) 익힙니다. 행동을 소모하지 않습니다.", t.Cost, id)
	if len(t.Skills) > 0 {
		s += "\n얻는 병법: " + strings.Join(t.Skills, " · ")
	}
	return s
}
