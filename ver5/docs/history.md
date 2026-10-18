# 결정 기록

이 문서가 맡는 범위: 설계 결정과 그 이유. 현재 규칙은 각 분야 문서(`logic-*`, `data-*`, `presentation-*`, `tools-*`, `system-*`)가 말하고, 여기에는 **왜 바꿨는가**만 적는다.
변경할 때마다 적는다. 변경한 파일 이름, 수정 내역, 의도(또는 이유)를 `파일이름: 수정내역 — 의도` 형식으로 남긴다. 새 항목을 위에 쌓는다. 설계 결정은 `결정 · 이유 · 영향받는 문서` 항목으로 쓴다.

---

## 2026-10-18

### 시나리오를 `assets/scenario/`로 옮기고 장 파일과 스테이지·맵 파일로 나눔
- `assets/data/script.json`·`maps/B0x.json`을 `assets/scenario/<시나리오>/`의 `scenario.json`(`Campaign`+`Chapters`)·`storyNN.json`(장의 노드)·`stageNN.json`(스테이지)·`maps/mapNN.json`(지형)으로 나눔 — 장 흐름이 한 곳에서 보이고, 큰 `Cells`가 사람이 고치는 스테이지 규칙과 섞여 diff를 해치지 않게. 스테이지 ID는 그대로(`B01`)이고 파일은 접두어를 뗀 번호. `Stage.Next`는 없애고 노드 `Next`와 장 순서(`Data.After`)가 대신함. 옛 데이터와 같은 시드의 AI 완주 상태 해시가 같음을 확인.
- 영향: `internal/content/scenario.go`, `internal/mapedit/store.go`, `docs/data-scenario-node.md`·`logic-campaign.md`·`data-map.md`

### 스토리 노드를 늘리고 플래그를 둠
- `say`·`narrate`·`scene`·`set`·`branch`·`end`를 더하고 `preparation`을 `battle`로, 문자열 안의 `이름:` 파싱(`dialogue`)을 `Speaker` 필드로 바꿈. `State`에 `Flags`·`Choices`·`Place`(저장 `Format 3`). 분기 조건은 문자열(`flag:` `alive:` `choice:노드=옵션` `won:` …)로 적고 로드 때 파싱·검증함.
- 영향: `internal/core/engine.go`·`cond.go`, `internal/content/cond.go`, `docs/todo/engine.md`

### 후보 그림(assets_candidate) 첫 묶음
- `assets_candidate/`: 새 폴더(게임 자산은 건드리지 않음). 닫힌 관문, 불탄 숲·나무·마을·주막, 그을음 3단계, 울타리 16종, 손상·파괴 다리, 화로·지도 탁자, 불길·연기 프레임과 장면 4개 시연(`viewer.html`) — 불·연기·그을음·울타리·관문·다리로 시나리오 장면을 꾸미려면 무엇이 필요한지 눈으로 보고 고르려고. 기존 그림을 바탕으로 만들어 도트 스타일을 맞춤. 이어서 전투맵(데포르메)·장면 크기를 나눔: 성문은 주둔 가능한 관문 타일과 구분되게 새로 그린 한 칸 성문으로(바닥 + 통로 앞뒤 끝의 문틀 두 개, 문 위는 비움 — 열리면 병사가 걸어 지나가도 어색하지 않게). 문짝에 두께와 외곽선을 주고 안쪽으로 열려 옆 성벽에 붙게 한 뒤, 다시 슬레이트 지붕을 얹되 기둥을 32px로 높여 기병이 지나갈 높이로 함(바닥·뒤 문틀·앞 문틀·지붕 4층). 도개교 성문은 어색해서 접고, 처음 형태(돌 블록 + 아치 + 지붕)를 주변 성벽 높이로 낮춘 성문 C로 대체(열리면 그림이 사라짐). 성문 C는 성벽 블록 그림 위에 만들고 아치를 반원 머리로 고침 — 바닥 외곽선이 안쪽으로 들어가 보이고 문 오른쪽 위가 눌려 보이던 것을 바로잡음. 성벽 칸 이음새의 세로 눈금을 지운 성벽 `wall_clean_*`도 추가. 시나리오 장면 후보 9개(길·언덕·진영·성 밖·성 안·성루·관아·궁궐·동승의 집·조조의 후원)와 새 소품 11종(병풍·침상·등롱·휘장·계단·정원석·매화·정자·이정표·향로)을 추가 — 시나리오 52장면의 장소를 전부 덮기 위해. 성루는 바닥 전체를 돌 통로로 하고 오른쪽 위 가장자리만 성가퀴로, 길은 길 방향을 따라 걷게, 관아·궁궐은 단상과 융단 중심으로 배치를 다시 짜고 휘장을 벽 방향에 맞춤, 불꽃은 좁은 불꽃 여러 개 타원 배치로, 울타리는 전투맵용 낮은 것과 장면용 두 벌로 다시 만듦.

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

