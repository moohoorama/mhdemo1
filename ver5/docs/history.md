# 결정 기록

이 문서가 맡는 범위: 설계 결정과 그 이유. 현재 규칙은 각 분야 문서(`logic-*`, `data-*`, `presentation-*`, `tools-*`, `system-*`)가 말하고, 여기에는 **왜 바꿨는가**만 적는다.
커밋에 남는 것(무엇을 바꿨나)은 적지 않는다. 새 항목을 위에 쌓는다. 항목 형식: `날짜 · 결정 · 이유 · 영향받는 문서`.

---

## 2026-10-18

### 설정에 경험치 배율을 추가
- `internal/storage/storage.go`: `Settings.XPPct`(60~100, 기본 80)와 검증, 선택지 목록을 `ScalePcts`로 데미지와 공용 — 레벨업 속도를 사용자가 조절하게.
- `internal/core/types.go`, `api.go`, `growth.go`: `State.XPPct`·`SetXPPct`, `actionXPAmount`·`supportXP`에서 곱함(최소 1) — 전투 행동·방어·지원 경험치와 예측 창이 같은 값을 쓰게. 이벤트 보상으로 한 칸을 주는 경우(`formation.go`)는 곱하지 않음.
- `internal/session/session.go`, `internal/gui/windows.go`: 요청마다 엔진에 동기화, 설정 창에 `경험치 배율` 줄 추가(창 높이 604).
- 테스트와 `docs/system-architecture.md`, `presentation-ui.md` 갱신.

### 초열 병법 소모 40, 이유 통솔 70, 문관 계열 회복 기본값 8
- `assets/data/skills.json`: 초열 `Cost` 50 → 40 — 첫 화계를 더 자주 쓰게.
- `assets/data/characters.json`: 이유 통솔 61 → 70 — 병법 최대치 68 → 73.
- `assets/data/classes.json`: 서생·참모·군사·방술사·선술사 `RecoveryBase` 5 → 8 — 문관 계열의 병법치 회복을 올림(이유 17 → 20). `docs/logic-combat.md` 갱신.

### 데미지 배율 선택지를 60~100%로, 기본 80%
- `internal/storage/storage.go`, `session/session.go`: 선택지 30·50·70·100 → 60·70·80·90·100, 값이 없으면 80(`DefaultDamagePct`). 세션은 `DamageScale()`을 엔진에 동기화 — 30%는 너무 느려져 쓸모가 적고 기본을 100%보다 낮게 두고 싶다는 요청.
- 테스트와 `docs/system-architecture.md`, `presentation-ui.md` 갱신.

### 2회 공격의 두 번째 타격을 ×0.7로
- `internal/content/content.go`, `assets/data/game.json`, `internal/testdata/mini/game.json`: `Rules.DoubleShare`(0.7) 추가 — 연타가 첫 타와 같은 피해를 내면 치명과 겹쳐 한 대상 3.0배까지 가서 과함.
- `internal/core/combat.go`: `attackOn`과 `counterBack`의 두 번째 타격에 곱함. 요극 추가 공격은 처음 해석(후속타 = 요극)이 틀려 되돌렸고 비율이 없다. 한 대상 최대는 1.5 + 1.5×0.7 = 2.55배(반격은 ×0.7 더).
- `internal/core/double_share_test.go`: 비율 검사. `docs/logic-combat.md` 갱신.

### 설정에 데미지 배율을 추가하고 선택지를 버튼 줄로 바꿈
- `internal/storage/storage.go`: `Settings.DamagePct`(30·50·70·100, 0은 100)와 검증 — 전투 속도를 사용자가 조절하게.
- `internal/core/types.go`, `combat.go`, `api.go`: `State.DamagePct`와 `SetDamagePct`, `damage()`에서 곱하고 최소 1 — 설정을 엔진 상태에 실어야 `Restore`한 예측·시험 복사본이 같은 피해를 내고 정렬·세이브가 어긋나지 않음. 지속 피해·회복은 곱하지 않음.
- `internal/session/session.go`: 요청 처리 끝에 설정값을 엔진에 동기화 — 불러온 세이브의 값이 아니라 현재 설정이 이기게.
- `internal/gui/windows.go`, `skin.go`: 설정 창을 `choiceRow`(이름 위·선택지 버튼 줄 아래, 고른 것은 눌린 채 금색)로 바꿈. 적 AI·전투 기록·대사 속도·재생 속도·UI 크기·데미지 배율 모두. 순환 버튼(`nextUIScale`)은 삭제 — 여러 값 중 지금 값을 한눈에 보고 바로 고르려고.
- `internal/storage/storage_test.go`, `session/session_test.go`, `core/damage_pct_test.go`: 설정 검증, 엔진 동기화, 배율 적용 테스트.

