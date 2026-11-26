# 보병·무도가 병종 특성 재설계 (논의안, 미적용)

작성일 2026-11-24. 게임 코드·데이터는 바꾸지 않았다. 수치는 `ver5/assets/data/classes.json`, `traits.json`, `characters.json`, `characters_story.json`, `skills.json`을 읽어 확인했다. 이전 날짜의 `class-trait-system-design/design.md` 2절 중 보병·무도가 행은 이 문서가 대체한다. 그 안의 장창병 `breakthrough`, 전차대 `pierce`, 권사 `hundred_foes`, 권성 `drain` 전체 값은 사용자 지시와 충돌하므로 버린다.

## 1. 지시 반영 확인

| 항목 | 특성 키 (`traits.json`) | 처리 |
|---|---|---|
| 진지고수 | `hold_ground` | 고유 유지, 풀·Plan 금지 |
| 호위 | `bodyguard` | 고유 유지, 풀·Plan 금지 |
| 찔러공격 | `pierce` | 고유 유지. 전차대 Pool과 Plan 60에서 뺀다 |
| 돌파 | `breakthrough` | 고유 유지. 장창병·전차대 제안에서 뺀다 |
| 기습 | `charge_aim` (명중 +3%p) | 고유 유지. 돌진(`charge_strike`)과 다른 특성이다 |
| 일당백 | `hundred_foes` | 고유 유지. 권사 제안에서 뺀다 |
| 의술 | `medicine` | 고유 유지 (번아만 보유) |
| 압박 | `press_prep` | 제외 (풀·Plan 어디에도 넣지 않는다) |
| 심공 | `drain` | 병종에는 10% 수준(`drain_weak`, 신규)만 준다 (3절) |
| 응급처치 | `first_aid` | 병종 후보로 사용. 중보병에 배치 |
| 반격강화 | `counter_boost` | 권성에 주지 않는 안을 기본으로 한다 (6절) |

유지할 기존 결정: 책사 격파 제거, 화계·수계·지계 강화 3종은 군사 쪽, 세객 수계 제거, 혼란은 종횡가, 파진(`break_formation`)은 고유 특성.

## 2. 원칙

1. **등급 규칙**: 기본 3 / 중급 5 / 상급 5 (`class-trait-system-design` 기준 그대로). 기본 등급은 Pool 3개다.
2. **계열 분리**: 보병 계열(경보병·중보병·장창병·근위병·전차대)과 무도가 계열(무도가·권사·권성)의 Pool에는 같은 특성을 넣지 않는다. 예외는 중립 특성 `deputy_appoint`(부관임명)뿐이다. Plan(레벨 자동 습득)도 같은 규칙을 따른다.
3. **겹치는 모양 금지**: 장창(`long_spear`)과 장풍(`palm_wind`)은 둘 다 상하좌우 2칸 근접 십자다. 보병이 장창을 가지므로 무도가 쪽 장풍은 뺀다.
4. **같은 역할 금지**: 대각공격(`diagonal`)은 보병 전용으로 둔다. 권사 Pool과 Plan에서 뺀다. 화계(`fire`)는 경보병에만 두고 무도가에서 뺀다.
5. **고유 특성과 압박은 풀·Plan에 들어가지 않는다.**
6. **강화 3종(`fire_boost`·`water_boost`·`earth_boost`)은 군사 전용**이므로 권성 Pool·Plan에서 뺀다.

## 3. 신규 특성은 하나만 필요

| 키 | 이름 | 제안 효과 | 비고 |
|---|---|---|---|
| `drain_weak` | 약한 심공 | 실제로 준 물리 피해의 10%만큼 병력을 회복한다. | 신규. 비용은 심공(500)보다 낮게(값 미정) |

- 심공을 낮추는 방법은 두 가지다. (a) 기존 `drain` 값을 10%로 바꾼다 — 현재 보유자는 없지만 `internal/core/traits_test.go:512`가 `drain`의 흡수 비율 0.5를 검증하므로 테스트를 고쳐야 한다. (b) 새 키 `drain_weak`을 만든다 — 기존 `drain`은 그대로 두고 병종에만 준다. **제안은 (b)**. 테스트를 건드리지 않고 병종 값만 낮춘다.
- 중복 정책: 같은 병력 회복 특성이 주장·부관에게 함께 있으면 기존 `fx` 중복 제거 규칙(`engine.go:360-376`)에 따라 한 번만 적용된다. 다만 `drain`과 `drain_weak`이 같은 인물에게 동시에 있는 경우는 현재 데이터에 없다(미검증).

나머지는 모두 기존 특성으로 구성했다. 기존 특성 목록에서 찾지 못해 새로 만들어야 할 특성은 이것 하나다. 보병의 밀집 방진 같은 새 특성은 기존 `iron_lock`(금쇄)·`four_generations`(사세삼공)와 같은 오라 계열이라 필요하지 않다고 판단했다.

## 4. 병종별 Pool 제안

