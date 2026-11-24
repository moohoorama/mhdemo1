# 승급 레벨 검증 결과 (2026-11-24)

## 결론

- 실제 게임 규칙의 승급 레벨은 **중급 20, 상급 40**이다. 사용자 말(20·40)이 맞다.
- 이전 주장(15·30)은 틀렸다. 근거는 `docs/logic-growth.md`의 낡은 문장이었고, 그 값은 2026-11-14에 이미 20·40으로 바뀌었다(`docs/history.md:1078`).
- 데이터와 코드는 서로 일치한다.

## 근거

### 1. 값의 출처: `ver5/assets/data/game.json`

- `game.json:86-89`: `"PromoteLevels": [20, 40]`. 인덱스 0은 Tier 0→1(중급), 인덱스 1은 Tier 1→2(상급) 승급 레벨이다.
- `game.json:80`: `"PointsPerLevel": 100`.
- 로드: `internal/content/content.go:61` `PromoteLevels []int // minimum level to promote from tier i`. 검증: `internal/content/validate.go:20`(비어 있으면 실패).

### 2. 강제 지점 (코드)

- 인수로 승급 검사: `internal/core/formation.go:191-193`
  `required := e.data.Rules.PromoteLevels[cl.Tier]`, `o.Level < required`면 `InsufficientLevel`.
- 인수 사용 가능 여부 표시: `internal/gui/insu.go:85`에서 `PromoteLevels[it.Tier-1]`을 화면 문구 "승급 레벨 N 이상"으로 출력(데이터 기반, 하드코딩 없음).
- 승급 전 상태 검증(세이브 로드): `internal/core/api.go:490-491`, `o.Level < R.PromoteLevels[t-1]`이면 `CorruptSave`.
- 레벨에 따른 자동 병종 결정: `internal/content/spawn.go:44-66` `PromotedAt`(`PromoteLevels[t-1]`, `PromoteLevels[cl.Tier]` 사용). 적·동맹·합류 장수가 이 값으로 병종을 정한다.
- 합류 장수: `internal/core/growth.go:265-272` `joinAt` → `promotedAt`.
- 초기 특성치: `internal/core/engine.go:184` `Points: PointsPerLevel * (level - 1)` (20레벨 1,900, 40레벨 3,900). 이는 레벨 공식이며 승급 레벨 자체는 아니다.

### 3. 인수 Tier와 병종 Tier

- `assets/data/items.json` 승급 인수: `insu_mid`(중급 인수) `Tier: 1`, `insu_high`(상급 인수) `Tier: 2`. 인수 Tier = 목표 병종 Tier다. 인수는 `Tier == cl.Tier + 1`을 요구한다(`formation.go:188`).
- `assets/data/classes.json`: Tier 0 = 경보병·경기병·궁병·민중 등 1단계, Tier 1 = 중보병·중기병·장창병·세객 등 중급, Tier 2 = 근위병·친위대·군사 등 상급.

### 4. 테스트

- `internal/core/away_test.go:55-62`: `R.PromoteLevels[0]`, `[1]` 기준 검사.
- `internal/core/campaign_content_test.go:441, 518-519`: `PromoteLevels[0]`, `[1]` 기준 검사. 값을 숫자로 박지 않는다.
- `internal/gui/uidemo_insu.go:95-96`: 데모도 `PromoteLevels[t]` 사용.

## 불일치 항목 (문서만 낡음, 코드·데이터 정상)

- `docs/logic-growth.md:63`: "0단계는 15, 1단계는 30레벨이다." 현재 값과 다르다. 이 문서를 고쳐야 한다. (수정은 하지 않았다, VERIFY ONLY.)
- `docs/history.md:1073`(look_test의 레벨 15 → PromoteLevels[0])·`:1078`(15·30 → 20·40 변경 기록)은 과거 기록이므로 정상이다.
- `docs/officer-trait-reform-preview.html:32`는 "평시 승급 기준은 Lv20·40"으로 맞다.
- `result/20261124/promotion-level-verification/summary.md`의 내용(20·40, 1,900·3,900)은 이번 확인과 일치한다.

## 미확인 사항

- 게임 실행 캡처로 실제 승급 창을 띄워 보지는 않았다. 판단은 데이터·코드 정적 확인에 근거한다.
- 적 AI의 승급 레벨 적용(`cmd/_traitcount`, `cmd/_campaign`)은 같은 `PromoteLevels`를 쓰는 것만 확인했고, 실행 결과는 보지 않았다.
- `internal/gui/uidemo_zz_tmpjulu.go:11`의 `Level = 15`는 임시 데모 코드로, 승급 규칙과 무관하다.