### 일람 탭을 간소화하고 병력일람 창을 추가
- `internal/gui/hud_units.go`: 일람 탭을 장수명·병력·병법치·명령 한 줄 목록으로 줄이고(보기 전환 제거), 정렬 순서를 `무정렬 → 내림 → 오름`으로 바꿈. 목록 상태를 `unitList`, 열 배치를 `unitTable`로 묶어 탭과 팝업이 같은 표 코드를 씀 — 우상단 패널(폭 320)에 6칸 두 줄은 너무 좁아서.
- `internal/gui/hud_roster.go`: 새 파일. `기타 › 병력일람` 오버레이(`roster`). 화면을 거의 채우는 창에 한 줄·큰 글씨(17)로 기본(병종·레벨·병력·병법치·명령)·전투력·열전·장비 네 보기를 보이고 같은 정렬을 씀. 행을 누르면 창을 닫고 그 장수로 이동.
- `internal/gui/hud_panel.go`, `windows.go`, `game.go`, `uidemo_hudpanel.go`, `hud_units_test.go`: 기타 탭에 `병력일람` 버튼(높이 180→220), 오버레이 크기·제목, 필드 `unitsList`/`rosterList`, 데모 프레임, 테스트 갱신.
- `docs/presentation-ui.md`: 일람 탭·기타 탭 설명을 새 동작에 맞춤.
- `internal/gui/hud_roster.go`: 필터·보기 버튼 폭을 창 폭에 맞춰 줄임 — UI 160%에서 `아군/적군/전체`와 `기본/전투력/열전/장비`가 겹쳤음.
- `internal/gui/layout_overlap_test.go`: 새 파일. 화면 5종 × UI 크기 5종 × 화면 13종(HUD 탭·메뉴·설정·작전 개요·병력일람 보기)을 그려 버튼끼리 겹치거나 화면을 벗어나면 실패 — 그리는 동안 `g.buttons`에 쌓이는 사각형만 읽으므로 네이티브 렌더러 없이 `go test`로 돎.
- `internal/gui/hud_panel.go`: 탭 순서를 미니맵·장수·일람·기록·기타로 바꾸고 `장수` 탭을 선택 없이도 고를 수 있게 함(선택 해제 뒤 돌아갈 기본 탭이 될 수 있음, 비면 안내 문구) — 장수를 기본 화면으로 두고 싶다는 요청.
- `internal/gui/events.go`: `deployment_finish` 이벤트를 기록에서 숨김 — 배치 종료마다 `deployment_finish  0`이 기본 형식으로 새어 나왔음.
- `internal/gui/hud_units.go`, `hud_roster.go`: 병력일람 창의 머리·값을 칸 가운데 정렬, 전투력의 `회복`을 `병법회복`으로 바꿈 — `회복`이 병력 회복으로 읽혔지만 실제로는 턴당 병법치 회복량(`Stats.Recovery`).

### 소비 아이템을 영걸전 방식으로 바꿈
- 병력 회복은 콩·보리·쌀(소·중·대), 병법치 회복은 술·특급주·빼갈(30·60·120). 군량은 화폐로만 쓴다.
- 이유: 화폐 군량과 소비 아이템 "소군량·중군량"의 이름이 겹쳤다. 영걸전의 회복 아이템 이름(콩·보리·쌀, 술 계열)을 따랐다.
- 영걸전의 술 계열은 사기 회복(PC판)이지만 여기서는 병법치 회복으로 쓴다. ver5에 병법치가 있고 소병법단이 이미 병법치 회복이었기 때문이다. 회복량은 기존 소병법단 15가 낮다고 보아 30·60·120으로 올렸다.
- ID는 `bean`, `barley`, `rice`, `liquor`, `fine_liquor`, `baijiu`. 쌀 2200은 임시 값이다. 상약·서 계열은 나중에 정한다.
- 영향: `data-item.md`, `todo/data.md`

## 2026-10-17

### 문서 구조를 `<분류>-<기능>.md`로 바꿈
- 분류: logic(어떻게 돌아가는가) / data(그 위에 어떤 데이터를 태우는가) / presentation / tools / system.
- 이유: 한 사실이 여러 문서에 되풀이되어 어긋났다(경험치·부관·일기토가 `battle.md`와 `design.md`에 중복 등). 한 사실은 한 문서에만 두고 나머지는 링크한다.
- 본문은 현재 코드·데이터가 하는 일만 현재형으로 쓴다. 미구현 설계는 `todo/`, 작업 중 plan은 `temp/`(git ignore). 구현이 끝나면 분야 문서로 옮기고 todo는 지운다.
- 정리 전 문서는 `docs_old/`에 보존한다(`design.md` 원본, 후반 시나리오 계획 `scenario.md`, `script-c1.md`, 옛 `scenario-system.md`, 특성 문서 3종, `ui-design.md`, `plans/`).

