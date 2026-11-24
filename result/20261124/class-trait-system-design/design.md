# 병종 특성 체계 재설계 (초안, 미적용)

작성일 2026-11-24. 게임 코드·데이터는 바꾸지 않았다. 아래 수치는 `ver5/assets/data/classes.json`과 `traits.json`을 읽어 계산한 것이다.

## 결론

- 병종 특성은 **등급별 학습 풀(Pool) 개수**로 맞춘다: 기본 3, 중급 5, 상급 5. 중급이 6개 이상이면 상급은 4개까지 줄인다.
- 확정된 조정(책사 격파 제거, 화계·수계·지계 강화 군사 이동, 세객 수계 제거, 혼란 종횡가 이동)은 반영했다. 세객 파진·장구·약점공략은 **미정**으로 표시만 하고 자리를 비워 두었다.
- 주장과 부관이 같은 특성을 가질 때 **능력치 상승 특성만** 합산하는 규칙을 제안한다. 현재 코드는 같은 특성을 한 번만 센다(중첩 없음). 근거는 아래 3절.
- 궁기병은 지금 정체성이 약하다(저격·조준은 궁병과, 기마술은 경기병과 겹침). 추천은 옵션 B: **백발백중 + 저지 사격**. 둘 다 데이터에 이미 있고 아무 풀에도 없으며 새 코드가 필요 없다.

## 1. 원칙

1. 개수 기준은 `Pool`(학습 풀)이다. `Plan`(레벨별 자동 습득)은 개수에 넣지 않는다. 이 기준은 기존 요약(`result/20261124/class-trait-count-20261124/result.md`)과 같다.
2. 기본 3 / 중급 5 / 상급 5. 중급이 많으면(6개 이상) 상급 4. 기본은 "약 3"이므로 ±1을 허용한다.
3. 규칙을 바꾸는 특성을 우선한다(`memory: srpg-trait-design-feedback`). 퍼센트 보정만 있는 특성은 같은 값이면 뒤로 민다.
4. 특성 이름은 효과가 읽히는 쉬운 말로 짓는다(`memory: trait-names-plain`).
5. 장수 고유로 남긴 특성은 병종 풀에 넣지 않는다: 함진(고순), 난사(황충), 기사(여포·방덕), 돌격(적토마·전해), 연환(방통), 괄목상대(여몽), 간접공격면역(여포·조운), 패도(조조 고유; 데이터에서는 풀에 없음). 기사는 궁기병 후보에서 뺀다(`memory: trait-owner-reservations`: "기사가 흔하면 궁기병으로 안 간다").
6. 민중(`commoner`)은 비전투 병종이라 제외한다.

## 2. 병종별 제안 (Pool 기준)

표기: `현재 → 제안 (개수)`. "확정"은 이미 합의된 조정이고, "제안"은 이번에 낸 안이다. 개수가 맞는 병종은 "유지"로 적었다.

### 기본 (목표 3)

| 병종 | 현재 | 제안 | 비고 |
|---|---|---|---|
| 경보병 | 3 | 유지 (3) | diagonal, fire, brace_prep |
| 경기병 | 3 | 유지 (3) | dash, accuracy_up, grass_footing |
| 궁병 | 4 | 3: relief, accuracy_up, aim_prep | snipe_prep 제외 (저격은 궁기병 정체성으로) |
| 서생 | 3 | 유지 (3) | fire, relief, foresight |
| 적병 | 3 | 유지 (3) | counter_art, earth, plunder |
| 황건적 | 2 | 3: counter_art, earth, brace_prep | 약간 적음, 필수 아님 |
| 무도가 | 3 | 유지 (3) | spirit_prep, counter_art, fire |

### 중급 (목표 5)

| 병종 | 현재 | 제안 | 비고 |
|---|---|---|---|
| 중보병 | 3 | 5: deputy_appoint, iron_armor, ranged_ward, block_up, diamond | 방어확률증가·금강 추가 |
| 장창병 | 4 | 5: + breakthrough | 돌파 추가 |
| 중기병 | 3 | 5: + vanguard, footwork | 선봉은 중첩 규칙 시험 대상(3절) |
| 궁기병 | 3 | 5: deputy_appoint, snipe_prep, aim_prep, sure_shot, suppressing_shot | 옵션 B (4절) |
| 노병 | 4 | 5: + accuracy_up | 연노병의 백발백중으로 이어지는 사다리 |
| 방술사 | 5 | 유지 (5) | |
| 흉적 | 4 | 5: + anti_cavalry | 대기병 전술 |
| 권사 | 2 | 5: + drain, hundred_foes, lone_warrior | 1대1 정체성 |
| 책사 | 9 | 5: deputy_appoint, spread, water, earth, focus | **확정**: 격파 제거, 화계강화·수계강화·지계강화 군사로 |
| 세객 | 11 | **6 확정**: deputy_appoint, spell_lower_agility/morale/attack/defense/mind | **확정**: 수계 제거, 혼란 종횡가로. **미정 3개 제외**: break_formation(파진), lasting(장구), weak_point(약점공략). 미정이 하나라도 세객에 남으면 6→9가 되므로 다시 정해야 한다 |

