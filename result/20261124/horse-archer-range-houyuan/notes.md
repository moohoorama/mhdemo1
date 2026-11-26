# 궁기병 1~3 사거리와 하후연 저지 상위 특성 설계 기록

작성일 2026-11-24. 기록만 했다. 게임 코드·데이터(`ver5/assets/data/*.json`, `ver5/internal/`)는 바꾸지 않았다.
요청: `process/horse-archer-range-houyuan-20261124/horse-archer-range-houyuan-20261124.md`

## 1. 요약

- 궁기병 사거리: `classes.json`의 `horse_archer`에서 `MinRange`를 2에서 1로 바꾸면 사각 없는 1~3칸(맨해튼 거리)이 된다. `Range 3`은 그대로 둔다. 병법 Min은 모두 1이라 막는 것이 없다.
- 하후연: 저지 사격(`suppressing_shot`, `ranged_slow: 2`)과 새 고유 특성(`ranged_slow: 2`)이 같은 키로 **더해져** 한 번의 적용으로 대상 다음 차례 이동 **−4**가 된다. 고유 특성 값이 −2여야 한다. −4 값이면 −6이 된다.
- 둘 다 데이터·코드 미반영 설계안이다.

## 2. 궁기병 사거리 (1~3칸, 사각 없음)

### 2-1. 현재 값 (확인함)

- `ver5/assets/data/classes.json` `horse_archer`(궁기병, Tier 1): Attack `ranged`, Weapon `bow`, Move 5, **Range 3, MinRange 2**. Bonus `[13, 7, 5, 11, 9, 15]`.
- 같은 등급 궁병 `archer`, 노병 `crossbow`, 노기병 `horse_crossbow`, 연노병 `repeating_crossbow`, 발석거 `catapult`도 모두 Range 3, MinRange 2다. 원거리 병종 전부가 1칸 사각을 가진다.

### 2-2. 판정 방식 (코드 확인함)

- `internal/core/rules.go:361-370` `rangedBand(u, s)`: 원거리 병종이면 `lo = max(1, s.Min, cl.MinRange)`, `hi = cl.Range + 보정`. 기마 사격(기사)은 `lo = 2`.
- `internal/core/rules.go:381-383` `weaponReach`: 거리 `d = |dx| + |dy|`가 `lo ≤ d ≤ hi`이면 원거리 사격. 맨해튼 마름모 고리다. 대각 규칙은 근접에만 있다.
- `skills.json`의 `Min`은 40개 전부 1이다(`grep '"Min"'` 결과 `"Min": 1`뿐). 따라서 병법이 하한을 올리지 않는다. 하한을 정하는 것은 병종 `MinRange`뿐이다.
- `MinRange`가 없거나 0이면 `max(1, …)`에 의해 하한은 1이다. 즉 1~3이 기본값이 된다. 명시적으로 `1`을 적는 것과 같다. 설계 기록에는 `"MinRange": 1`을 명시하는 쪽을 추천한다. 읽는 사람이 의도를 알 수 있다.
- 장창(`pikeman`)은 원거리가 아니다. 근접 직선 판정(`meleeReach`, 장창이 `FxLongReach`)이며 `Range 1`에 두 칸 직선이다. 궁기병과 판정 경로가 다르다.

### 2-3. 제안 값

| 항목 | 현재 | 제안 |
|---|---|---|
| `horse_archer` `Range` | 3 | 3 (유지) |
| `horse_archer` `MinRange` | 2 | **1** |
| 사격 범위 | 맨해튼 2~3 | 맨해튼 1~3 (거리 1: 4칸, 거리 2: 8칸, 거리 3: 12칸) |

- 노기병(`horse_crossbow`)도 같은 방식으로 1로 낮출지는 별도 결정이다. 이 기록은 궁기병만 다룬다. 요청은 궁기병과 노기병을 함께 적었지만(`horse-archer-close-range` 설계와 `class-trait-system-design` 4절의 "궁기병·노기병"), 요청 본문은 궁기병만 명시한다.

### 2-4. 1칸 사격을 바꿀 때 함께 볼 것 (미검증)

