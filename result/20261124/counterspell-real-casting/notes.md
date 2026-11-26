# 반계의 실제 책략 처리와 보복 소멸 규칙 검토

검토만 했다. 코드·데이터·테스트는 바꾸지 않았고 실행도 하지 않았다. 경로는 `ver5/internal/core/` 기준이다.
요청 원문: process/counterspell-real-casting-20261124/counterspell-real-casting-20261124.md
이전 기록: result/20261124/retaliation-stack-duration/notes.md, result/20261124/fazheng-reprisal-counterspell/notes.md

## 0. 요약
- 반계는 `combat.go:974-975`에서 `strike`를 한 번 더 부르는 구조다. 치명 판정·원소 배율·원소 면역·명중 시 특성 디버프는 `strike` 공용이라 이미 작동한다.
- 빠진 것은 책략 효과의 나머지 절반이다: 치명 표식, 3x3 지형 변화, 스탯 디버프·둔화, 버프 해제, 연쇄, 사거리 검사, 전용 이벤트, 피해 집계.
- 원 책략의 명중 여부를 보지 않는다(974행). 빗나간 책략도 되돌아간다.
- 보복 소멸 규칙의 구현 지점은 두 곳이다. 자기 군 턴 끝은 `endFaction` 루프(`combat.go:1253-1263`), 직접 공격 후는 `useSkill`의 `physical`/`spell` 호출 뒤(`combat.go:718-722`). 직접 공격 판정은 `c.Kind`가 아니라 스킬 `Kind`로 해야 한다. `u.Attacked`는 기본 공격만 켠다.

## 1. 반계(spell_reflect) 현행 처리

### 1-1. 데이터
- `ver5/assets/data/traits.json:590-597` `spell_reflect`, Effects `{"reflect": 1}`. 같은 계열 `counter_scheme`(607~610)은 `reflect_status`로 상태이상 책략을 되돌린다.
- 상수: `content/vocab.go:72` `FxReflect = "reflect"`, 73행 `FxReflectStatus`.

### 1-2. 발동 경로
- `spell()`의 `apply` 클로저 `combat.go:950`. 공격 책략 분기(`s.Kind == content.KindMagic`, 951행)에서 처리하고 `return`(977행)한다. 책략 분기는 물리 반격 경로와 섞이지 않는다.
- 원 피격: `combat.go:952` `e.strike(u, q, s, blow{share: 1})`. 결과로 `dealt`(952~953행), 치명 표식(955~956행), 버프 해제(958행), 스탯 디버프(959~968행 부근), 둔화(969행 부근)가 이어진다.
- 반계: `combat.go:974-975`
  ```
  if !derived && e.has(q, content.FxReflect) && q.HP > 0 && u.HP > 0 && !confused(q) {
      e.strike(q, u, s, blow{share: 1, derived: true})
  }
  ```
  - 공격자가 q, 피격자가 u인 `strike`를 한 번 더 부른다. `derived: true`라서 되돌려진 책략은 다시 되돌아가지 않는다.
  - MP는 소모하지 않는다. 병법치 소모는 `useSkill`(`combat.go:690`)에만 있고 반계는 거치지 않는다. 요청(병법치 없이 재시전)과 맞다.
  - 지력·계수·원소는 q 기준이다. `damage(q, u, s)`가 `strike` 안에서 호출된다(`combat.go:300`).
  - `castingMP(q)`(`combat.go:124`)는 `e.caster`가 q가 아니므로 q의 현재 MP를 본다. 패긴(FxPatience) 배율에 영향이 있는지는 미확인.

