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

// traitHelp explains a trait and what learning it costs.
func (g *Game) traitHelp(id string) string {
	return g.traitDesc(id) + fmt.Sprintf("\n\n특성치 %d · 행동을 소모하지 않습니다.", g.S.Data.Traits[id].Cost)
}

// traitNotes say what each trait does, as core/rules.go applies it.
var traitNotes = map[string]string{
	"필살":    "물리 공격이 항상 치명타가 됩니다.",
	"독공":    "물리 공격으로 피해를 주면 대상을 3턴 동안 중독시킵니다.",
	"속공":    "병법 '신속'을 쓸 수 있습니다.",
	"연환":    "단일 대상 병법을 쓰면 대상 곁의 다른 적 하나에게도 같은 병법이 이어집니다.",
	"침착":    "자기 진영 차례가 시작될 때 중독·화상·약화·혼란·봉책·속박이 풀립니다.",
	"위압":    "물리 공격이 빗나가도 피해를 절반은 줍니다.",
	"국사무쌍":  "병법을 써도 행동이 끝나지 않아 한 턴에 여러 번 병법을 쓸 수 있습니다.",
	"부호":    "도구를 써도 행동이 끝나지 않습니다.",
	"분기술":   "병법 '분기'를 쓸 수 있습니다.",
	"반격술":   "병법 '반격'을 쓸 수 있습니다.",
	"원시술":   "병법 '원시'를 쓸 수 있습니다.",
	"호걸":    "물리 피해 ×1.1.",
	"용장":    "활이 아닌 무기로 자신보다 무력이 낮은 장수를 칠 때 물리 피해 ×1.2.",
	"투신":    "병력이 절반 이하일 때 물리 피해 ×1.3.",
	"강행":    "황무지·산지·숲의 이동 비용이 1 줄어듭니다.",
	"행군":    "황무지 이동 비용이 1이 됩니다.",
	"철벽":    "받는 물리 피해 ×0.85.",
	"불굴":    "병력이 30% 이하일 때 받는 피해 ×0.75.",
	"방패":    "받는 물리 공격의 명중률 -10%.",
	"원거리방어": "활 부대에게 받는 피해 ×0.8.",
	"책략방어":  "병법으로 받는 피해 ×0.8.",
	"논객":    "병법 '설파'·'이간'을 쓸 수 있습니다.",
	"계략":    "병법 '혼란'·'봉책'을 쓸 수 있습니다.",
	"군율":    "중독·혼란·봉책·속박·약화·화상에 걸리는 기간이 1턴 줄어듭니다.",
	"정화":    "병법 '해독'·'정신통일'·'정화'를 쓸 수 있습니다.",
	"의술":    "회복 병법의 회복량 ×1.25.",
	"인덕":    "자기 진영 차례가 시작될 때 곁의 아군 병력을 최대치의 3% 회복시킵니다.",
	"보급":    "도구의 회복량 ×1.25.",
	"양생":    "자기 진영 차례가 시작될 때 병력을 최대치의 3% 회복합니다.",
	"집중":    "차례가 시작될 때 병법치 회복 +2.",
	"격려":    "병법 '격려'·'분발'을 쓸 수 있습니다.",
	"맹공":    "병법 '맹공'을 쓸 수 있습니다.",
	"기병숙련":  "기병일 때 물리 피해 ×1.15.",
	"보병숙련":  "보병일 때 물리 피해 ×1.15.",
	"궁병숙련":  "궁병일 때 물리 피해 ×1.15.",
	"도적숙련":  "도적일 때 물리 피해 ×1.15.",
	"탈취":    "병법 '탈취'로 적의 군량을 빼앗을 수 있습니다.",
	"행운":    "명중률과 치명타율 +5%.",
	"군신":    "활이 아닌 무기로 자신보다 무력이 낮은 장수를 치면 반드시 명중하고 치명타가 됩니다.",
	"패왕":    "활이 아닌 무기의 물리 피해 ×1.25. 물리 명중률이 80% 아래로 내려가지 않습니다.",
	"천명":    "적 차례마다 처음 받는 피해가 절반이 됩니다.",
	"협동":    "곁에 아군이 둘 이상 있으면 물리 피해 ×1.15.",
	"지휘":    "병법 '대분발'·'대방진'을 쓸 수 있습니다.",
	"부관":    "승급한 병종은 장수 한 명을 부관으로 두어 그 특성을 함께 쓰고 경험치를 나눠 줍니다.",
	"백출":    "병법치 소모가 절반이 됩니다.",
	"심공":    "물리 공격으로 준 피해의 1/4만큼 병력을 회복합니다.",
}

// traitDesc explains a trait, with the skills it grants.
func (g *Game) traitDesc(id string) string {
	s := traitNotes[id]
	if s == "" {
		s = "효과 설명이 없습니다."
	}
	for _, name := range g.S.Data.Traits[id].Skills {
		s += "\n\n[" + name + "] " + strings.ReplaceAll(g.skillHelp(name, nil), "\n", " · ")
	}
	return s
}
