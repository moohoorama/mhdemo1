package gui

import (
	"fmt"

	"srpg/internal/core"
)

// name is the shown name of a unit or officer (unit IDs are officer IDs).
func (g *Game) name(id string) string {
	if o, ok := g.S.Data.Officers[id]; ok && o.Name != "" {
		return o.Name
	}
	return id
}

// eventText is one log line; important events are also announced over the field.
func (g *Game) eventText(e core.Event) (line string, important bool) {
	a, t := g.name(e.Actor), g.name(e.Target)
	switch e.Kind {
	case "damage":
		return fmt.Sprintf("%s → %s  피해 %d", a, t, e.Amount), false
	case "miss":
		return fmt.Sprintf("%s의 공격이 %s에게 빗나감", a, t), false
	case "critical":
		return fmt.Sprintf("%s의 치명타!", a), false
	case "heal":
		return fmt.Sprintf("%s → %s  병력 +%d", a, t, e.Amount), false
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
		return fmt.Sprintf("%s 경험치 +%d", a, e.Amount), false
	case "learn":
		return fmt.Sprintf("%s 특성 습득: %s", a, e.Target), true
	case "promotion":
		return fmt.Sprintf("%s 승급: %s", a, e.Target), true
	case "duel":
		return fmt.Sprintf("일기토  %s VS %s", a, t), true
	case "victory":
		return "승리!", true
	case "turn":
		return fmt.Sprintf("%d턴 · %s 진영", e.Amount, factionName(e.Target)), false
	case "join":
		return e.Target + " 합류", true
	case "grant":
		return fmt.Sprintf("%s ×%d 획득", g.itemName(e.Target), e.Amount), true
	case "deputy":
		return fmt.Sprintf("%s 부관: %s", a, t), false
	case "attack-hit", "spell-hit", "move", "cost", "recover-hp", "recover-mp":
		return "", false
	}
	if e.Text != "" {
		return e.Text, false
	}
	return fmt.Sprintf("%s %s %s %d", e.Kind, a, t, e.Amount), false
}

func (g *Game) itemName(id string) string {
	if eq, ok := g.S.Data.Equipment[id]; ok {
		return eq.Name
	}
	return id
}

func factionName(f string) string {
	return map[string]string{"ally": "아군", "enemy": "적군"}[f]
}

var terrainNames = map[byte]string{'.': "초원", 'd': "황무지", 's': "산지", 'f': "숲", 'c': "성벽", 'i': "성내", 'v': "마을", '~': "강"}

var statusNames = map[string]string{"speed": "신속", "charge": "돌격", "counter": "반격", "range": "원시",
	"morale": "사기↑", "attack": "공격↑", "defense": "방어↑", "poison": "중독", "burn": "화상",
	"confusion": "혼란", "seal": "봉책", "root": "속박", "weak": "약화"}

var badStatus = map[string]bool{"poison": true, "burn": true, "confusion": true, "seal": true, "root": true, "weak": true}

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