표기: `현재 → 제안 (개수)`. 신규는 `drain_weak`뿐이고 나머지는 기존 특성이다.

### 4-1. 보병 계열

| 병종 | 등급 | 현재 Pool | 제안 Pool | 기존/신규 | 비고 (Plan) |
|---|---|---|---|---|---|
| 경보병 `light_infantry` | 기본 | diagonal, fire, brace_prep (3) | 유지 (3) | 기존 | Plan 10 diagonal 유지 |
| 중보병 `heavy_infantry` | 중급 | deputy_appoint, iron_armor, ranged_ward (3) | deputy_appoint, iron_armor, ranged_ward, block_up, first_aid (5) | 기존 | `block_up` = 방어확률증가(명중 −10%p), `first_aid` = 응급처치(스킬 `first_aid`, 자가 회복, 비용 10). Plan 10 diagonal, 20 brace_prep 유지 |
| 장창병 `pikeman` | 중급 | deputy_appoint, long_spear, siegecraft, assist (4) | + sure_footing (5) | 기존 | `sure_footing` = 답파(지형 보정 100 미만이면 100). 돌파 제외 |
| 근위병 `guards` | 상급 | castle_defense, siegecraft, composure (3) | castle_defense, siegecraft, composure, diamond, iron_lock (5) | 기존 | `diamond` = 금강, `iron_lock` = 금쇄(주변 8칸 아군 방어 버프). `fortune`(호운)은 대안 |
| 전차대 `chariot` | 상급 | vital_strike, charge_strike, impact (3) | + crit_boost, anti_cavalry (5) | 기존 | `crit_boost` = 치명타강화, `anti_cavalry` = 대기병 전술. **Plan 60 `pierce`는 제거**하고 빈자리를 다른 항목으로 채운다 |

중기병 `heavy_cavalry`는 보병 계열이 아니지만 `counter_ward`(반격피해감소)를 무도가로 옮기므로 Pool이 3에서 2로 줄어든다. 대체 특성은 미정(7절).

### 4-2. 무도가 계열

| 병종 | 등급 | 현재 Pool | 제안 Pool | 기존/신규 | 비고 (Plan) |
|---|---|---|---|---|---|
| 무도가 `brawler` | 기본 | spirit_prep, counter_art, fire (3) | spirit_prep, counter_art, relentless (3) | 기존 | `relentless` = 연전(반드시 연속 공격 성공). `fire`는 빼서 경보병과 겹치지 않게 한다. Plan 10 counter_art 유지 |
| 권사 `pugilist` | 중급 | deputy_appoint, diagonal (2) | deputy_appoint, lone_warrior, grit, counter_ward, drain_weak (5) | 기존 + 신규 1 | `lone_warrior` = 일기당천, `grit` = 근성(병력 50% 미만에서 공격·방어 증가), `counter_ward` = 반격피해감소(중기병에서 옮김), `drain_weak` = 약한 심공. Plan 20 diagonal → grit으로 교체 |
| 권성 `fist_saint` | 상급 | spirit_spread, palm_wind, fire_boost (3) | spirit_spread, retaliate, deadly, drain_weak, lone_warrior (5) | 기존 + 신규 1 | `retaliate` = 응징(사거리와 관계없이 모든 물리 공격에 반격, 책략 제외), `deadly` = 필살. `palm_wind` 제거(장창과 모양 겹침), `fire_boost` 제거(강화 3종은 군사). Plan 30 fire, 50 palm_wind, 60 fire_boost 제거 |

권성의 `retaliate`는 반격강화 대신 쓰는 안이다. 번궁의 고유 특성과 충돌하지 않는다(6절).

### 4-3. 집계

- 보병 계열: 경보병 3 · 중보병 5 · 장창병 5 · 근위병 5 · 전차대 5. 중급이 2개뿐이라 상급 4 조정은 필요 없다.
- 무도가 계열: 무도가 3 · 권사 5 · 권성 5.
- 모든 Pool 특성 키는 `traits.json`에 있다(확인 완료). 고유 특성(진지고수·호위·찔러공격·돌파·기습·일당백·의술)과 압박은 어느 Pool에도 없다.

## 5. 계열 분리 확인

| 겹칠 수 있던 항목 | 현재 위치 | 제안 후 |
|---|---|---|
| diagonal (대각공격) | 경보병·중보병·장창병·근위병·권사·흉적·친위대 | 권사 Pool·Plan 20에서 제거. 보병 전용 |
| fire (화계) | 경보병·무도가·서생·군사 등 | 무도가에서 제거. 경보병만 보유 |
| long_spear vs palm_wind | 장창병 Pool / 권성 Pool·Plan 50 | 장풍 제거. 장창은 보병 |
| fire_boost | 권성 Pool·Plan 60, 책사, 도독 | 권성에서 제거 (군사 전용 원칙) |
| counter_art (반격술) | 무도가, 적병, 황건적 | 무도가 유지. 적병·황건적은 별도 병종군이라 이번 범위 밖이며 그대로 둔다 |