### 헌제 캐릭터 추가
- `assets/data/characters.json`: 헌제(`xiandi`)를 추가 — 경보병(전투 모습을 따로 만들 필요가 없어 일반 보병), 레벨 1, 능력치 28/14/72/33/40/78(통솔·무력은 낮고 지력·매력이 높은 황제), 장비 없음, `Look.Scene`은 `xiandi_noncombat`, 진영은 `han`. 초상은 아직 없어 병종 초상이 나온다. 사용자 요청.
- `assets/sprites/xiandi_noncombat.spr`, `internal/sprite/testdata/units/xiandi_noncombat.png`: 헌제의 장면용 비무장 도트를 빌드 — 면류관과 바닥에 끌리는 도포.
- `docs/todo/data.md`, `docs/data-character.md`: 헌제·손견이 `characters.json`에 있다고 적음. 하비 전투의 여포 백마·`DuelFirst`는 이미 `todo/data.md`에 적혀 있어 그대로 둠.

### 조조의 시작 장비를 의천검으로
- `assets/data/characters.json`: 조조의 `Equipment`를 철창에서 의천검(+가죽갑옷)으로 바꿈 — 사용자 결정. 조조가 `다재다능`이라 검을 장착할 수 있고, 검을 들었으니 검 든 도트(`caocao_sword`)로 보이며 적의 물리 공격 명중률이 15%p 낮다.

### 청공검(부채꼴 5칸)과 의천검(회피 15%p) 추가
- `assets/data/equipment.json`: 청공검(`qinggang_sword`, 검, 공격 +3)과 의천검(`yitian_sword`, 검, 공격 +4)을 추가 — 조운이 청공검을 빼앗아 드는 이야기와 조조의 보검을 장비로 만들려고. 무기 종류가 검이라 장착하면 조조·조운의 검 든 도트가 나온다.
- `internal/content/vocab.go`, `internal/core/combat.go`: 효과 키 `sweep_arc`(근접 일반 공격이 앞 칸을 중심으로 공격자를 둘러싼 5칸의 적도 비율만큼 침, `arcAhead`)와 `evade`(방어자의 회피, 물리 명중률에서 %p만큼 뺌)를 추가 — 사용자가 정한 장비 효과. 부채꼴은 화면 방향 8칸 고리(N NE E SE S SW W NW)에서 앞을 중심으로 좌우 둘씩이며 앞이 E면 N·NE·E·SE·S다. 사용자가 처음 적은 E 예시에서 N이 빠진 것은 실수라 정정함.
- `assets/data/traits.json`: 장비 전용(비용 0) 특성 `qinggang_edge`(청공: `sweep_arc` 0.5)와 `heaven_blade`(의천: `evade` 15)를 추가. 처음에 임시로 넣었던 찔러공격·사방격 효과는 이것으로 바꿈.
- `internal/core/traits_test.go`, `internal/testdata/mini/traits.json`: 부채꼴(NE·SW·E 앞에서 맞는 칸, 거리가 있으면 안 닿음)과 회피(명중률이 정확히 15%p 내려감) 시험을 추가.
- `docs/logic-traits.md`, `logic-combat.md`, `data-item.md`, `todo/data.md`: 장비 전용 특성, 물리 명중식, 청공·의천 효과 설명, 아이템 할 일(육도만 남음)을 갱신.
- 누가 언제 얻는지(`grant` 노드, 시작 장비)는 시나리오 데이터에서 정한다. 지금 조조·조운의 시작 장비는 철창 그대로.