- 반격: `combat.go:523-528` `counterBack`과 `api.go:866-876` `counterUnits`는 `inRange(v, u, basic)`를 쓴다. 인접한 근접 병종은 궁기병 공격을 받고 반격할 수 있다. 인접한 **궁병·노병 계열**(MinRange 2)은 1칸에서 반격하지 못한다. 한쪽만 맞는 교전이 생긴다.
- `rangedAttack`(`combat.go:133-144`)은 원거리 병종이면 `true`다. 1칸 사격도 원거리 물리로 처리되어 저지 사격 조건(직접 원거리 물리)을 만족한다.
- 표시: `internal/gui/officer_cards.go:39-40`은 `MinRange > 1`일 때만 "최소~최대" 형식으로 보여 준다. MinRange를 1로 낮추면 카드에 "3"만 나온다. "1~3"으로 보이게 하려면 GUI 조건을 바꿔야 한다. 이번 설계 범위 밖이다.
- 문서: `ver5/docs/logic-combat.md:142`는 "`MinRange`~`Range`의 마름모 고리"라고 적는다. 값이 바뀌면 문서도 맞출지 결정한다.
- 테스트: `internal/testdata/mini/classes.json`에 MinRange가 있다. 궁기병 값을 2로 가정하는 테스트가 있는지 확인이 필요하다(이번에 검색하지 않았다).
- AI 사격 위치 선택과 ZOC, 원거리 면역, 화시·연발 연계는 확인하지 않았다.

### 2-5. 요청과 현재 데이터의 충돌

- 요청 원문(close-range 요청)은 궁기병의 "사거리가 짧다"고 한다. 현재 `Range`는 3으로 궁병과 같고, 이번 요청은 1~3이다. 최대 사거리는 줄지 않는다. "짧은 사거리"는 최대가 아니라 **인접 사격과 사각 제거**로 읽어야 하는지 확인이 필요하다.
- 요청은 궁기병의 "순발·사기가 낮다"고 한다. 현재 순발 11, 사기 9다. 궁병은 순발 10·사기 9, 경기병은 순발 9·사기 8이다. 궁기병이 궁병보다 순발이 높다. 낮추려면 Bonus를 바꿔야 하며 이번에는 정하지 않았다.

## 3. 하후연 저지 상위 특성

### 3-1. 현재 데이터 (확인함)

- `ver5/assets/data/characters_story.json` `xiahouyuan`(하후연): 직업 `horse_crossbow`(노기병, Tier 2, 궁기병 아님). Level 8.
- `Traits`: `["suppressing_shot", "forced_march", "charge_aim"]` (저지 사격, 추행진, 기습).
- `Learn`: `[]`. `Equipment`: `short_bow`, `leather_armor`, `war_horse`.
- 이동 −4 특성은 없다. 저지 사격은 **선천(Traits)** 보유다. 학습으로 얻은 것이 아니다.
- `ver5/assets/data/traits.json` `suppressing_shot`: 이름 "저지 사격", Desc "직접 원거리 물리 첫 타격이 명중하면 대상의 다음 차례 이동력 −2.", Kind `talent`, Tone `attack`, Cost 1000, `Effects: {"ranged_slow": 2}`. Owner 없음, Requires 없음.

### 3-2. 효과 합산 경로 (코드 확인함)

1. 발동: `internal/core/combat.go:436-443`. 물리 공격의 `i == 0`(첫 타), `v == aim`(주 대상), `hit`, `n > 0`, 물리 종류이고 `rangedAttack`이 참일 때만 `k := int(e.fx(u, FxRangedSlow))`를 읽어 `k > 0`이면 `applyStatus(v, "speed", 1, -k)`를 한 번 호출한다.
2. 합산: `engine.go:360-367` `fx(u, key)`는 `fxOf(e.char(u.Character), key)`에 상태 효과를 더한다. `engine.go:369-376` `fxOf`는 `traits(c)`를 순회하며 각 특성의 `Effects[key]`를 **더한다**.
3. 중복 제거: `engine.go:340-357` `traits(c)`는 map(set)에 넣는다. 같은 **특성 id**는 한 번만 센다. 서로 다른 id는 더해진다. 주장 특성 + 장비 특성 + 부관의 innate·Learned가 모두 포함된다(부관 포함, 352-355행).
4. 적용: `combat.go:252-268` `applyStatus`는 같은 상태가 있으면 **절댓값이 큰 쪽**을 남긴다. 합산은 `applyStatus`가 아니라 `fxOf`에서 끝난다. 따라서 한 번의 공격에서 `-k`가 한 번 들어간다.
5. 이동 계산: `rules.go:186` `cl.Move + u.Status["speed"].Value + fx(FxMove)`. 상태 `speed`의 값이 이동력에 그대로 더해진다. `speed`는 `content.Harmful`에 없어 면역(`statusImmune`)에 막히지 않는다.

