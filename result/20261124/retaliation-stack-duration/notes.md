# 법정 보복 중첩을 버프 지속 턴으로 처리할 수 있는가

게임 데이터(ver5/assets/data/*.json)와 Go 코드는 바꾸지 않았다. 아래는 현행 코드 읽기 결과와 제안이다. 실행 테스트는 하지 않았다.
요청 원문: process/retaliation-stack-duration-20261124/retaliation-stack-duration-20261124.md
이전 기록: result/20261124/fazheng-reprisal-counterspell/notes.md, result/20261124/fazheng-earth-reprisal/design.md
(아래 파일 경로는 모두 /Users/younwookim/orca/mhdemo1/ 아래다.)

## 1. 질문

법정 특성(요청서는 "보복", 이전 설계 문서는 "응징". 기존 `retaliate`가 이미 "응징"이므로 새 이름이 필요하다)의 동작:

- 피격 1회마다 +1 중첩
- 자기 턴이 끝날 때 미사용이면 -1
- 자기 턴의 직접 공격(일반 공격, 공격 책략, 무예)에 쓰면 중첩 적용 후 소멸

질문: 기존 상태의 지속 턴(Turns) 메커니즘을 중첩 표현에 재사용할 수 있는가?

## 2. 현행 메커니즘 (file:line, 경로는 ver5/internal/ 기준)

### 2-1. 상태 저장 구조

- `Status{Turns, Value int; Source string}`: `core/types.go:21-24`. 유닛마다 `Unit.Status map[string]Status`.
- 상태 부착 `applyStatus(v, id, turns, value)`: `core/combat.go:252-268`.
  - 같은 부호이면 더 큰 |Value|를 남기고 Turns는 `max`(261~266행). **중첩해서 더하지 않는다.** 같은 id를 다시 걸어도 Turns가 늘지 않는다.
  - 부호가 반대면 상태를 지운다(257~259행, 상쇄).
  - 따라서 중첩을 `applyStatus`로 쌓을 수 없다. 중첩 쓰기는 별도 코드가 필요하다.

### 2-2. 지속 턴 감소 (소유자 군의 턴 끝)

- `core/combat.go:1240` `endFaction()`.
  - 1243~1250행: 이번에 끝나는 군(`ending`, `endingArmy`)에 속한 유닛(`u.ActsIn(ending, endingArmy)`)만 처리한다.
  - 1253~1263행: `for _, k := range keys(u.Status)`에서 `s.Turns--`(1258행), `Turns <= 0`이면 삭제(1259~1261행), 아니면 저장(1262행).
  - 1255행: `counter`와 `holdGround`는 감소에서 **제외**된다. 제외 목록을 두는 선례다.
- 결론(요청 a·b): **감소는 턴 시작이 아니라 소유자 군의 턴이 끝날 때** 일어난다. 피격은 보통 적 턴에 일어나므로, 적 턴에 쌓인 중첩은 아군 턴이 끝날 때 한 번 줄어든다. "-1 per turn if unused"와 자연스럽게 맞는다(3절).

### 2-3. fx와 Value

- `fx(u, key)`: `core/engine.go:360-368`. 상태의 `Turns > 0`이면 `content.StatusEffects[id][key]`라는 **고정 값**만 더한다. `Value`는 읽지 않는다. 따라서 `fx`로 "중첩 × 5%"를 만들 수 없다.
- `Value`를 %로 쓰는 곳: `core/rules.go:80-86`(attack/defense/mind/morale/agility/disrupt), `rules.go:186, 344, 363, 391, 426`(speed/range), `combat.go:87`(hold_ground). 새 id가 `content.StatStatuses`(`content/vocab.go:211`)에 없으면 이 경로와 무관하다.
- `Turns`를 "지금 활성인가"로 읽는 곳: `fx`(engine.go:363), 책략 막기(`combat.go:86, 154, 199`), 혼란(`combat.go:758`), 뿌리(`engine.go:805`, `rules.go:195`), 반격 태세(`combat.go:515`).

### 2-4. 피격 지점

- 피해 적용은 `hurt(v, n, u)`(`combat.go:270-283`)가 공통이다. 호출처는 `combat.go:305`(화덕성군, 0 피해), `combat.go:333`(`strike` 본체), `combat.go:1445`(`dot`, 독·화상 등 지속 피해).
- `strike()`(`combat.go:297`)는 빗나가면 `emit("miss")` 후 반환하므로(317~325행 부근) 빗나간 공격은 `hurt`에 닿지 않는다. 명중이면 333행 뒤에서 `hit && n > 0`을 확인할 수 있다.
- 반격·반계·지원 공격도 모두 `strike`를 지나며 `blow.derived`로 표시된다(`combat.go:289-292` `blow`, 406행 `attackOn(..., derived, ...)`). 반계 호출은 `combat.go:974-975`.

### 2-5. 직접 공격과 소모 지점

- 직접 공격 진입 `useSkill(u, c)`(`combat.go:690`): 병법치 소모 701~703행, `c.Kind == "attack"`이면 `u.Attacked = true`(715행), `s.Kind == KindPhysical`이면 `e.physical`(719행), 아니면 `e.spell`(721행).
- 반격·반계·지원은 `useSkill`을 거치지 않는다(`combat.go:974`은 `strike` 직접 호출). 따라서 **`useSkill`에서 소모하면 반격·반계가 자연스럽게 제외된다.**
- 선례 `sure_crit`(휘기, 다음 물리 공격 반드시 치명): `exchange()`에서 `delete(u.Status, "sure_crit")`(`combat.go:862`). 자기 공격 후 상태를 지우는 기존 패턴이다. 단 `spell()` 경로에는 같은 삭제가 없어 책략에는 소모되지 않는다. 중첩은 물리·책략·무예 모두 소모해야 하므로 `exchange`에만 두면 부족하다.
- 무예 탭: `assets/data/skills.json`에서 `"Tab": "martial"`은 반격(counter, Kind buff), 장전(reload, buff), 조준(aim, buff) 3개뿐이다. 물리 공격 스킬(무쌍, 난사, 십자사격, 연발, 관통, 화시, 저격)은 `Kind: physical`이고 Tab이 비어 있다. "무예"를 직접 공격으로 셀지는 정의 문제다(6절 질문 3).

### 2-6. 저장·검증 제약

- 세이브 검증 `core/api.go:747`: `content.Statuses[id]`에 없는 id, `Turns < 1`, `Turns > 99`, `abs(Value) > 100`, 알 수 없는 `Source`면 손상 세이브로 판정된다.
- 상태 id 등록: `content/vocab.go:217-225` `Statuses`. 새 id는 여기 등록해야 한다.
- 함의: Turns에 중첩을 두면 최대 99. Value에 두면 절댓값 100. 중첩 상한이 5~20 정도면 둘 다 안전하다.
- Turns는 1 이상이어야 저장된다. 중첩이 0이면 상태를 **삭제**해야 한다(감소 루프 1259행이 이미 그렇게 한다).

### 2-7. 해제와 표시

- 파진류 `stripBuffs`(`combat.go:1124-1133`)는 `buffStatuses`(1117행) 목록과 능력치 버프만 지운다. 새 id를 목록에 넣지 않으면 파진에 지워지지 않는다(설계 결정).
- GUI가 `Turns`를 "남은 턴"으로 그리는지는 확인하지 않았다(미검증).

## 3. 지속 턴 재사용 시 보복 규칙이 맞는가

| 요청 규칙 | Turns 재사용 시 | 맞나? |
|---|---|---|
| 피격 +1 중첩 | 피격 때 Turns를 `old+1`로 직접 쓴다(applyStatus 금지) | 가능. applyStatus는 max라서 쓰면 안 된다(252~268행) |
| 턴 끝 미사용 -1 | 소유자 턴 끝 감소(1258행)가 자동으로 -1 | 맞다. 추가 코드 없음 |
| 0이면 소멸 | 감소 루프가 삭제(1259~1261행) | 맞다 |
| 직접 공격 후 전부 소멸 | `delete(u.Status, id)` | 맞다(sure_crit 선례) |
| 중첩 수치 적용(+5% × 중첩) | `fx`는 Turns 값을 쓰지 않는다(363행). `damage()`/`boost()`에 새 분기 필요 | 새 코드 필요 |

### 3-1. 맞는 지점

- 감소와 소멸이 공짜다. 감소 루프와 `Turns <= 0` 삭제가 이미 있다.
- 세이브 검증(Turns 1~99)과도 맞는다.

### 3-2. 어긋나는 지점

1. **Turns의 뜻은 "남은 기간"이다.** 중첩을 Turns에 두면 한 필드가 "얼마나 남았나"와 "몇 겹인가"를 동시에 맡는다. `applyStatus`의 max 병합(265행), 상쇄(257행), 재부착(261행)은 모두 Turns를 기간으로 가정한다. 중첩 쓰기는 이 규칙을 우회해야 한다.
2. **감소 시점.** 감소는 소유자 턴 끝에 일어난다(1240~1262행). 적 턴에 맞아 쌓인 중첩은 아군 턴 끝에 -1된다. 아군 턴 **중**에 생긴 중첩(자기 공격 중 반격을 받음 등)도 같은 턴 끝에 -1된다. 소모 시점과 함께 정해야 한다(6절 질문 2).
3. **fx 미사용.** 중첩 수치는 `fx`가 아니라 `damage()`(`combat.go:28`)/`boost()`(`combat.go:108`)에서 읽어 곱해야 한다. 상태 id는 `StatusEffects` 표(vocab.go:161)에 넣을 필요가 없다.
4. **표시 의미.** 표시가 Turns를 기간으로 그린다면 "중첩 3"이 "3턴 남음"으로 읽힌다(미검증).

## 4. 실행 가능한 방안 (추천 순)

### 방안 1 (추천): 전용 상태 id, 중첩은 Value, 감소 한 분기 추가

- `Statuses`에 id 등록(vocab.go:217). `Turns`는 보유 표시용으로 1 이상을 두고, Value를 중첩으로 쓴다.
- 피격: `strike()`에서 `hit && n > 0`이면 Value를 +1(상한 적용).
- 감소: `endFaction` 루프(1253~1262행)에 이 id만 Value를 -1(0이면 삭제). 이 한 분기가 "-1 per turn"이다.
- 소모: `useSkill`에서 physical/magic 스킬 실행 뒤(715~721행 부근) 삭제. 적용: `damage()`에서 Value를 읽는다.
- 장점: 세이브 검증과 상태 표시 체계에 그대로 들어가고, 중첩과 기간을 분리한다. 변경은 vocab 1줄, strike 2~3줄, endFaction 2~3줄, useSkill 2~3줄, damage 2~3줄 정도.
- 단점: 감소 분기가 특수 취급이다.

### 방안 2: Turns를 중첩으로 재사용 (코드 최소)

- 피격: `u.Status[id] = Status{Turns: old.Turns+1}`(applyStatus 금지). 감소·삭제는 기존 루프가 그대로 한다.
- 소모: 삭제. 적용: `damage()`에서 Turns를 곱한다.
- 장점: 감소 분기가 필요 없다. 가장 짧다.
- 단점: 의미 과부하(3-2의 1·4). 중첩이 99를 넘지 않도록 막아야 한다. 적 턴 피격 +1과 아군 턴 끝 -1이 맞물려 "맞은 직후 한 번은 살아남는다"는 순서가 숨은 규칙이 된다.

### 방안 3: 유닛 필드 추가 (가장 깨끗함, 변경 큼)

- `Unit`에 `Retaliation int \`json:",omitempty"\`` 추가. 감소·소모는 방안 1과 같은 자리에서 처리.
- 장점: 상태 체계와 분리되어 의미가 가장 명확하다.
- 단점: 표시·파진·세이브 검증을 따로 붙여야 한다. 프로젝트 규칙상 큰 diff는 허용되지만 표시 작업이 늘어난다.

## 5. 추천

**방안 1.** 방안 2는 감소를 공짜로 얻는 대신 applyStatus 규칙과 충돌하고, 나중에 "남은 턴" 표시와 섞인다. 중첩은 Value에, 감소는 endFaction의 한 분기로 두는 편이 의미가 맞고 변경도 작다. 빠른 시제품은 방안 2로 먼저 돌려 보고, 표시를 붙일 때 방안 1로 옮기는 것도 가능하다.

공통 원칙:
- 피격 집계는 `strike()`에서 명중(`hit`)이고 `n > 0`일 때만 센다(빗나감·면역 0 피해 제외, 질문 5).
- 소모는 `useSkill`에서 physical/magic 스킬을 실행한 **뒤**에 한다. `exchange`나 `spell` 안에 두면 경로가 섞일 수 있다.
- 중첩 적용은 `damage()`에서 공격자 자신의 중첩을 읽는다. 반격(`counterShare`, `combat.go:513`)과 반계(`combat.go:974`)는 `useSkill`을 거치지 않으므로 소모 대상이 아니다. 단 적용 여부는 질문 2·3에 따라 정한다.

## 6. 설계 질문 (사용자 확인 필요)

1. 이름: "보복"(요청서)으로 할지, 이전 설계의 "응징"으로 할지. 기존 `retaliate`가 "응징"이므로 "보복"이 적합해 보인다.
2. 소모 순서: 자기 공격 중 맞은 반격으로 생긴 중첩을 공격 **후** 지울지, 공격 **전**의 중첩만 적용할지. 공격 중간에 생긴 중첩은 같은 턴 끝 -1에 걸린다.
3. "무예": 물리 공격 스킬(무쌍·난사 등, `Kind: physical`)만 셀지, 무예 탭 전체(반격·장전·조준 포함)를 셀지. 반격·장전·조준은 공격이 아니다. 기본 제안: `Kind`가 physical 또는 magic인 스킬만 소모.
4. 피격 기준: 반격(derived)으로 맞은 것도 셀지, 지원 공격·반계·지속 피해(`dot`, 독·화상)도 셀지. 기본 제안: 적의 직접 공격과 반격으로 맞은 피격을 센다(지속 피해 제외).
5. 빗나감과 면역 0 피해(`combat.go:305`)는 세지 않는 것으로 확정할지.
6. 상한: 중첩 최대치(예: 5 또는 10)와 중첩당 수치(+5%)를 정한다. 상한이 없으면 세이브 검증(Turns ≤ 99, |Value| ≤ 100)에 걸릴 수 있다.
7. 부관 전달: 법정이 부관으로 주장에게 주면 피격 주체가 주장이 되고, 상태도 주장 유닛에 붙는다. 소모도 주장의 직접 공격에서 일어난다. 구조상 가능하나 테스트가 필요하다.
8. 파진(stripBuffs)에 지워질지 여부(`combat.go:1117` 목록에 넣을지).
9. 적 턴에 맞고 아군 턴 끝 감소 전에 쓰지 않았을 때 "한 번 줄어든 뒤 남는" 체감이 요청과 맞는지.

## 7. 미검증 항목

- GUI가 상태의 Turns를 "남은 턴"으로 표시하는지(GUI 코드 미확인).
- 중첩 상태에 `Status.Source`(공격자 id)를 쓰면 세이브 검증(`api.go:747`)의 Source 조건에 걸리는지. 공격자가 사라진 경우 `e.unit(v.Source) == nil`이면 손상 판정된다. 중첩에는 Source를 비워 두는 편이 안전하다.
- `endFaction` 감소가 AI·동맹군에도 같은 규칙으로 작동하는지(`ActsIn` 조건만 읽었다).
- `useSkill`에서 `e.physical`/`e.spell` 뒤에 소모할 때 이벤트 후처리(`retreat` 연쇄, 723행 이후)와 충돌하는지는 읽지 않았다.
- 실행 테스트와 전투 시뮬레이션은 하지 않았다. 감소·소모 순서는 `internal/core` 테스트로 확인해야 한다.
- 법정의 실제 특성 배정(characters.json)은 확인하지 않았다.

## 8. 진행 경로 (제안)

1. 사용자가 6절 질문 1·2·3·4에 답한다.
2. 방안 1 또는 2로 상태 id와 감소·소모 분기를 정한다.
3. 테스트 3개: (a) 적 턴 피격 2회 후 아군 턴 끝에 중첩 1, (b) 자기 물리 공격 후 전부 소멸, (c) 반격 피격의 중첩 처리(질문 4 결과대로).
4. 데이터·코드 변경은 승인 후에만 진행한다.

## 9. 요약

- 중첩을 Turns에 쓰는 것은 가능하다. 감소와 0 소멸이 `endFaction`(combat.go:1240~1262)에서 공짜로 된다.
- 다만 `applyStatus`는 Turns를 max로 합치고(252~268행), `fx`는 Turns가 0보다 큰지만 본다(engine.go:363). 중첩 쓰기는 우회해야 하고, 수치는 새 코드로 읽어야 한다.
- 추천: 중첩은 전용 상태의 Value, 감소는 `endFaction`에 한 분기, 소모는 `useSkill`의 physical/magic 실행 뒤. 빠른 시제품은 Turns 재사용으로 가능하다.
- 선례: `sure_crit`(combat.go:862)가 자기 공격 후 상태를 지우는 같은 패턴이다.