### 장비한 무기에 따라 도트가 바뀜(조조·조운), 검 색을 어두운 회색으로
- `internal/content/content.go`, `validate.go`: `Look.Sprites`의 키를 `병종` 또는 `병종:무기 종류`로 넓히고 검증도 맞춤 — 청공검을 빼앗아 드는 조운처럼 무기를 바꿔 들면 모습이 바뀌게 하려고.
- `internal/core/types.go`, `engine.go`: `UnitView.Weapon`(장착 무기 종류) 추가 — 화면이 무기 종류를 알아야 해서.
- `internal/gui/look.go`, `view.go`, `deployment.go`, `duel.go`, `duel_gallery.go`, `editor.go`, `editor_ui.go`: `battleClassArt`와 `duelFighter`가 무기 종류를 받아 `병종:무기` → `병종` → `Look.Sprite` → 병종 스프라이트 순으로 고름. 정의만 있는 곳(일기토 보조·갤러리)은 정의의 무기 장비로 판정(`startingWeapon`).
- `assets/data/characters.json`: 조조·조운의 `Look.Sprite`를 창 든 `caocao`·`zhaoyun`으로 두고, 경기병·중기병·친위대가 검을 들면 `caocao_sword`·`zhaoyun_sword`가 나오게 함 — 사용자 결정(무기에 따라 다르게).
- `tools/doteditor/tools/mounted_kits/heroes.py`: 검 날을 흰색(철1)에서 어두운 회색(철3·철2 램프, 날 가장자리만 철1에 가까움)으로 바꾸고 `caocao_sword`·`zhaoyun_sword`를 다시 변환 — 사용자 요청("좋은 검이 너무 하얗다"). 날 색은 도트의 철 태그를 따르므로 조운은 은빛이라 조조보다 밝은 회색이다.
- `assets/sprites/{caocao,zhaoyun}.spr`와 PNG: 창 든 도트를 빌드, `caocao_sword`·`zhaoyun_sword`는 다시 빌드.
- `internal/gui/look_test.go`: 무기에 따른 선택(창·검·없음·다른 무기) 시험을 추가.
- `docs/presentation-scene.md`, `data-character.md`, `todo/data.md`: 무기별 스프라이트 규칙, 조조·조운 열전, 청공검 아이템 할 일을 갱신.

### 검 든 조운(zhaoyun_sword)
- `tools/doteditor/tools/mounted_kits/heroes.py`: 검 든 조운 유닛 추가. 조조의 검 설정(관우의 `언월도 횡베기` 휘두르기에 짧은 곧은 검)을 `swordsman`으로 뽑아 두 유닛이 같이 쓰고, 은빛 갑옷 기병은 `silver_rider`로 뽑아 창 든 조운과 나눔 — 사용자 요청(조운도 조조처럼 검). 같은 코드를 쓰므로 조조의 검은 그대로.
- `tools/doteditor/units/zhaoyun_sword.yaml`, `assets/sprites/zhaoyun_sword.spr`, `internal/sprite/testdata/units/zhaoyun_sword.png`: 새 파일(기반 중기병, 태그 색은 `zhaoyun`과 같음). 조운의 `Look.Sprite`는 아직 창 든 `zhaoyun`이고, 검 든 도트를 쓸지는 사용자 결정을 기다림.