### 3-3. 산술

| 하후연 특성 구성 | `ranged_slow` 합 | 적용 `speed` 값 | 대상 다음 차례 이동 |
|---|---|---|---|
| 저지 사격만 (현재) | 2 | −2 | −2 |
| 저지 사격 + 고유 −2 (**제안**) | 2 + 2 = 4 | −4 | **−4** |
| 저지 사격 + 고유 −4 (close-range 초안) | 2 + 4 = 6 | −6 | −6 |
| 고유 −2만 (저지 사격 제거) | 2 | −2 | −2 |

- 결론: 요청의 −4를 맞추려면 고유 특성의 `ranged_slow`를 **2**로 둔다.
- `applyStatus`는 더하지 않는다. 한 번의 공격에서 `−4`가 한 번 들어가고, 같은 대상에게 다른 출처의 `speed`가 있으면 절댓값이 큰 쪽이 남는다. 예: 대상이 이미 `speed −5`이면 −5가 남는다.
- 주의: `close-range` 설계(`result/20261124/horse-archer-close-range/design.md` 4-4절)는 "`applyStatus`가 절댓값이 큰 쪽을 남기므로 −6이 그대로 남는다"고 적었다. 결과(−6)는 맞지만 이유는 `fxOf` 합산이다. `applyStatus`의 큰 값 규칙은 합산이 아니다. 그 문서의 이유 문장은 정정이 필요하다.
- 주의: `traits(c)`는 **부관** 특성도 센다. 하후연의 부관이 다른 id의 `ranged_slow` 특성을 가지면 합계가 더 커진다. 같은 id(저지 사격)이면 중복 제거되어 한 번만 센다. 장비 특성(`Equipment`)도 같은 방식으로 더해진다.

### 3-4. 선행(Requires)과 학습 경로 (코드 확인함)

- `internal/core/engine.go:390-420` `learnable(c)`: 학습 후보 = (`Learn` 목록 + 병종 Pool/Plan + 장비 `Learn`) 중 **아직 보유하지 않고**, `Requires`의 모든 항목이 `owned`에 있는 것. `owned = e.traits(c)`이므로 **선천 특성도 선행으로 인정된다**. 하후연은 `suppressing_shot`을 선천으로 가지므로 선행 조건은 바로 충족된다.
- 즉 "저지 사격을 선행 학습"하려면 현재 상태 그대로면 된다. 다만 "학습"이라는 흐름을 만들려면 `suppressing_shot`을 `Traits`에서 빼고 `Learn`에 넣어야 한다. 이는 하후연의 출발 상태를 바꾼다. 결정이 필요하다.
- 상위 특성은 하후연의 `Learn`에 넣어야 학습 후보가 된다. 현재 `Learn`은 비어 있다. 선천 `Traits`에 넣으면 학습 흐름 없이 처음부터 가진다.
- 검증: `internal/content/validate.go:67-70`은 `Requires`가 존재하는 특성인지와 순환만 본다. `spawn.go:286-296`의 선행 검사는 `ValidateSpawn`(스테이지 배치)에서만 돈다. 스토리 인물의 선천 특성에는 선행 검사가 없다. 이 때문에 `fire_god`(화신, `Requires: fire`)을 가진 황보숭·주유가 선행 없이 로드된다. 선천 특성에는 선행이 강제되지 않는다는 뜻이다.