세객의 6개는 "중급이 많다"에 해당한다(6 ≥ 6). 그래서 상급 종횡가는 4개로 잡는다.

### 상급 (목표 5, 중급이 많으면 4)

| 병종 | 현재 | 제안 | 비고 |
|---|---|---|---|
| 전차대 | 3 | 5: vital_strike, charge_strike, impact, pierce, breakthrough | |
| 근위병 | 3 | 5: castle_defense, siegecraft, composure, diamond, fortune | castle_defense(중갑)는 중첩 대상 |
| 친위대 | 4 | 5: diagonal, charge_strike, impact, vital_strike, vanguard | vanguard 추가. 중첩 시험 2순위 |
| 노기병 | 3 | 5: reload_prep, fire_arrow_art, rapid_art, far_sight, cross_shot_art | 사거리·십자사격. 중복 가능 특성은 연노병·발석거와 겹침 |
| 연노병 | 4 | 5: + sure_shot | 노병의 accuracy_up 다음 단계 |
| 발석거 | 3 | 5: siegecraft, cross_strike, far_sight, suppressing_shot, accuracy_up | |
| 군사 | 1 | 4: terrain_craft, fire_boost, water_boost, earth_boost | **확정**: 강화 3종 이동. 확산(spread)은 책사에 남겼으므로 4개. 5개로 하려면 확산도 옮길지 물어볼 것 |
| 종횡가 | 1 | 4: stratagem_spread, stratagem(**확정 이동**) + 미정 2개 | **미정**: 파진·장구·약점공략 중 2개를 여기로. 세객이 6개라 상급은 4개 |
| 선술사 | 3 | 5: exhort_boost, lasting_buffs, terrain_mend, mind_aura, lecture | |
| 도독 | 8 | 5: deputy_appoint, fire, water, spread, exhort | 화계·수계 강화와 구휼 제외. 개수 초과 조정이므로 확인 필요 |
| 의적 | 3 | 5: vital_strike, crit_boost, relief, pursuit, do_or_die | |
| 권성 | 3 | 5: spirit_spread, palm_wind, drain, pursuit, deadly | fire_boost는 군사 정책에 맞춰 뺌 |

### 집계 (제안 적용 가정, 미정 제외)

- 기본 7종: 전부 3 (황건적은 3으로 올림).
- 중급 10종: 세객 6, 나머지 9종은 5.
- 상급 12종: 군사 4, 종횡가 4(미정 포함 시), 나머지 5.
- 자동 확인: `proposal` 계산 스크립트(scratchpad)로 모든 특성 id가 `traits.json`에 있는지 확인했다. 없는 id는 없다.

## 3. 주장·부관 중첩 규칙

### 현재 코드가 하는 일 (근거)

- `ver5/internal/core/engine.go:340-357` `traits(c)`: 특성을 **map(set)** 에 넣는다. 주장의 innate·Learned·장비 특성을 넣고(346-349), 부관이 있으면 부관의 innate와 Learned를 같은 set에 넣는다(352-355). 장비 특성은 부관에게서 건너오지 않는다(`logic-traits.md:41`과 일치).
- `engine.go:360-367` `fx(u, key)` 와 `engine.go:369-376` `fxOf`: `traits(c)`의 결과를 **중복 없이** 순회해 효과 값을 더한다. 주장과 부관이 같은 특성을 가지면 한 번만 더해진다. 다른 특성끼리는 더해진다.
- `engine.go:378-385` `ownFx`: 부관 특성을 뺀다. 용인술(deputy_plus)과 부관 능력치 섞기에만 쓰인다(`formation.go:148`, `formation.go:163`).
- `engine.go:1111-1123` `turnBuffs`: 매 자기 차례 버프 값을 `fx(u, "turn_buff:"+id)`로 구한다. 따라서 같은 특성이 두 번 세어지지 않는다.
- `combat.go:1338-1340`: 턴 버프를 `applyStatus`로 건다. `combat.go:252-268`의 `applyStatus`는 같은 상태가 있으면 **절댓값이 큰 쪽을 남긴다**. 여러 출처를 더하지 않는다.
- `combat.go:1366-1393` 오라 버프: `once(key)`가 같은 키를 **인접 아군 여럿이어도 한 번만** 센다(`got[key]`). 오라는 이미 중첩이 없다.
- `combat.go:380-390` `bulwark`: 철벽·철갑은 합산하지 않고 **작은 값**을 쓴다. 중첩을 막는 선례다.
- `rules.go:37-76` `stats()`: 포위 수(outnumbered)·동료 수(companions)·일기당천(lone)·돌진(charge_power)은 칸 상황을 보고 곱한다.
- `formation.go:131-135` `DeputyTraits`: 부관이 주는 특성은 innate와 Learned. 장비는 제외.