### 1-3. 요청 조건과 현행 차이
| 항목 | 현행 | 근거 |
|---|---|---|
| 원 책략 명중 확인 | **없음.** 빗나가도, 화덕성군 0 피해로 `hit=true`가 돼도 반계는 실행된다 | `combat.go:305-306`(면역 조기 반환, hit=true), `321`(miss 후 반환), `974`(hit 검사 없음) |
| 반계 blow의 명중 굴림 | 있음. q→u의 `hitChance`로 따로 굴린다 | `combat.go:308` |
| 반계 치명 판정 | 있음. `critForced(q,u,s)`·`critChance(q,u,s)`로 q 특성 기준 | `combat.go:311-313`, `critChance` 220행 |
| 반계 데미지·원소 배율 | 있음. `damage(q,u,s)`의 `FxPower+원소` 배율 | `combat.go:57` |
| 원소 면역(u가 면역) | 있음. `strike` 안에서 `v`(=u)의 `FxImmune+원소`를 본다 | `combat.go:303` |
| 명중 시 특성 디버프(FxHitDebuff), 명중 상태이상(FxHitStatus), 치명 혼란 | 있음. `strike` 안이라 반계에도 적용된다 | `combat.go:343-360`(Harmful 루프 353행) |
| 치명 표식(markCell) | **없음.** 원 대상 q에만 찍힌다 | `combat.go:955-956`(apply 안의 원 대상 분기). 반계 호출은 이 분기를 타지 않는다 |
| 지형 변화(TerrainTurns 3x3) | **없음.** 원 대상 `v` 주변에만 찍힌다 | `combat.go:1070-1077` |
| 스탯 디버프(FxSpellStat), 둔화(FxSpellSlow) | **없음.** apply의 magic 분기 안에만 있다 | `combat.go:959-969` |
| 버프 해제(FxDispelBuffs) | **없음** | `combat.go:958` |
| 연쇄(FxChain) | **없음** | `combat.go:1079`(원 apply만) |
| 피해 기록 | 일반 `damage` 이벤트(반계 전용 종류 없음). `dealt`에 더하지 않아 `spell-hit` 집계에서 빠진다 | `combat.go:280`(hurt가 emit), `952-953`, `1083-1085` |
| 사거리·대상 유효성 | **없음.** `validateSkill`을 거치지 않는다 | `rules.go:445`, `useSkill` 690행 이후 경로 밖 |
| 원 책략의 효과 옵션(Effect 문자열, heal/cure 등) | magic 분기는 `return`으로 끝나므로 반계에서도 실행되지 않는다(magic 책략에는 해당 효과 분기가 원래 없음) | `combat.go:977` |

- "반계가 물리 반격처럼 보인다"는 현행 구조에서는 해당 없다. 반계는 `strike`를 직접 부르고 반격 경로(`counterBack`/`strikeBack`, `combat.go:491`)를 거치지 않는다.

### 1-4. 현행 문제 정리
1. 원 책략이 빗나가도 반계가 발동한다(974행에 hit 조건 없음). 의도인지 확인 필요(6절 Q1).
2. 면역 대상(화덕성군)이 0 피해를 받아도 반계가 발동한다.
3. 반계된 책략은 지형을 바꾸지 않고 치명 표식도 찍지 않는다.
4. 반계 피해는 `dealt`에 들어가지 않으며, 이벤트 종류도 일반 `damage`라 화면이 반계를 구분하지 못한다.
5. 사거리 검사가 없어 거리가 먼 대상에게도 되돌아간다(의도 확인 필요, Q5).
- 기존 메모(`result/20261124/counterspell-real-casting/notes.md` 이전 판본)는 "GUI 재생이 attack으로 고정되어 물리 모션을 쓴다"고 적었다. 이번 검토에서는 GUI 코드를 읽지 않았으므로 미검증이다(7절).

## 2. "진짜 시전"에 필요한 것

이미 있는 조각:
- 치명 판정·배율: `strike` → `critForced`(`combat.go:311`, 책략 분기 190~199행 부근) → `boost(..., crit)`(`combat.go:108`, 호출 323행 부근).
- 원소 배율·면역: `damage`의 `FxPower+원소`(`combat.go:57`), `strike`의 면역(`combat.go:303`).
- 표식 쓰기: `markCell`(`battle_events.go:320`) → `placeMark`(322행~). 원소별 표식 상태는 `R.CritMarks[element]`에 있고 없으면 무시한다.
- 지형 변화 쓰기: `combat.go:1070-1077`의 3x3 루프(`placeMark`). 기준 칸은 원 대상 `v`.
- 책략 효과 블록: `apply`의 magic 분기(`combat.go:951-972`)에 버프 해제·스탯 디버프·둔화가 있다.

