# 확정 병종·장수 특성 변경 적용 결과 (2026-11-24)

요청: `process/class-trait-implementation-20261124/class-trait-implementation-20261124.md`
근거: `result/20261124/implementation-decisions-summary/summary.md`, `class-trait-revision-notes/notes.md`, `horse-archer-range-houyuan/notes.md`, `horse-archer-close-range/design.md`, `firecraft-officer-decisions/result.md`, `retaliation-damage-condition/result.md`, `promotion-level-verification/result.md`

## 작업 방식

워크트리를 다른 세션이 여러 파일로 함께 쓰고 있다. `classes.json`, `characters_story.json`, `traits.json`, `combat.go`에는 미커밋 변경이 이미 들어 있었다. 그중 미확정 항목이 섞여 있어, 두 데이터 파일은 HEAD(8c5fa25)에 확정 항목만 얹은 내용을 임시 인덱스로 만들어 커밋했다(9a3bd59). 워크트리의 다른 세션 변경은 건드리지 않았다. 워크트리 파일은 그대로 남아 있다.

## 항목별 결과

| # | 항목 | 상태 | 위치 / 근거 | 검증 |
|---|---|---|---|---|
| 1 | 궁기병 사거리 1~3 (`MinRange` 2→1, Range 3 유지) | 적용·커밋 | `ver5/assets/data/classes.json` horse_archer `MinRange` (9a3bd59) | 데이터 확인. 요청 원문 확정 |
| 2 | 궁기병 Pool: 저격 제외, 저지 사격(`suppressing_shot`) 추가 | 적용·커밋 | classes.json horse_archer `Pool` = deputy_appoint, aim_prep, suppressing_shot | 파이썬 점검, 테스트 기대값(`classPools`)과 일치 |
| 3 | 원시·저격을 병종 특성으로 주지 않음 | 적용(궁기병 범위) | 궁기병 Pool에서 snipe_prep 제거. 원시(far_sight)는 궁기병에 없음 | 동일 |
| 4 | 이탈사격 자리에 저지 사격 | 적용 | 데이터에 이탈사격 키 없음(설계 문서 5절). 저지 사격을 궁기병 Pool에 넣음 | 동일 |
| 5 | 책사 격파 제외 (`crush`) | 적용·커밋 | classes.json staff_officer `Pool` | 파이썬: 어느 학습 풀·계획에도 `crush` 없음 |
| 6 | 격파 → 정욱 고유 | 적용·커밋 | characters_story.json chengyu `Traits` += crush (9a3bd59) | 데이터 확인 |
| 7 | 격파 → 법정 고유 | 미적용 | 법정 인물 항목이 characters*.json에 없음. 인물 추가·출전은 미확정(summary 10번) | 해당 없음 |
| 8 | 화계강화·수계강화·지계강화 책사 → 군사 | 적용·커밋 | 책사 Pool·Plan(33/36/39)에서 제거, 군사 Pool 추가 (classes.json) | 파이썬 점검 |
| 9 | 군사 Pool 4개 (지형 조성 + 세 강화) | 적용·커밋 | strategist `Pool` | 동일. Plan 레벨은 HEAD 그대로 |
| 10 | 세객 수계 제외 | 적용·커밋 | orator `Pool`·`Plan`(22) 제거. 종횡가 Plan 22 수계도 상속분으로 제거 | 동일 |
| 11 | 혼란 → 종횡가 | 적용·커밋 | orator Pool·Plan(15) 제거, diplomat Pool에 stratagem 추가 (Plan@15는 HEAD 그대로) | 동일 |
| 12 | 파진 고유 (`break_formation`) | 적용·커밋 | orator Pool·Plan(28), diplomat Plan(28) 제거. 보유자 전풍·서서·단복·괴기는 그대로 | 파이썬: 학습 풀·계획에 없음 |
| 13 | 승급 레벨 20·40 | 확인만 (변경 없음) | game.json `PromoteLevels` [20,40], 검사 formation.go:191-193 | 기존 검증 문서와 일치 |
| 14 | 보복 중첩: 실제 HP 피해 > 0일 때만 중첩 (빗나감·0 피해 제외) | 적용(워크트리만, 미커밋) | `ver5/internal/core/combat.go:337` `if hit && n > 0 && v.HP > 0 && ...` | `TestReprisalAccumulatesOnlyDamageAndExpires` PASS. 빗나감 경로는 테스트 없음 |
| 15 | 허사 심모→화공 | 미적용 | 결과 문서가 "게임 적용 보류"로 기록. 화계 선행·NPC 계획 검증 필요 | 워크트리에는 다른 세션이 이미 바꾼 상태(미커밋) |
| 16 | 황보숭 화신→화공 | 미적용 | 위와 같음 | 위와 같음 |
| 17 | 번궁 반격강화→화공 | 미적용 | 요청 목록 밖. 학습 항목 변경이라 보류 | 워크트리에 미커밋 변경 존재 |
| 18 | 하후연 저지 상위 특성 (−2, 합계 −4) | 미적용 | 특성 이름·비용 미정, 저지 사격 선행 흐름 미정(설계 4-4절 a/b/c). 워크트리에는 `suppression_mastery`(비용 1000)가 추가되고 하후연의 `suppressing_shot`이 빠진 미커밋 상태 | 상충 가능성 있음 |
| 19 | 노기병 사거리·계획 변경 (MinRange 1, L40 저지 사격, L50 저격 제거) | 미적용 | 노기병은 "별도 결정"(houyuan 노트 2절, close-range 설계 3절). HEAD는 MinRange 2, 저격 L50 | 워크트리에 미커밋 변경 존재 |
| 20 | 종횡가에 장구·약점공략 이동, 세객 장구·약점공략 제외 | 미적용 | 사용자 논의 기록(revision-notes)에는 혼란만 이동. 장구·약점공략 고유화는 미정. summary 3번은 이동을 적었으나 근거 불충분 | HEAD는 세객 Pool에 남음. 워크트리는 이동된 미커밋 상태 |
| 21 | 군사 화계·수계·지계강화 계획 레벨 조정(40/45/50), 종횡가 계획 레벨 조정 | 미적용 | 레벨 결정 기록 없음 | 워크트리에 미커밋 변경 존재 |
| 22 | 세객 '방진/수성/상급 능력치/반계 시전/이야기 막 명칭' (process 요청 본문의 나머지) | 미적용 | 요청 목록 밖. summary상 수성 부여 대상·방진 합산 방식·반계 발동 조건 미정 | 해당 없음 |
| 23 | 하후연 외 저지 상위 특성 traits.json 항목 `suppression_mastery` | 미적용(워크트리 미커밋) | 18번과 같음 | 커밋 안 함 |
| 24 | 격파 제외 후 책사 Plan 정리 | 적용 | staff_officer Plan에 crush는 없었음. 화계·수계·지계강화 Plan(33/36/39) 제거 | 파이썬 |
| 25 | 번궁 Pool에 counter_boost 추가(워크트리 classes.json ~1539행) | 미적용 | 확정 기록 없음 | 워크트리 미커밋 |