결론: **같은 특성은 주장·부관이 둘 다 가져도 한 번만 적용된다.** 중첩은 현재 없다.

### 제안 규칙

1. **중첩 대상은 "능력치를 항상 올리는 자기 버프 특성"만**으로 한다. 데이터에서는 `turn_buff:*` 효과를 가진 특성: 선봉(vanguard, 공격 +20), 중갑(castle_defense, 방어 +20), 학창(crane_calm, 정신 +20). 패도(hegemony)는 조조 고유이므로 제외한다.
2. 상황에 따라 값이 변하는 특성(일기당천 lone, 포위 수 outnumbered, 동료 수 companions, 돌진 charge_power)은 **중첩하지 않는다.** 이미 주변 칸을 보고 계산하므로 부관이 같은 특성을 가져도 칸 수는 같다. 중첩하면 같은 칸 수에 두 배가 되어 과하다.
3. 오라(aura_buff)는 지금처럼 **중첩하지 않는다**(`once`). 인접 아군이 여럿이어도 같다.
4. 범위는 **주장과 그 부관 한 명** 사이로 한정한다. 부관은 한 명이므로 최대 2회 합산이다. 인접 아군 간에는 중첩하지 않는다.
5. 합산 상한은 한 능력치당 +40%(두 번 중첩 시 상한)로 제안한다. 상한 없이도 두 배지만, 선봉 두 개면 공격 +40%가 매 차례 유지되므로 밸런스 확인이 필요하다(열린 질문).

### 구현 스케치 (코드 변경 아님, 참고)

- `traits.json`의 해당 특성에 `"Stack": true` 같은 표시를 추가한다(또는 `game.json`에 목록).
- `traits(c)`를 **map이 아닌 목록**으로 바꾸되, 주장 쪽은 기존처럼 중복 제거하고, **부관 쪽에서 `Stack` 특성만** 중복을 허용해 덧붙인다.
- `fx`·`fxOf`는 목록을 순회하므로 자동으로 합산된다. `turnBuffs`도 그대로 맞는다.
- 주의: `traits()`는 UI 표시(`api.go:827`, `engine.go:215`), 학습 가능 판정(`engine.go:391`), 장비 판정(`engine.go:947`)에도 쓰인다. UI용 목록은 중복을 지우고 "×2" 같은 표시를 따로 줘야 한다. 학습 판정은 `contains`만 쓰므로 영향이 없다.
- `applyStatus`의 절댓값 규칙 때문에 합산한 값 하나만 들어간다. 별도 수정이 필요 없다.

## 4. 궁기병 정체성 옵션

현재 궁기병(`classes.json` `horse_archer`): 공격 `ranged`, 사거리 3, 최소 사거리 2(`MinRange`), 이동 5, 풀 = 부관임명·저격(snipe_prep)·조준(aim_prep), 계획 = 초원 보조(10), 기마술(20). 문제는 저격·조준이 궁병 풀에도 있고(궁병은 저격도 있음, 제안에서는 뺌), 기마술·초원은 경기병과 같다는 것이다.

### 옵션 A: 돌진 사격 (돌진을 궁기병에 넣기)

- 내용: 궁기병 풀에 기존 돌진(charge_strike, 이동 칸 × 공격력 +5%)을 넣는다.
- 장점: 새 코드 없음. 돌진은 공격력 스탯에 더해지므로(`rules.go:76`) 원거리 공격에도 적용될 것으로 보인다. 말을 달려 쏜다는 그림이 바로 읽힌다.
- 단점: 돌진은 전차대·친위대의 근접 정체성이다. 궁기병에 넣으면 기병이 "달리면서 쏘는" 쪽으로 가는데, 최소 사거리 2 때문에 멀리서 쏘려면 많이 움직여야 해서 이동-사격 리듬이 생긴다. 원거리 피해에 공격력이 쓰이는지는 확인하지 않았다(미검증).

### 옵션 B: 백발백중 + 저지 사격 (추천)