필요한 작업(방향만, 코드 변경 없음):
1. 반계 호출을 원 책략과 같은 처리(표식·지형·디버프·해제·연쇄)로 묶는다. 방법은 (a) `apply`를 "공격자·피격자를 받는 함수"로 빼서 원·반계 양쪽에서 부르거나, (b) 반계 블록에 같은 처리를 복제하는 것이다. (a)가 중복이 적다. 재귀 방지는 기존 `derived`를 그대로 쓴다.
2. 치명 표식: 반계 치명일 때 `markCell(u.X, u.Y, s.Element)`(조건 `s.TerrainTurns == 0`은 955행과 같게).
3. 지형 변화: `s.TerrainTurns > 0`이면 기준 칸 주변 3x3에 `placeMark`. 기준 칸은 Q2.
4. 명중 조건: 원 책략이 명중하고 피해가 있을 때(`hit && n > 0`)만 되돌리거나, 면역 0 피해는 제외. Q1.
5. 사거리: Q5.
6. 집계: 반계 피해를 원 시전자 `dealt`에 넣을지, 전용 이벤트(예: `reflect`)로 구분할지. Q6.
7. 구현 검증 규칙상 `validate.go:98`은 TerrainTurns가 Square 형태 magic에만 허용한다. 반계도 같은 전제를 따른다(원 책략 기준이므로 추가 검사는 필요 없음).

## 3. 보복(응징) 소멸 규칙: 자기 턴 종료 시 전부 소멸, 직접 공격 후 전부 소멸

### 3-1. 현행 턴 종료 처리
- `endFaction()`(`combat.go:1240`).
  - 1243~1246행: 끝나는 군(`ending`, `endingArmy`)에 속한 유닛만 처리(`ActsIn`). 군별로 종료되므로 AI 군도 같은 경로를 탄다.
  - 1253~1263행: `keys(u.Status)` 순회. 1255행에서 `counter`와 `holdGround`는 건너뛴다(제외 목록). 1258행 `s.Turns--`, 1259~1261행 0 이하 삭제, 아니면 저장.
- 턴 시작 처리(`combat.go:1296-1310`): 1307행 `delete(u.Status, "counter")`, 1308행 `delete(u.Status, holdGround)`. 자기 턴 시작 때 지운다.
- `Attacked` 초기화: `combat.go:1353`(턴 시작 블록). 턴 종료 시점에는 이번 턴 값이 남아 있다.

### 3-2. 직접 공격 판정의 함정
- `useSkill`(`combat.go:690`)에서 `c.Kind == "attack"`일 때만 `u.Attacked = true`(714~715행).
- `c.Kind == "attack"`은 **기본 공격**이다. `skillDef`가 `c.Kind == "attack"`에 `Rules.BasicAttack`을 돌려준다(`rules.go:251-`).
- 무쌍·난사 등 특수 공격과 모든 책략은 `c.Kind == "skill"`이다(`api.go:58-62`의 `kinds` 목록).
- 따라서 `u.Attacked`로는 특수 물리 공격과 책략을 구분할 수 없다. 직접 공격 여부는 스킬 정의의 `s.Kind == KindPhysical || s.Kind == KindMagic`으로 판정해야 한다. `KindMixed`는 별도로 정한다(Q9).
- 후속 공격(`battle_events.go:264`, `Kind: "attack"`, `e.physical` 직접 호출)은 `useSkill`을 거치지 않는다(Q10).

### 3-3. 구현 지점 (두 곳)

