# 이야기 대단락 명칭: 장 → 막

사용자 결정: 난세도래·주유천하처럼 본편의 큰 이야기 단락은 1장·2장·3장 대신 1막·2막·3막으로 표시한다. 스테이지·시나리오의 개별 장 번호까지 일괄 변경하라는 뜻으로 확대하지 않는다.

현재 게임 종료 카드 확인:
- `ver5/assets/scenario/hero/story16.json`: `1장 난세도래\n끝` → `1막 난세도래\n끝`.
- `ver5/assets/scenario/hero/story27.json`, `story27v.json`: `2장 주유천하\n끝` → `2막 주유천하\n끝`.
- 3막 이름 및 관련 실제 카드 여부는 아직 확인되지 않아 이름을 임의로 정하지 않는다.
- `ver5/docs/fullstory.md`의 대단락 용어·예시 및 관련 코드 주석도 해당 범위에서 일치시킨다. 일반 스테이지 장 번호는 유지한다.

이전 지시에 따라 게임 데이터는 아직 수정하지 않았다. 실제 적용 시 스토리 문서·장면 규칙을 따르고 종료 카드 캡처로 확인한다.

---

## 조사 결과 (2026-11-24, 코드·데이터·본문 미수정)

경로는 모두 `ver5/` 기준. 확인 명령: `grep -rn "난세도래\|주유천하" ver5 --exclude-dir=.git`

### 0. 확인 시점 상태

위 "현재 게임 종료 카드 확인" 항목은 이 조사 시점에 이미 반영되어 있었다. 데이터 3줄과 `docs/fullstory.md:1799-1800`이 `막`이다. 아래 표의 "현재" 열이 그 상태다. 다른 세션이 작업 중일 수 있으니 되돌리지 않았다.

### 1. 큰 단락 이름이 나오는 곳 (23줄, 15개 파일)

| 위치 | 현재 표기 | 성격 | 처리 제안 |
|---|---|---|---|
| assets/scenario/hero/story16.json:436 | 1막 난세도래 끝 | 표시 문자열(`Text`, 장 끝 카드) | 완료 |
| assets/scenario/hero/story27.json:586 | 2막 주유천하 끝 | 표시 문자열 | 완료 |
| assets/scenario/hero/story27v.json:140 | 2막 주유천하 끝 | 표시 문자열 | 완료 |
| docs/fullstory.md:1799-1800 | 1막·2막 | 줄거리 기준 문서(단일 원본) | 완료 |
| docs/data-scenario-node.md:49 | 「1장 난세도래 / 끝」 | 문서 예시 | 막으로 변경 |
| internal/content/content.go:617 | 주석 `("1장 난세도래 / 끝")` | 주석 | 막으로 변경(주석만) |
| internal/gui/dialogue_box.go:287 | 주석 `("1장 난세도래")` | 주석 | 막으로 변경(주석만) |
| output/chapters/ch16.md:204 | 장 끝 카드 1장 난세도래 | 생성물 | 재생성 |
| output/chapters/ch27.md:215, ch27v.md:214 | 장 끝 카드 2장 주유천하 | 생성물 | 재생성 |
| output/scenario.md:3957, 6677, 7308 | 〃 | 생성물 | 재생성 |
| plan/_design_22_27.md:504, 673 | 「2장 주유천하 / 끝」 | 과거 설계 문서 | 두지 않음 |
| plan/_decision_22_27.md:98, 244 | 〃 | 과거 결정 기록 | 두지 않음 |
| plan/synopsis_ch22_end.html:641, 835 | 〃 | 과거 시놉시스 | 두지 않음 |
| docs/history.md:782, 4124, 4170 | 〃 | 작업 이력 | 두지 않음 |

표시 전용: 화면에 뜨는 것은 데이터 `Text` 3줄뿐이다. 문서 예시·주석·생성물은 화면에 나오지 않는다.

### 2. `장` 번호가 붙는 곳: 두 척도가 섞여 있다

- **시나리오 장 번호(01~27)**: 파일 단위. `ChapterNumber(node)`가 챕터 ID 앞자리 숫자를 쓴다(`internal/content/scenario.go:179-210`). 화면에는 `제 16 장`으로 뜬다.
- **큰 단락(1막·2막)**: 장 끝 카드에만 나오고 데이터 키는 없다.

유지할 표시(시나리오 장 번호):
- `internal/gui/hud_corner.go:199-203` `chapterName` → `"%d장"`
- `internal/gui/waypoints.go:81`, `internal/gui/scenario.go:311` → `"제 %d 장"`
- `internal/gui/save_window.go:147` 열 머리 `"장"`
- `tools/scenario_doc/build.py:128` `"# 제 %s 장 (%s)"`, `:145` `"장 끝 카드"`(카드 종류 이름)