## 커밋

- `9a3bd5962da6cc3be952287c03592385aa87e0ef` (`feat(class): apply confirmed class trait changes`), 부모 `8c5fa25`
- 변경 파일: `ver5/assets/data/classes.json`(55줄), `ver5/assets/data/characters_story.json`(정욱 격파 1항목)
- `ver5/internal/core/combat.go` 보복 조건(14번)은 커밋하지 않았다. HEAD에는 보복 구현 코드가 아예 없고, 워크트리의 보복 구현은 다른 세션의 미커밋 변경이라 파일 단위 커밋이 그 변경까지 포함하게 된다.

## 빌드·vet

- 워크트리(현재 상태) `go build ./...`: 종료 코드 0.
- 워크트리 `go vet ./internal/content/ ./internal/core/`: 종료 코드 0, 출력 없음.
- 커밋 9a3bd59만 체크아웃한 임시 워크트리에서 `go vet ./internal/content/ ./internal/core/`: 출력 없음.
- 같은 임시 워크트리의 `go build ./...`: `internal/gui`에서 실패(`cutscene.go` `NoFocus`, `deployment.go` `Excluded`, `explore.go` `def.Cut` 등). HEAD가 아직 커밋되지 않은 다른 세션 코드의 심볼을 참조하기 때문이며 이번 변경과 무관하다.

## 테스트 (`go test -p 1 -count=1 -run ...`)

- `./internal/content/` `TestClassPoolsFollowTraitDesign`, `TestEveryLearnableSkillTraitOpensASkill`: FAIL. 둘 다 데이터 로드 단계에서 `explore25_back`/`explore27v_travel` 등 명명된 인물 "말하는 것 없음" 검사에 걸린다. 같은 테스트를 HEAD(8c5fa25) 임시 워크트리에서 돌려도 같은 실패가 나므로 이번 변경과 무관하다. 이 때문에 풀 검사 본문은 실행되지 않았고, 풀 검증은 파이썬으로 대신 했다.
- `./internal/core/` `TestOfficerSpellSpecialties`, `TestOfficerReformBatch`, `TestOfficerReformTraitsInBattle`: FAIL. 같은 종류의 로드 실패. HEAD 임시 워크트리에서도 동일하게 실패.
- `./internal/core/` `TestReprisalAccumulatesOnlyDamageAndExpires`: PASS.
- `./internal/core/` `TestFormationAndDeputyStatIncrease`: 실행 결과 ok.
- 파이썬 점검(커밋 데이터 기준): 학습 풀 항목이 모두 `traits.json`에 존재. `crush`, `break_formation`은 어떤 학습 풀·계획에도 없음. `water`는 책사·방술사 등에 남음. `stratagem`은 종횡가 풀에 있고 세객에는 없음. `suppressing_shot`은 궁기병 풀에만 있음.
- 참고: `class_pools_test.go`의 `classPools` 기대값은 HEAD부터 heavy_infantry, guards, royal_guard, horse_crossbow, repeating_crossbow, catapult에서 데이터와 어긋난다. 이번 변경과 무관하다.

## 검증하지 못한 것

- 게임 실행·전투 시뮬레이션·캡처 확인은 하지 않았다.
- 보복 빗나감 경로는 테스트가 없다.
- 학습 풀 검사 테스트는 HEAD 기준 로드 실패 때문에 돌지 않았다.
- 워크트리의 다른 세션 미커밋 변경(허사·황보숭·번궁·하후연·노기병·종횡가 장구·약점공략 이동 등)은 확정 항목이 아니므로 건드리지 않았다. 사용자 판단이 필요하다.