### 시나리오 v1(1~20)을 ver5의 기준 시나리오로 채택
- 원본(`scenario_v1.txt`)은 현재 게임 시스템 이전에 쓴 것이라, 큰 줄기에 맞으면 수정을 허용한다. 문서 이름은 `conceptual_scenario.md`(뼈대 이야기).
- 기존 B01~B03은 폐기하고 새로 만든다. 도원결의로 시작하던 흐름을 영천(노식)으로 바꾼다.
- 범위는 1~20 전체. "이벤트 스테이지"는 별도 시스템이 아니라 전투 없이 장면만 나오는 스테이지(2 낙양, 6 산조, 12 서주 계승, 16 허도)다.

### 군량을 화폐로 삼고 출전을 필수/정원/증원 3단계로 함
- 금 보상은 군량으로 바꾼다. 출전 비용, 상점 매매, 보상이 한 화폐로 정리된다.
- 필수(반드시 출전), 정원(무료로 채우는 인원), 증원(정원을 넘어 최대까지, 군량을 내고 출전). 어려우면 군량을 들여 병력을 늘리고, 아낄 수 있으면 아낀다. 시나리오의 "4기부터 병량 소모"는 이 증원 비용이 대신한다.
- 출전 전 회의장(조조전식)에서 대화와 군수물자 매매를 한다.
- 영향: `todo/engine.md`, `todo/data.md`

### 쓰러뜨릴 수 없는 적(여포)은 HP 임계 회복으로 처리
- 새 규칙 없이 이벤트 어휘로 푼다. HP가 일정 수준 아래로 내려가면 대사와 함께 100%로 회복하고, 일기토로만 퇴각한다. 한 방에 죽지 않게 HP 하한을 둔다.
- 영향: `todo/engine.md`

### 호로관의 연합 장수는 목순으로 통일
- 방열과 목순이 비슷하고, ver5에 이미 목순·무안국과 `hulao-mushun` 일기토가 있다.

### 병종 도트는 새로 그리지 않음
- 백마의종은 기병+흰색 진영색, 쇠뇌 노병대는 기존 노병, 함진영은 색을 바꾼 창병으로 쓴다. 장면 배경 맵은 막사류·성내류·궁궐류로 합쳐 쓴다.
- 영향: `todo/graphics.md`

### `design.md`를 방향과 원칙만 남기고 줄임
- 옛 `design.md`(ver2/ver4, 5437줄)는 `docs_old/`에 보존한다. 경험치·부관·일기토 규칙은 `logic-*`가 맡는다.
- 옛 설계서의 다음 서술은 현재 코드·데이터와 달라 버렸다: 레벨별 필요 경험치 곡선(실제는 모든 레벨 10,000 고정), "피격·방어 경험치 없음"(실제는 `ExpDefenseShare` 0.5), 부관 임명 특성을 병종이 준다는 서술(실제는 학습 특성).
- 특성 문서 3종은 `logic-traits.md` 하나로 합쳤다. 병종 `Grants` 폐지, 승급 레벨 15·30(옛 문서는 20·40) 등 데이터를 따랐다.

## 2026-10 (옛 `battle.md` 개정 이력에서 옮김)

현재 값은 `logic-combat.md`, `logic-growth.md`와 `assets/data/`가 말한다. 아래는 바꾼 이유만 남긴다.