식별자(바꾸면 안 됨):
- 챕터 ID: `assets/scenario/scenario.json`의 `"01"`…`"27"`, `"25v"`, `"26v"`, `"27v"`
- 파일명: `storyNN.json`, `stageNN.json`, `explore*.json`, `assets/scenes/chNN.json`
- 노드 ID: `c16-end`, `c27-card`, `c27v-card`. `internal/core/mode_longzhong_test.go:70-92`가 `c27-card`·`c27v-card`를 ID로 검사한다. 노드 ID는 저장 파일에도 들어가므로(`save_window.go:112`) 바꾸면 기존 저장이 깨진다.
- 필드 `Card`(`internal/content/content.go:617`, `validate.go:330`, `"Card": true`)
- Go 식별자와 `Rules.GearTiers`의 `Chapter` 키(`assets/data/game.json:92-`, 값 1·17은 챕터 번호)

### 3. 막 경계 (제안)

근거: 장 끝 카드 위치, `memory/equipment-grade-tiers.md`(1장 = 01~16, 2장 = 17~, 3장 = 적벽 뒤 미구현).
- 1막 난세도래: 챕터 01~16, 끝 카드 `c16-end`(하비 뒤)
- 2막 주유천하: 챕터 17~27과 가상 판 25v~27v, 끝 카드 `c27-card`(연의 화용도 뒤), `c27v-card`(가상 강릉 수비 뒤)
- 3막: 아직 없음. `GearTiers`의 임시 시작은 28장.

2막 시작은 카드로 나오지 않는다(`fullstory.md`가 2막 제목을 띄우지 않는다고 적음). 경계는 끝 카드로만 정해진다.

### 4. 제안하는 변경 목록 (적용 안 함)

1. 표시 문자열 3줄은 이미 막이다. 추가 없음.
2. `docs/data-scenario-node.md:49`를 막으로 고친다.
3. `content.go:617`, `dialogue_box.go:287` 주석을 막으로 고친다(동작 영향 없음).
4. `tools/scenario_doc/build.py`를 돌려 `output/` 6줄을 다시 만든다. 손으로 고치지 않는다.
5. 그대로 둘 것: `plan/`(과거 기록), `docs/history.md`(이력), HUD·저장 목록의 `N장`/`제 N 장`, 모든 식별자.
6. `build.py:145`의 `"장 끝 카드"`는 질문 2 결정에 따른다.

### 5. 위험

- 노드 ID 변경 금지(저장·`mode_longzhong_test.go`).
- 척도 혼동: 메모리 `equipment-grade-tiers.md`는 "큰 장(1·2·3장)"을 막 개념으로 쓰고, 코드는 `GearTiers`를 챕터 번호로 쓴다.
- `fullstory.md`의 `N부` 소단락(예: `1부 · 같은 꿈` = 챕터 1~7, `2부 · 하북`)과 막이 겹쳐 세 척도(막·부·장)가 생긴다.
- 단일 원본 규칙: `docs/fullstory.md`를 먼저 고친 뒤 데이터로 옮긴다(메모리 `fullstory-single-doc`).
- 테스트: `mode_longzhong_test.go`는 노드 ID만, `scenario_test.go`는 `ChapterNumber`만 검사한다. 카드 문자열이나 `제 %d 장`을 검사하는 테스트는 찾지 못했다.

### 6. 사용자에게 물을 질문

1. HUD `제 16 장`은 그대로 둘까, 아니면 `1막 · 제16장`처럼 막을 함께 보일까? 후자는 UI 변경이다.
2. 생성 문서의 `장 끝 카드`를 `막 끝 카드`로 바꿀까? 카드는 막 끝에 뜨지만 챕터 파일 끝에도 붙는다.
3. `fullstory.md`의 `N부`와 막의 관계는? 1부(1~7)·2부(8~16)가 1막 안에 있는 것으로 보인다.
4. 3막 시작을 28장으로 확정할지.
5. 1막=01~16, 2막=17~27(+25v~27v) 경계에 동의하는지.

### 7. 확인하지 못한 것

- 게임 화면에서 `1막 난세도래 / 끝` 카드가 깨지지 않는지(캡처 미확인).
- `internal/gui/uidemo_*.go`, `docs/ui-mockup/index.html`의 문자열은 전수 확인하지 않음. `reports/` 스냅샷은 생성물이라 제외.
- Go 테스트는 실행하지 않았다(텍스트 변경이 없었음).
- 다른 세션이 데이터를 바꾼 이유는 확인하지 못했다.