### 일기토 우선 AI 설정, 여포의 백마는 하비에서, 나머지 장수 도트 연결
- `internal/content/content.go`, `internal/core/types.go`, `engine.go`, `api.go`: 스폰 설정 `DuelFirst`를 `Guard`와 같은 경로(스폰 → 유닛 → 저장 검증)로 추가하고 AI 정책 버전을 `greedy-8`로 올림 — 백마로 활을 쏘는 여포가 일기토 상대를 멀리서 죽여 일기토가 사라지는 문제를, 호로관만 특성을 덮어쓰는 대신 AI 설정으로 풀려고(옛 저장·리플레이는 버전 검사로 거부됨).
- `internal/core/formation.go`: `pendingDuel`을 `duelFor`에서 떼어 내고 `DuelPending(a, b)`를 공개 — 위치와 상관없이 치르지 않은 일기토 상대인지 AI가 물을 수 있게.
- `internal/ai/ai.go`: `DuelFirst` 유닛은 일기토 상대와 맞닿는 이동 칸을 1500점으로 먼저 고르고, 상대를 맞닿기 전의 공격(원거리 포함)은 건너뜀. 일기토가 꺼져 있으면(`duels` 거짓) 무시.
- `internal/ai/coalition_test.go`: 호로관 테스트를 `coalitionDuels`로 나누고, 여포에게 백마를 줘도 일기토 두 건이 일어나는지 보는 시험을 추가 — `DuelFirst`를 끄면 무안국이 먼저 쓰러져 실패함을 확인.
- `assets/data/maps/B03.json`: 호로관의 여포와 연합 장수(목순·무안국)에 `DuelFirst`를 켬. 앞서 넣은 여포 특성 덮어쓰기(`Traits: ["valor"]`)는 뺌.
- `assets/data/characters.json`: 여포의 기본 특성에서 `white_horse`를 뺌 — 사용자가 마지막 전투(하비) 즈음에 얻게 하려고 함. 하비 스테이지가 아직 없어 `docs/todo/data.md`에 "하비 스폰에 `Traits`와 `DuelFirst`" 할 일로 적음.
- `assets/data/characters.json`, `assets/sprites/{yuanshao,zhaoyun,yuanshu,sunjian}.spr`, `internal/sprite/testdata/units/*.png`: 원소·조운·원술은 `Look.Sprite`를 새 도트로 연결하고(원소의 `themeA` 노랑과 조운의 말 색 `Look.Palette`는 도트에 이미 있어 지움), 손견은 열전 수치로 새 캐릭터를 추가(경기병·오 진영, 초상은 아직 없어 병종 초상) — 사용자 요청.
- `internal/gui/scenario_test.go`: 캐릭터 스프라이트 검사가 `Look.Sprites`의 값도 확인하도록 넓힘 — 병종별 스프라이트 ID가 틀려도 잡히게.
- `docs/data-scenario-node.md`, `logic-battle-event.md`, `system-save.md`, `todo/data.md`, `data-character.md`: `DuelFirst`, 저장 AI 버전, 하비의 여포, 열전을 갱신.