- **물리 피해식**을 `max(1, 공격−방어+(레벨+10)×20)`으로 바꿨다(`PhysicalLevelScale` 20). 공방비 보정(0.8~1.2배)과 `DamageCurve` 규칙값은 없앴다. 이유: 같은 수치끼리면 레벨이 높은 쪽이 강하도록.
- **책략 피해식**을 `max(1, 정신차+(레벨+10)×10)`으로 바꿨다(`MagicLevelScale` 10). 이유: 물리와 같은 레벨 기반 구조로 통일.
- **최대 MP와 MP 회복**을 `Rules`의 곡선(`ManaMaxFactor`, `ManaRecoveryFactor`)으로 바꿨다. `MPCoeff`·`RecoveryCoeff`는 없앴다. 이유: 능력치 곡선을 다른 전투 수치와 맞추려고.
- **상위 병종 HP 보너스**를 직접 올리고 별도 승급 배율은 폐기했다. 이유: 매력 80 기준으로 2단계 약 +10%, 3단계 약 +20%를 맞추려고.
- **초열·업화 재조정**: 책략 피해식 개정에 맞춰 MP와 계수를 조정했다. 업화 MP와 물리 병법 MP, 저장 MP 정책은 유지했다.
- **`troop_scholar` 지력 30→60, 조조의 병종 경로를 경기병→중기병→친위대로, 사마의 운을 100으로** 바꿨다.
- **문무겸비**(`civil_martial`/`blend_bonus`)를 신설했다. 부여 대상은 조조·강유·등애뿐이고 학습 비용은 정하지 않았다(부여만 가능). 이유: 문무 양쪽 병종 보너스를 평균으로 올려 조조 같은 총대장형을 표현하려고.
- **규칙 버전을 올려 옛 저장을 불러오지 못한다.** 추가 행동(`free_skill`·`free_item`)과 국사무쌍·부호를 없앴다. 병종 `Grants`를 없애고 책략 권한을 학습 특성으로 옮겼다. 이유: 규칙을 단순화하고 병종과 특성의 역할을 분리하려고.
- **부관 보정률에 상한이 없다**는 점을 문서에 못박았다. 이유: 보정률이 1을 넘으면 부관이 앞선 능력치가 부관 수치를 넘을 수 있어서.
- **밸런스 점검 항목**(동급 전투의 격퇴까지 타격 수, 병종 간 대결, 문관의 지속 병법 횟수, 부관 유무별 성장 차)은 `design.md`의 밸런스 축에 있다. 수치를 바꿀 때는 같은 시드·정책·편성으로 전후를 비교하고, 코어를 바꾼 뒤에는 `go test ./... && go vet ./...`를 돌린다.

## 2026-10-18 모바일 조작·일기토 컷신 정리
- `internal/gui/unit_bar.go`: 흰 trail이 남은 상태에서 회복하면 `trailRate`가 음수가 되어 trail이 끝없이 늘어나던 버그를 절댓값 속도와 `hold=0`으로 고침 — 간옹 HP바 하얀 막대가 무한히 길어짐. 회귀 테스트 `unit_bar_test.go` 추가.
- `internal/gui/hud.go`: 행동 메뉴(menu/sub) 단계에서는 지형 `brief`를 그리지 않음 — 메뉴 4개 항목을 가림.
- `internal/gui/command.go`: 터치 입력에서 이동 목적지·공격/병법 대상을 처음 누르면 노란 하이라이트로만 표시하고(`order.armed`, `arm`, `marks`) 같은 칸을 다시 눌러야 명령 — 롤오버가 없어 지형 정보를 못 보이고 오터치가 잦아서.
- `internal/gui/duel_cutin.go`: 컷인 초상을 확대(1.35배, 크게 오프셋)하지 않고 패널 폭에 맞춤, 이름 글씨 크기를 밴드 높이에 비례해 줄임 — 얼굴이 너무 커지고 모바일에서 이름이 밴드를 넘침.
- `internal/gui/duel_decor.go`, `battle_speech.go`: 낮은 화면(`smallScreen`)에서 하단 장수 카드를 줄이고 말풍선을 키움 — 모바일에서 카드는 크고 대사는 작음.
- `internal/gui/duel_cutin.go`: 컷신 이름과 "아군/적군 장수" 표기를 각 초상 패널의 보이는 영역 중앙(x 25%·76%)에 가운데 정렬 — 이름이 화면 가운데에 몰려 있었음.
- `internal/duelview/view.go`, `internal/gui/duel.go`: 일기토 말풍선에 `BubbleScale` 추가, 낮은 화면에서 1.6배 — 모바일에서 대사가 작음. `DrawText`가 좌표에도 배율을 곱하므로 x/k, y/k로 넘겨 위치를 유지.
- `internal/duelview/bubble.go`, `view.go`, `internal/gui/battle_speech.go`: 말풍선 글자를 화면 해상도로 다시 그리는 `DrawSharp` 추가, 일기토는 캔버스를 확대한 뒤 그 위에 그림(`drawBubbles`) — 1x로 그린 글자를 확대해서 모바일에서 뭉개져 보였음.
- `internal/duelview/decor.go`: 일기토 진영의 깃발을 3개에서 6개(군기 4·지휘 깃발 2)로 늘리고, 기존 `banner`/`command_flag` 그림을 열 단위로 물결치게 비튼 4프레임 애니메이션(`wave`, 170ms)을 런타임에 만들어 씀 — 새 그림 없이 펄럭임을 내려고. 깃발마다 위상이 달라 같이 움직이지 않음.
- `internal/gui/duel.go`: 낮은 화면의 일기토 말풍선 배율 1.6→2 — 대사가 아직 작다는 의견.
- `internal/gui/duel_gallery.go`: 메뉴의 일기토 모드 뷰어에도 `Decor: g.duelDecor()`를 넘김 — 깃발·바위·풀이 없는 빈 경기장이었고, 깃발 펄럭임이 전투 중 일기토에만 보였음.
