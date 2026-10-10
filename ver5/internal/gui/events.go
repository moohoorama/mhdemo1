package gui

import (
	"fmt"
	"strings"

	"srpg/internal/core"
)

// eventText is one log line; important events are also announced over the field.
func (g *Game) eventText(e core.Event) (line string, important bool) {
	a, t := g.name(e.Actor), g.name(e.Target)
	switch e.Kind {
	case "damage":
		if e.Actor == "" {
			return fmt.Sprintf("%s: 지속 피해 %d", t, e.Amount), false
		}
		return fmt.Sprintf("%s → %s: 피해 %d", a, t, e.Amount), false
	case "miss":
		return fmt.Sprintf("%s의 공격이 %s에게 빗나감", a, t), false
	case "critical":
		return fmt.Sprintf("%s의 치명타!", a), false
	case "double":
		return fmt.Sprintf("%s의 2회 공격!", a), false
	case "heal":
		return fmt.Sprintf("%s → %s: 병력 +%d", a, t, e.Amount), false
	case "effect":
		return fmt.Sprintf("%s → %s  효과 적용", a, t), false
	case "item":
		return fmt.Sprintf("%s → %s  도구 사용 +%d", a, t, e.Amount), false
	case "retreat":
		return a + " 퇴각", true
	case "death":
		return a + " 전사", true
	case "level":
		return fmt.Sprintf("%s 레벨 업! Lv%d", a, e.Amount), true
	case "experience":
		return fmt.Sprintf("%s: 경험치 +%d", a, e.Amount), false
	case "cost":
		return fmt.Sprintf("%s: 병법치 -%d", a, e.Amount), false
	case "recover-hp":
		return fmt.Sprintf("%s: 병력 +%d", a, e.Amount), false
	case "recover-mp":
		return fmt.Sprintf("%s: 병법치 +%d", a, e.Amount), false
	case "move", "place":
		if len(e.Path) > 0 {
			p := e.Path[len(e.Path)-1]
			return fmt.Sprintf("%s: (%d,%d) 이동", a, p.X, p.Y), false
		}
		return "", false
	case "learn":
		return fmt.Sprintf("%s 특성 습득: %s", a, g.traitName(e.Target)), true
	case "promotion":
		return fmt.Sprintf("%s 승급: %s", a, g.className(e.Target)), true
	case "duel":
		return fmt.Sprintf("일기토  %s VS %s", a, t), true
	case "defense-down":
		return fmt.Sprintf("적의 진형이 흐트러졌다! 적 전체 방어 −%d%%", e.Amount), true
	case "attack-down":
		return fmt.Sprintf("적이 전의를 잃었다! 적 전체 공격 −%d%%", e.Amount), true
	case "victory":
		return "승리!", true
	case "turn":
		return fmt.Sprintf("%d턴 · %s 진영", e.Amount, factionName(e.Target)), false
	case "join":
		return g.name(e.Target) + " 합류", true
	case "grant":
		return fmt.Sprintf("%s ×%d 획득", g.itemName(e.Target), e.Amount), true
	case "deputy":
		return fmt.Sprintf("%s 부관: %s", a, t), false
	case "attack-hit", "spell-hit":
		return "", false
	}
	if e.Text != "" {
		return e.Text, false
	}
	return fmt.Sprintf("%s %s %s %d", e.Kind, a, t, e.Amount), false
}

// itemName is the shown name of an item or piece of equipment.
func (g *Game) itemName(id string) string {
	if eq, ok := g.S.Data.Equipment[id]; ok {
		return eq.Name
	}
	if it, ok := g.S.Data.Items[id]; ok {
		return it.Name
	}
	return id
}

func (g *Game) className(id string) string {
	if c, ok := g.S.Data.Classes[id]; ok {
		return c.Name
	}
	return id
}

func (g *Game) traitName(id string) string {
	if t, ok := g.S.Data.Traits[id]; ok {
		return t.Name
	}
	return id
}

func (g *Game) skillName(id string) string {
	if s, ok := g.S.Data.Skills[id]; ok {
		return s.Name
	}
	return id
}

func factionName(f string) string {
	return map[string]string{"ally": "아군", "enemy": "적군"}[f]
}

// statusLabels are the engine's status ids as the screen names them.
var statusLabels = map[string]string{"speed": "신속", "range": "원시", "counter": "반격", "eightfold": "팔진",
	"morale": "사기", "attack": "공격", "defense": "방어", "agility": "순발", "poison": "중독", "burn": "화상",
	"confusion": "혼란", "seal": "봉책", "root": "속박", "disarm": "봉격", "weak": "약화", "attack-down": "공격↓", "defense-down": "방어↓"}

// percentStatus lists the statuses whose value is a signed percent.
var percentStatus = map[string]bool{"attack": true, "defense": true, "morale": true, "agility": true}

// statusText is a status with its size or remaining turns: stat changes show their percent,
// the battle-long "-down" ones their cut, the rest their turns.
func statusText(id string, s core.Status) string {
	switch {
	case percentStatus[id]:
		return fmt.Sprintf("%s %+d%% %d", statusLabels[id], s.Value, s.Turns)
	case strings.HasSuffix(id, "-down"):
		return fmt.Sprintf("%s %d%%", statusLabels[id], s.Value)
	}
	return fmt.Sprintf("%s %d", statusLabels[id], s.Turns)
}

var errorNames = map[string]string{
	"StaleRevision": "화면이 최신 상태가 아닙니다", "NotYourTurn": "지금은 행동할 수 없습니다",
	"OutOfRange": "범위 밖입니다", "InvalidTarget": "대상이 올바르지 않습니다", "ActionSpent": "이미 행동했습니다",
	"InsufficientMP": "병법치가 부족합니다", "MoveSpent": "이미 이동했습니다", "WrongPhase": "지금은 할 수 없습니다",
	"InvalidSlot": "슬롯이 올바르지 않습니다", "NoGame": "진행 중인 게임이 없습니다",
	"IncompatibleSave": "호환되지 않는 저장입니다", "CorruptSave": "저장 파일이 손상되었습니다",
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func koreanError(code string) string {
	if s, ok := errorNames[code]; ok {
		return s
	}
	return code
}

type statRow struct {
	name string
	get  func(core.Stats) int
}

var statRows = []statRow{
	{"병력", func(s core.Stats) int { return s.MaxHP }}, {"병법치", func(s core.Stats) int { return s.MaxMP }},
	{"공격", func(s core.Stats) int { return s.Attack }}, {"방어", func(s core.Stats) int { return s.Defense }},
	{"정신", func(s core.Stats) int { return s.Mind }}, {"순발", func(s core.Stats) int { return s.Agility }},
	{"사기", func(s core.Stats) int { return s.Morale }},
}