### 여포의 백마와 활 든 도트, 병종별 스프라이트, 근접 판정을 8칸으로
- `internal/content/content.go`, `validate.go`: `Look.Sprites`(병종 ID → 스프라이트 ID)와 `Class.Shoot`(`mounted_range`로 쏠 때 취하는 병종) 추가, 검증은 없는 병종과 원거리가 아닌 `Shoot`를 막음 — 한 인물이 병종마다 다른 모습을 갖게 하고(병종을 바꾸면 보이는 모습이 바뀜), 근접 병종이 쏠 때의 모습을 데이터로 정하려고.
- `internal/core/combat.go`, `rules.go`, `types.go`, `engine.go`: 근접·원거리 판정을 `|dx|<=1 && |dy|<=1`(`adjacent`)로 바꾸고, `UnitView.Shoot`(`shootClass`)을 추가 — 사선으로 붙은 적을 칠 때 창을 휘두르는데 규칙은 원거리가 되는 어긋남을 없애려고(사거리는 그대로 `|dx|+|dy|`).
- `internal/gui/look.go`, `player.go`, `view.go`: 전투·장면 스프라이트가 `Look.Sprites[병종]`을 먼저 보고, 기본 공격이 8칸 밖이면 공격 동안 `shoot` 모습으로 바꿔 화살을 쏨(`shootArt`, `shootsFromRange`) — 여포가 평소엔 화극을 들고 거리가 벌어지면 활을 쏘게 하려고.
- `assets/data/classes.json`(경기병·중기병·친위대 `Shoot: horse_archer`), `traits.json`(`white_horse`의 `mounted_range` 2→3, 설명 갱신), `characters.json`(여포 `Traits`에 `white_horse`, `Look.Sprite`=`lubu`, `Look.Sprites`=경기병 `lubu`·궁기병 `lubu_bow`, 도트에 구워 넣은 `Look.Palette` 제거) — 백마의 최대 사거리를 궁기병의 사거리(3)로 맞추고 여포에게 줌.
- `assets/data/maps/B03.json`: 호로관의 여포 스폰에 `Traits: ["valor"]`를 덮어씀 — 백마를 주자 여포가 3칸 밖에서 무안국을 쏴서 일기토(`hulao-wuanguo`)가 일어나지 않아 호로관 AI 통합 테스트가 깨졌고, 연출된 일기토 흐름을 지키려고.
- `tools/doteditor/tools/mounted_kits/heroes.py`, `units/lubu_bow.yaml`, `lubu.yaml`: 활 든 여포(`lubu_bow`, 궁기병 키트의 활·화살통·사격 포즈에 여포의 중갑·마갑·꿩깃 머리)를 추가하고 `lubu`를 전 동작으로 확정, `assets/sprites/lubu.spr`·`lubu_bow.spr`와 스프라이트 비교용 PNG를 빌드 — 여포의 두 모습.
- `internal/core/traits_test.go`, `internal/gui/look_test.go`, `internal/testdata/mini/classes.json`: 시험 추가 — 8칸 판정과 `Shoot`, 병종별 스프라이트와 활 모습 전환 조건.
- `docs/logic-combat.md`, `logic-traits.md`, `presentation-scene.md`, `data-character.md`: 근접 판정, 백마, 병종별 스프라이트와 활 모습 전환, 여포 열전을 현재형으로 갱신.

### 조조·조운에게 다재다능(검·창 겸용), 조조는 검 든 도트로
- `internal/content/vocab.go`: 효과 키 `wield:<무기 종류>`(`FxWield`) 추가 — 병종 무기와 다른 무기를 장착하는 규칙을 데이터(특성)로 주려고. 접두 키라 `validate.go`의 키 검사가 그대로 통과한다.
- `internal/content/spawn.go`: `Data.Carries(class, traits, equipment)`와 `Data.SpawnTraits(spawn)` 추가, 스폰 장비 검증이 `Carries`를 씀 — 장비 호환 판정이 엔진·저장·스폰·편집기 여러 곳에 복사돼 있어서 한 함수로 모음.
- `internal/content/validate.go`, `internal/core/engine.go`, `internal/core/api.go`, `internal/gui/editor.go`, `internal/gui/editor_ui.go`: 각자의 `Weapon`/`Armor` 비교를 `Data.Carries`로 교체(저장 검증은 장수 정의·학습·덮어쓴 특성으로 판정) — 특성으로 무기를 얻은 장수가 장착·저장·편집기 목록에서 막히지 않게.
- `assets/data/traits.json`: `versatile`(다재다능, 비용 0: `wield:spear`, `wield:sword`) 추가. `assets/data/characters.json`: 조조 `Traits`에 `versatile`, `Look.Sprite`를 `caocao_sword`로, 조운 `Traits`에 `versatile` — 사용자 결정(조조는 검 든 도트를 쓰고 조조·조운이 검과 창을 모두 든다). 조운의 `Look.Sprite`는 아직 그대로(새 도트는 `.spr` 빌드 전).
- `assets/sprites/caocao_sword.spr`, `internal/sprite/testdata/units/caocao_sword.png`: 새 파일. `build.py`로 `.spr`만 빌드(`make apply` 전체는 돌리지 않음)하고, 스프라이트 테스트가 시트마다 비교용 PNG를 요구해 이 유닛만 `golden.py`와 같은 방식으로 PNG를 만듦.
- `internal/testdata/mini/traits.json`, `internal/core/core_test.go`: 시험용 특성 `lancer`(`wield:lance`)와 `TestEquipmentRules` 확장 — 특성으로 다른 무기는 장착되고 그 밖의 종류는 여전히 거부됨을 확인.
- `docs/logic-traits.md`, `docs/data-item.md`, `docs/data-character.md`: 다재다능 규칙과 장착 규칙, 조조·조운 열전을 현재형으로 갱신.