(a) 자기 턴 종료 시 전부 소멸: `endFaction` 루프에 한 분기 추가
```
for _, k := range keys(u.Status) {
    if k == <보복 id> { delete(u.Status, k); continue }   // 자기 군 턴이 끝나면 공격 여부와 무관하게 전부 소멸
    s := u.Status[k]
    if k == "counter" || k == holdGround { continue }
    ...
}
```
- 위치: `combat.go:1253-1263`, 기존 제외 목록(1255행) 옆.
- 이 분기는 그 군의 자기 턴 종료(`ActsIn`, 1243행)에만 돈다. 적 턴에 맞아 쌓인 보복도 아군 턴 끝에 전부 지워진다. 요청과 맞다.
- 감소(`Turns--`)를 쓰지 않으므로 `Turns` 값은 상관없다. 이전 설계(매 턴 -1)와 다르다(Q11).

(b) 직접 공격 후 전부 소멸: `useSkill`에서 `physical`/`spell` 호출 뒤
```
if s.Kind == content.KindPhysical || s.Kind == content.KindMagic {   // 3-2의 판정
    delete(u.Status, <보복 id>)
}
```
- 위치: `combat.go:718-722`(`e.physical`는 719행, `e.spell`는 721행 직후, `events := append(...)` 이전).
- 선례: `sure_crit`는 `exchange()`의 `combat.go:862`에서 지운다. 책략 경로(`spell`)에는 같은 삭제가 없다. 보복은 물리·책략 모두 소모하므로 `useSkill`이 맞다.
- 반격·반계·지원 공격은 `useSkill`을 거치지 않으므로 소모되지 않는다. 요청("반격 등 다른 반응에는 적용하지 않는다")과 맞다.
- 공격 **중에** 받은 반격으로 쌓인 보복은 `physical`이 끝난 뒤 지워진다(Q12).

(c) 턴 시작 시 안전장치(선택): `combat.go:1307-1308` 옆에 같은 삭제를 두면 세이브 로드 등으로 남은 잔여도 비워진다. 필수는 아니다.

### 3-4. 세이브 검증과 파진
- 상태 id는 `content/vocab.go` `Statuses`에 등록해야 한다. 세이브 검증은 `api.go:743` 부근(잘못된 id·Turns 범위·Source)이다. 이 줄은 이전 기록(retaliation-stack-duration notes)에서 가져왔고 이번에 재확인하지 않았다.
- `Status.Source`는 비운다. 공격자 id를 넣으면 공격자가 사라진 뒤 손상 판정이 날 수 있다.
- 파진 `stripBuffs`(`combat.go:1122`)는 `buffStatuses`(`combat.go:1117`) 목록에 있는 것만 지운다. 보복 id를 넣을지는 Q13.

### 3-5. 예시 순서
1. 적 턴: 아군 A가 2회 피격 → 보복 집계 2(집계 조건은 4절).
2. 아군 A 턴: 대기(또는 아이템) → `endFaction`에서 전부 삭제. 0.
3. 아군 A 턴: 직접 공격 → `useSkill`의 공격 후 삭제. 0. 공격 중 반격으로 받은 집계는 같은 공격 뒤에 지워져 다음 턴으로 넘어가지 않는다.

## 4. 피격 집계 조건 (요청 3, 선택지)

집계 지점 후보: `strike()`의 `combat.go:333` 직후(`n := e.hurt(...)` 뒤). `strike`는 `blow`(`combat.go:289-292`: share, derived, second)를 받는다. `derived`는 `strike` 안에서 쓰이지 않고 `attackOn`(435행)에서만 쓰이므로, 집계에 쓰려면 `strike`에 넘겨야 한다.

| 옵션 | 조건 | 장점 | 단점 |
|---|---|---|---|
| A | `hit && n > 0` (명중하고 피해가 들어간 blow만) | 빗나감·면역 0 피해 제외, 직관적 | 면역 대상은 안 쌓임 |
| B | `hit` (0 피해 포함) | 면역 대상도 쌓임 | "맞았다"의 뜻이 흐려짐 |
| C | 공격 시도 전부(빗나감 포함) | 가장 단순 | 빗나간 공격에도 쌓여 요청과 어긋날 수 있음 |
| D | 적의 직접 공격만(`!b.derived`) | 반격·반계·지원 피격 제외 | 반격으로 맞은 것은 안 쌓임(요청 의도 확인) |
| E | 반격(derived)도 포함 | 맞은 것은 전부 셈 | 반격 연쇄가 중첩을 키움 |
| F | blow 단위 / 공격 단위 | 쌍타(`second`)를 두 번 셀지 한 번 셀지 | 쌍타 스킬 체감 차이 |
| G | 지속 피해(`dot`, 독·화상, `combat.go:1437-1445`) | 기본 제외 권장 | 포함하면 턴 끝 피해로 쌓임 |