## 6. 권성·번궁 문제

- 번궁(`fangong`, 권성 병종)은 현재 `footwork`(보법)과 `counter_boost`(반격강화)를 고유로 가진다(`characters_story.json`, Traits와 Learn 모두).
- **반격강화를 권성 Pool에 넣으면 번궁의 `counter_boost`가 더는 고유가 아니므로 번궁의 고유 특성 하나를 교체해야 한다.** 교체 대상과 내용은 사용자 결정 전까지 정하지 않는다.
- 이 초안은 반격강화를 쓰지 않고 `retaliate`(응징)로 같은 역할을 맡긴다. 그래서 번궁의 교체는 필요하지 않다.
- 응징(`retaliate`)은 현재 문추(`wenchou`)와 태사자(`taishici`)가 보유한다. 이 둘이 고유인지 확인이 필요하다. 사용자 지시 목록에 없으므로 고유로 보지 않았다.

## 7. 열린 질문 (사용자 확인 필요)

1. **장풍 처리**: 권성에서 빼고 장창을 보병에 남기는 안(제안)과 반대로 장풍을 살리고 장창을 빼는 안 중 어느 쪽인가. 둘 다 상하좌우 2칸 근접 십자라 동시 보유는 겹친다.
2. **권성의 반격**: 응징(제안, 번궁 영향 없음) vs 반격강화(번궁 고유 교체 필요, 미결정).
3. **심공 10%**: 새 키 `drain_weak`(제안) vs 기존 `drain` 값 자체를 10%로 낮추기(테스트 수정 필요). 흡수 적용 시점(어느 타격에 붙는지)도 확인해야 한다.
4. **응급처치 위치**: 중보병(제안) vs 권사. 응급처치는 자가 회복이라 무도가 "기로 버티는" 이미지와도 맞는다. 취향 확인 필요.
5. **중기병**: `counter_ward`를 권사로 옮기면 중기병 Pool이 2개가 된다. 대체 특성을 정해야 한다(예: `footwork`·`vanguard`는 기존 설계에 있었음). 기병 계열은 이번 지시 범위 밖이라 보류.
6. **흉적 `anti_cavalry`**: 전차대(제안)와 흉적 제안이 같은 특성을 쓴다. 흉적 쪽을 다른 특성으로 바꿀지 정해야 한다. 흉적은 이번 계열 밖이다.
7. **전차대 Plan 60**: `pierce` 제거 후 빈 자리에 무엇을 넣을지(제안 없음).
8. **권성 `deadly`(필살, 비용 1500)**: 상급 무도가에 두기에 비싼지 확인. 밸런스 미검증.
9. **중보병 `block_up`**: 중기병 기존 제안의 `footwork`(명중 −15%p)와 효과가 겹친다. 둘 다 "적 명중률 감소"라 같은 역할을 두 계열이 나눠 가지는 것이 맞는지 확인.
10. **세객·종횡가·군사·도독**: 이번 문서 범위가 아니다. 군사 Pool의 강화 3종 이동과 종횡가 Plan은 기존 설계 그대로 둔다.

## 8. 적용 시 작업 목록 (적용하지 않음)

- `ver5/assets/data/classes.json`: 위 Pool·Plan 변경 (경보병·중보병·장창병·근위병·전차대·무도가·권사·권성 + 중기병 counter_ward 제거).
- `ver5/assets/data/traits.json`: `drain_weak` 추가 (Kind `talent`, 흡수 비율 0.1).
- `ver5/internal/content/class_pools_test.go`: `classPools` 맵이 Pool을 그대로 복제하고 있으므로 갱신이 필요하다. 이 테스트는 `first_aid`·`brace_prep` 같은 특성이 스킬을 여는지도 확인한다(`first_aid`는 스킬 `first_aid`를 연다).
- 고유 특성 위치 확인: 찔러공격·돌파·기습·일당백·의술·진지고수·호위·압박이 병종 Pool·Plan에 남지 않았는지 검색으로 확인.

## 9. 미검증

- 게임 실행, 전투 캡처, `cmd/_balance` 시뮬레이션은 하지 않았다. 모든 수치와 밸런스는 데이터 읽기에 근거한다.
- `drain_weak`이 `FxLifeSteal` 효과로 데이터만으로 동작하는지 코드 경로를 따라가지 않았다.
- `retaliate`(응징)가 반격술 태세와 함께일 때 중첩되는지, `counter_ward`와 `retaliate`가 같은 공격을 두 번 막지 않는지 미확인.
- 제안 특성의 비용(1000·1500 등)과 강도는 검토하지 않았다. 특히 `deadly`, `crit_boost`(전차대)는 시험이 필요하다.
- 흉적·적병·황건적·중기병 등 이번 범위 밖 병종의 Plan은 전부 확인하지 않았다.
- 기존 이전 초안(`result/20261124/infantry-martial-traits-redesign/design.md`)은 이 문서로 덮어썼다. 원본은 scratchpad에 보관했다.