### 장수 7명의 전용 도트를 3D 변환(class-sprite-3d)으로 시작
- `tools/doteditor/tools/mounted_kits/heroes.py`: 새 파일. 조조(친위대 + 관모·검은 수염), 여포(중기병 + 방천화극 + 꿩깃 금관), 원소(친위대 + 금투구, 장식 높이 절반), 조운(은빛 갑옷 기병 + 붉은 깃), 손견(두건 쓴 경기병), 헌제(바닥에 끌리는 도포의 서생 + 면류관), 원술(가는 몸 + 금 문양 + 원소에서 옮긴 긴 장식)을 기존 키트의 머리·갑옷·몸 변형으로 등록 — 병종 도트 위에 팔레트만 바꾸면 한 병종의 장수들이 구별되지 않아서. S 방향 얼굴은 양쪽 1px씩 좁혀 경기병 폭에 맞춤. 금투구 하이라이트 키 `n`은 말 가죽 키와 겹쳐 `m`으로 합침.
- `tools/doteditor/units/{caocao,yuanshao,lubu,zhaoyun,sunjian,yuanshu}.yaml`: 새 파일. 대기(idle) 8방향 × 4프레임만 3D 변환이고 나머지 동작은 기반 병종(친위대·중기병·경기병·경보병)의 프레임 그대로 — 사용자가 먼저 대기 모션만 요청했기 때문.
- `tools/doteditor/units/xiandi_noncombat.yaml`: 새 파일. 헌제는 비무장(NW·SW, 대기 1 + 걷기 2 + 동작 1)만 — 장면용 인물이라 전투 모션이 필요 없다는 결정.
- 태그 색은 `grid.py colors`로 장수별로 지정: 여포 적토마(붉은 말), 조운 백마·은빛 철 색·붉은 깃(themeC), 조조·손견·헌제 검은 머리카락(수염이 가장 밝은 hair1), 원술·헌제 금 장식(themeC).
- `tools/doteditor/units/lubu.yaml`: 말 피부·갈기 태그 색을 이전 `Look.Palette` 값(`#e06050/#b03020/#741a10`, `#901818/#601010/#300808`)으로 되돌림 — 미리보기용 어두운 적토마 색보다 기존 색이 더 선명해서 사용자가 기존 색을 선택.
- `tools/doteditor/units/lubu.yaml`: 철(`#6a6a78/#3a3a46/#1a1a22`)과 themeA(`#d04848/#9a2424/#5a1010`)도 이전 `Look.Palette` 값으로 맞춤 — 기존 여포의 어두운 철과 붉은 가죽 톤을 유지하려고.
- 여포 진영색은 검정으로 바꾸지 않고 기존 `dong`(동탁과 같은 색)을 유지 — 검정은 3D 미리보기에서만 쓴 임시 색이었고, `game.json`의 `dong`을 바꾸면 동탁까지 바뀌기 때문. 변경한 파일 없음.
- `tools/doteditor/units/{caocao,yuanshao,lubu,zhaoyun,sunjian,yuanshu}.yaml`: 걷기·공격·피격·탈진·막기까지 3D 변환(+`--denoise`)해 8방향 전 동작을 채움 — 대기만 바꾸면 움직일 때 기반 병종으로 보이기 때문. 원술 막기 행에서 `Bl.foot`의 `style` 인자를 빠뜨려 실패해 `heroes.py`에서 `Bl.SELECTED`를 넘기도록 고침.
- `assets/data/game.json`: 진영 `han`(한 황실, `#ffd85a/#e0a81e/#9a6a10`)을 추가 — 헌제의 옷(진영색 도포)을 황색으로 보이게 하려는데, 진영색 태그는 도트에서 장수별로 못 바꾸고 `yuan`(원소·원술의 레몬 노랑)과 구분되는 황금색이 필요해서. 헌제 캐릭터를 만들 때 `Look.Faction: han`으로 쓴다.
- `tools/doteditor/tools/mounted_kits/heroes.py`: `caocao_sword` 유닛 추가(검 든 조조, 대기·걷기·공격·피격·탈진·막기 8방향). 관우의 기마 코드를 재사용해 `units.glaive_at` 자리에 곧은 검(손잡이·금 날밑·길이 1.35 날)을 그리고, `언월도 횡베기` 휘두르기와 궤적을 그대로 따르되 무기 도달 거리를 언월도의 약 2/3(`MOUNT_REACH['glaive']` = (.45, 1.5))로 줄임 — 창대 달린 언월도보다 짧되 적에게 닿지 않아 보이지 않게.
- `tools/doteditor/units/caocao_sword.yaml`: 새 파일(기반 친위대, 머리카락 태그 색은 `caocao`와 같음). 창 든 `caocao`는 그대로 두고 둘 중 게임에서 쓸 것을 나중에 고른다.