- 기본 제안: A와 D(또는 E)를 조합하고 G는 제외. F는 blow 단위가 단순하다(쌍타는 두 번 집계, Q14).
- 중첩 상한은 상태 Turns 1~99, Value ±100 검증(`api.go:743`)과 맞춰 정한다(Q15).

## 5. 사용자에게 물을 질문
- Q1. 빗나간·면역 책략에도 반계를 막을까? (기본: 막는다)
- Q2. 반계 지형 기준 칸: 되맞는 쪽 u(기본) 또는 반계 시전자 q?
- Q3. 반계 치명이 나면 표식을 찍을까? (기본: 찍는다)
- Q4. 반계에 디버프·둔화·버프 해제·연쇄를 모두 적용할까? (기본: 전부)
- Q5. 사거리 제한을 둘까? (현행: 없음)
- Q6. 반계 피해를 원 시전자의 spell-hit 집계에 넣을까, 전용 이벤트 `reflect`로 구분할까?
- Q7. 같은 편 책략을 반계하는 경우도 허용할까? (현행: `q.Faction`을 보지 않는다)
- Q8. 반계 blow의 명중 굴림은 그대로 둘까, 원 책략 명중을 상속할까?
- Q9. `KindMixed` 스킬은 직접 공격인가?
- Q10. 후속 공격(`battle_events.go:264`)은 직접 공격인가? (기본: 아니다)
- Q11. "-1 per turn"(이전 설계)을 버리고 "자기 군 턴 끝에 전부 소멸"로 확정할까?
- Q12. 공격 중 반격으로 쌓인 보복은 공격 후 지울까(기본: 지운다)?
- Q13. 파진(`stripBuffs`)에 지워지게 할까? (기본: 아니다)
- Q14. 피격 집계는 4절 A~G 중 무엇인가? 쌍타는 두 번 셀까?
- Q15. 중첩 상한과 중첩당 수치(예: +5%)는 얼마로 할까?

## 6. 미검증 항목
- GUI가 반계 피해·표식·지형 변화를 어떻게 재생하는지 읽지 않았다(기존 메모의 "attack 고정 모션"도 미확인).
- `damage()`의 원소·패긴 배율 세부, `hitChance`/`critChance` 수치는 호출 관계만 읽었다.
- `counterShare`·`strikeBack`의 `counter` 인자가 반격과 직접 공격을 가르는 흐름은 전체를 읽지 않았다.
- 세이브 검증(`api.go:743`) 전체와 `Statuses` 등록(`content/vocab.go`)은 이번에 재확인하지 않았다.
- 실행 테스트, 전투 시뮬레이션, 게임 캡처는 하지 않았다. 반계 빗나감·면역 재현은 `internal/core` 테스트로 확인해야 한다.
- 법정·반계 특성의 실제 배정(characters.json)은 확인하지 않았다.

## 7. 출처
- 반계: `ver5/internal/core/combat.go:950-977`, `ver5/internal/core/combat.go:297-337`(strike), `ver5/internal/core/combat.go:690-725`(useSkill), `ver5/internal/core/combat.go:1066-1085`(spell 후처리), `ver5/internal/core/battle_events.go:320`(markCell), `ver5/assets/data/traits.json:590-610`.
- 보복 소멸: `ver5/internal/core/combat.go:1240-1263`(endFaction), `1296-1310`(턴 시작), `1353`(Attacked 초기화), `ver5/internal/core/rules.go:251`(skillDef), `ver5/internal/core/api.go:58-62`(커맨드 종류).