### 3-5. 제안 특성 (미확정 값은 표시)

```
"pursuit_block": {
  "Name": "추격 봉쇄",
  "Desc": "직접 원거리 물리 첫 타격이 명중하면 대상의 다음 차례 이동력 −2. 저지 사격과 함께 있으면 합산해 −4가 된다.",
  "Kind": "talent",
  "Tone": "attack",
  "Cost": 1000,
  "Owner": "xiahouyuan",
  "Requires": ["suppressing_shot"],
  "Effects": {"ranged_slow": 2}
}
```

- 키 `pursuit_block`, 이름 "추격 봉쇄"는 제안이다. 쉬운 이름 원칙(`trait-names-plain`)에 따랐다. 최종 이름은 미정이다.
- `Cost`: 1000은 저지 사격과 같은 등급의 안이다. 미정.
- `Owner: "xiahouyuan"`: 장수 고유 특성이므로 병종 Pool에 넣지 않는다(`trait-owner-reservations`). `validate.go:56`이 Owner가 존재하는 인물인지 본다.
- `Requires: ["suppressing_shot"]`: 학습 전에 저지 사격 보유를 요구한다.
- `Effects: {"ranged_slow": 2}`: 합계 −4의 근거. 값을 4로 적으면 −6이 된다.
- 특성 설명문의 "−4"는 저지 사격과 합쳐진 값이다. 특성 하나만의 값은 −2다. 설명문 표현은 최종 문구에서 다시 정한다.

### 3-6. 하후연 데이터 반영 시 확인 (미반영)

- `traits.json`에 특성 추가, `characters_story.json` `xiahouyuan.Learn`에 추가(또는 `Traits` 추가). 둘 중 하나.
- 학습 흐름을 원하면 `suppressing_shot`을 `Traits`에서 `Learn`으로 옮길지 결정.
- 하후연 NPC 계획과 획득 시점(`LearnAfter`), 획득 전 상태.
- 부관으로 같은 `ranged_slow` 특성을 가진 장수가 붙는 경우의 합계.
- 저지 사격의 전투 검증: `cmd/_balance` 또는 게임 캡처에서 −4 이동이 다음 차례에 실제로 적용되는지. 미검증.

## 4. 결정됨과 미정 (이 기록 기준)

| 항목 | 상태 | 근거 |
|---|---|---|
| 궁기병 사각 없는 1~3칸, 장창과 다름 | 요청 기준 결정 | `horse-archer-range-houyuan` 요청 |
| `MinRange` 2→1 (값 제안) | 제안 (데이터 미반영) | 2-3절 |
| 궁기병 순발·사기 낮추기 | 미정 | 2-5절 충돌 |
| 궁기병 "짧은 사거리"(close-range 요청)와 1~3 충돌 | 미정 (확인 필요) | 2-5절 |
| 하후연 저지 사격 선행 후 상위 특성, 두 효과 합쳐 이동 −4 | 요청 기준 결정 | `horse-archer-range-houyuan` 요청 |
| 고유 특성 `ranged_slow` 값 2 | 산술상 필요한 값 (제안) | 3-3절 |
| 고유 특성 키·이름·비용·Learn/Traits 배치 | 미정 | 3-5, 3-6절 |
| `suppressing_shot`을 `Traits`에서 `Learn`으로 옮기기 | 미정 | 3-4절 |

## 5. 미검증

- 게임 실행, `cmd/_balance` 시뮬레이션, 캡처를 하지 않았다. 수치는 데이터 읽기와 손계산이다.
- 1칸 사격의 AI 선택, ZOC, 원거리 면역, 화시·연발과의 상호작용은 확인하지 않았다.
- `rules.go`의 나머지 곱셈 경로와 `combat.go:441` 주변 세부(저지 사격이 근접 공격에서 발동하지 않는다는 조건)는 `rangedAttack`의 판정으로만 확인했다.
- `horse_archer`의 MinRange 2를 가정하는 테스트(`*_test.go`)는 검색하지 않았다.
- 노기병의 MinRange 변경 여부는 결정하지 않았다.