- 게임 반영(`make apply`, `Look.Sprite` 연결, 여포 진영색 검정)은 하지 않음 — 스킬의 정지선이며 `dong` 진영색 변경은 동탁에도 영향을 주므로 사용자 결정을 기다린다.

### 소비 아이템을 영걸전 방식으로 바꿈
- 병력 회복은 콩·보리·쌀(소·중·대), 병법치 회복은 술·특급주·빼갈(30·60·120). 군량은 화폐로만 쓴다.
- 이유: 화폐 군량과 소비 아이템 "소군량·중군량"의 이름이 겹쳤다. 영걸전의 회복 아이템 이름(콩·보리·쌀, 술 계열)을 따랐다.
- 영걸전의 술 계열은 사기 회복(PC판)이지만 여기서는 병법치 회복으로 쓴다. ver5에 병법치가 있고 소병법단이 이미 병법치 회복이었기 때문이다. 회복량은 기존 소병법단 15가 낮다고 보아 30·60·120으로 올렸다.
- ID는 `bean`, `barley`, `rice`, `liquor`, `fine_liquor`, `baijiu`. 쌀 2200은 임시 값이다. 상약·서 계열은 나중에 정한다.
- 영향: `data-item.md`, `todo/data.md`

## 2026-10-17

### 변경 기록 규칙 추가
- `AGENTS.md`: 변경할 때마다 `history.md`에 `파일이름: 수정내역 — 의도`를 적는다는 규칙을 추가 — 무엇을 왜 바꿨는지 한곳에서 추적하려고.
- `docs/history.md`: 머리말의 "커밋에 남는 것은 적지 않는다"를 새 규칙에 맞게 고침 — 규칙이 서로 어긋나지 않게.