- 내용: 궁기병 풀에 백발백중(sure_shot, 원거리 공격 반드시 명중, `combat.go:160`)과 저지 사격(suppressing_shot, 원거리 명중 시 대상 이동력 −2, `combat.go:441`)을 넣는다.
- 장점: 둘 다 데이터에 이미 있고 현재 어느 풀에도 없다. 새 코드 없음. "쏘면 반드시 맞고 적의 발을 묶는다"로 이름과 효과가 바로 읽힌다(`memory: trait-names-plain`). 연노병에 백발백중이 있으므로 노병 → 궁기병·연노병 사다리가 생긴다.
- 단점: 백발백중은 강하다(비용 1000). 저지 사격의 이동력 −2는 수치 규칙이라 메모리의 "이동 봉쇄 금지"에는 걸리지 않지만, 기병끼리 한 턴에 여럿이 맞으면 전선이 막힐 수 있다. 밸런스 확인 필요.

### 옵션 C: 기마 원사 (새 특성, 쏘고 빠지기)

- 내용: 새 특성 "사격 후 물러남"(공격 후 이동 가능) 또는 "기마 사격 명중 보정"(이동 칸 수 × 명중률).
- 장점: 궁기병다운 리듬(쏘고 빠지기)이 가장 강하게 나온다. 궁기병 고유 특성이 된다.
- 단점: 쏘고 난 뒤 이동은 행동 순서를 바꾸는 이동 메커닉이다. 메모리(`srpg-trait-design-feedback`)가 복잡한 이동 메커닉을 거절했고, 버그 위험이 크다. 명중 보정형은 기존 `charge_accuracy`(기습, 이동 칸 × 명중 +3%p)와 같은 계열이라 새롭지 않다. 기사(horse_archery)는 여포·방덕 고유라 쓸 수 없다.

### 추천

옵션 B. 새 코드 없이 정체성이 살아나고, 궁병과 기병 양쪽과 겹치지 않는다. A는 B 다음 후보(돌진은 이미 정예 창병 전용 정체성이라 궁기병에 두면 근접과 원거리가 섞인다). C는 메모리 기준과 충돌한다.

## 5. 열린 질문 (사용자 확인 필요)

1. 등급 기준이 **풀 개수**가 맞는지(계획 항목 수가 아니라). 계획은 레벨에 따라 자동으로 배우므로 풀과 다르다.
2. 세객 미정 3개(파진·장구·약점공략): 세객에 남길지, 종횡가로 보낼지, 장수 고유로 뺄지. 어느 쪽이냐에 따라 세객 6 또는 9, 종횡가 2~4가 정해진다.
3. 군사 상급을 4로 둘지, 확산(spread)도 옮겨 5로 할지.
4. 도독(8→5)과 권성(fire_boost 제외)에도 "강화 3종은 군사" 원칙을 적용할지. 제안에는 적용했다.
5. 중첩 규칙: 선봉·중갑·학창만 중첩할지. 상한(+40%)을 둘지. 주장·부관 중첩을 실제로 켜면 부관의 가치가 커지므로 부관 선택 흐름(`formation`)과 함께 볼지.
6. 황건적 3(브레이크 프리프), 중보병·중기병 추가 특성의 선택(block_up, diamond, vanguard, footwork 등)은 제안이며 취향 확인이 필요하다.
7. 궁기병 옵션 B를 고를지, A나 C를 원할지.

## 6. 미검증

- 게임 실행이나 전투 캡처로 확인하지 않았다. 모든 수치는 데이터와 코드 읽기에 근거한다.
- 돌진(charge_power)과 저지 사격이 원거리 피해에 실제로 적용되는지는 `combat.go`의 피해 계산 전체를 따라가지 않았다(미검증).
- `traits()` 호출처 전부를 확인하지 않았다(`api.go`, `engine.go`, `equipment_advice.go`, `combat.go:380`만 읽음). 목록을 map에서 바꾸면 UI 표시 중복이 생길 수 있다는 점은 추정이다.
- 풀에 특성이 있어도 `Plan`에서 자동 습득하는 것과 겹칠 수 있다. Plan은 바꾸지 않았다.
- 제안 특성의 밸런스(비용, 값)는 검토하지 않았다. 특히 백발백중+저지 사격, 선봉 중첩은 시뮬레이션(`cmd/_balance`) 확인이 필요하다.
- 기존 요약 `class-traits-proposed-count-20261124/summary.md`는 폴더만 있고 파일이 없다. 대신 `class-trait-count-20261124/result.md`와 `class-trait-discussion-20261124/result.md`를 근거로 삼았다. 두 요약의 괄호 값(승급 경로 합산)은 이번 기준(풀)과 다르다.