### 비무장 스프라이트와 장수 도트 범위 확정
- `docs/todo/graphics.md`: "비무장(장면용)"을 신규 제작에서 기존 병종 도트 + 옷색·진영색 재사용으로 바꿈(서복·선비·헌제는 서생, 병사는 경보병, 피난민·촌민·노인은 서생/경보병, 전령은 경보병/경기병) — 장면 배우가 `cast`·`Look.Scene`·병종 스프라이트 순으로 스프라이트를 찾으므로 새로 그릴 필요가 없다.
- `docs/todo/graphics.md`: 전용 도트 장수를 조조·여포·조운·손견·원소로 확정, 동탁·공손찬은 보류, 나머지는 병종 도트 + 팔레트 — 시나리오 등장 비중이 큰 인물만 그려 작업량을 줄이려고. `_noncombat` 시트는 장면에서 어색할 때만 추가.

### 전용 도트 장수에 공손찬·동탁 추가
- `docs/todo/graphics.md`: 전용 도트 장수를 조조·여포·조운·손견·원소·공손찬·동탁(배불뚝이)로 확정, 보류 표기 삭제 — 공손찬은 언급 49회로 비중이 크고, 동탁은 배불뚝이 체형으로 병종 도트와 달라 전용이 필요하다.

### 전용 도트 장수를 12명으로 확장
- `docs/todo/graphics.md`: 전용 도트 장수에 고순·원술·헌제·허저·문추를 추가(조조·여포·원소·조운·공손찬·손견·동탁 포함 12명), 비무장 목록에서 헌제를 뺌 — 언급 횟수가 많거나 체형·역할이 병종 도트와 달라 팔레트만으로는 부족한 인물이라서.

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

### 쓰러뜨릴 수 없는 적(`Unyielding`)을 두지 않음
- 여포는 그냥 강하게 만든다. 일기토 외의 이유로 퇴각해도 그대로 둔다. HP 임계 회복(`hp`+`heal`)도 쓰지 않는다.
- 영향: `todo/engine.md`

### 일기토 디버프를 없애고 이벤트로 처리함
- `AttackDown`/`DefenseDown` 필드와 `attack-down`/`defense-down` 상태를 지웠다. 디버프는 스킬 디버프 하나뿐이다. 일기토 승리 뒤의 효과는 `duel:ID` 조건 이벤트의 `buff` 동작으로 건다(이벤트 시스템 구현 전까지 옛 B01의 두 효과는 비어 있다).
- 영향: `logic-duel.md`, `data-scenario-node.md`, `todo/engine.md`

### 일기토는 인접하는 순간 자동 발생
- 공격이 일기토로 바뀌던 방식을 버린다. 지정한 두 장수가 인접한 상태가 되면(이동 뒤 대기·공격·소환·합류 어느 경우든) `settle()`의 첫 단계에서 벌어진다. 결과는 지금처럼 데이터로 고정이다.
- 영향: `todo/engine.md` 2.5

### 이벤트 트리거는 상태 조건 하나로 통일함
- 엣지 트리거(`enter`·`stay`·`adjacent`·`hp`·`defeated`)를 없애고 상태 조건(`at`·`adjacent`·`hp`·`defeated`…)으로 줄인다. 검사 시점 필드는 두지 않고, 시점이 필요하면 `turnstart:진영` 조건을 쓴다. 방화병의 "1턴 생존"은 `turnstart:enemy`(적 차례 시작, 아군이 한 번 행동한 뒤)에 `at`을 검사하는 것으로 한다.
- 영향: `todo/engine.md` 2.2

### 불은 타일 두 종류로 표현함
- 불길(진입 불가)과 불붙은 땅(배율 0.8, 차례를 맞으면 최대 병력의 20% 손실). 지나가기만 하면 피해가 없다. 불이 붙는 것은 `terrain` 동작이 타일을 바꾸는 것으로만 일어난다.
- 영향: `todo/engine.md` 2.6

### 전투 없는 장은 스테이지가 아니라 스토리
- 2 낙양, 6 산조, 12 서주 계승, 16 허도, 24 삼고초려는 노드 시퀀스로만 이루어진 스토리다. 전투 중 이벤트와 구별한다.
- 영향: `conceptual_scenario.md`, `todo/engine.md`, `todo/data.md`

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
