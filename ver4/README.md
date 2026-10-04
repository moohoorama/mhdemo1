# ver4

[설계서](design.md)(v0.6)를 따르는 삼국지 SRPG. 규칙·캠페인·저장·CLI는 ver1 Go 코어를 이어받고,
전장 화면은 ver3 방식(아이소메트릭 맵, 8방향 병종 도트, 진영색, 그림자)을 Ebitengine으로 옮겼다.
AI가 진행하는 페이즈는 [클레임 리플레이](design.md#09-ai-페이즈의-클레임-리플레이)로 보여 준다.

```sh
cd ver4
go run ./cmd/srpg-gui                      # 그래픽 (1280×850)
go run ./cmd/srpg-cli                      # 사람용 CLI, -json 으로 JSON Lines
go run ./cmd/srpg-sim -seeds 10            # 반복 시뮬레이션
go test ./... && go vet ./...
go build -o bin/srpg-cli ./cmd/srpg-cli && python3 tools/smoke_cli.py   # CLI 전체 캠페인
go run ./cmd/srpg-gui -audit /tmp/shots -shots 600,1200   # 새 게임 자동 진행, 지정 틱(60/초) 화면 저장
python3 tools/build_assets.py              # assets/graphics 다시 묶기 (Pillow, PyYAML, Go)
```

GUI 조작(조조전식, [UI 설계](ui-design.md)): 아군 클릭 → 이동 칸 클릭 → 유닛 옆 행동 메뉴 → 대상에 마우스(예측 창) → 클릭.
우클릭/Esc는 한 단계 취소(이동 취소 포함), 우클릭 드래그·화살표로 화면 이동, 휠 확대(×0.5–2, 기본 ×1에서 병종 도트가 1:1).
병법·도구·학습 항목에 마우스를 올리면 효과 설명이 뜬다. 예측 창은 양쪽 병력·병법치 게이지와 변화량, 명중률을 보여 준다.
일기토는 명령이 아니다. 지정된 두 장수가 서로 공격하면 자동으로 벌어진다(설계서 0.6).
적을 클릭하면 그 적의 위협 범위를 보여 준다. U 부대 일람 · T 적 전체 위협 범위 · L 전투 기록 · E 진영 종료 ·
Tab 다음 미행동 부대 · F 재생 속도(×1·×2·×4) · Enter 재생 건너뛰기 · Space 추천 한 명령 · P 자동 진행 ·
F5/F9 저장/불러오기 · ` 명령 입력(CLI 문법, 디버그).

## 구조

```
ver4/
├─ design.md                 설계서 v0.6 (ver2 v0.5 + 클레임 리플레이)
├─ scenario.md               시나리오·스테이지 설계 (B01 상세, 전체 스테이지 목록, 숨은 조건 연쇄)
├─ scenario-system.md        시나리오 노드·회의장·상점·전투 이벤트 데이터 구조
├─ script-c1.md              서장~계교 대본과 전투 설계
├─ cmd/srpg-{gui,cli,sim}
├─ internal/
│  ├─ core, ai, session, storage, cli, sim, content   ver1에서 가져온 코어 (변경점은 아래)
│  ├─ replay                 클레임 리플레이: 행동 기록, 클레임, 점유 장부로 선행 연결
│  └─ gui                    Ebitengine 화면: assets(로딩·진영색), field(전장), player(재생), view(카메라·동기화),
│                            game(요청·입력), command(행동 흐름·예측·위협), hud, windows, scenario, effects, skin, events
├─ assets/
│  ├─ content/campaign.json  실행 콘텐츠 (ver1과 같음)
│  ├─ fonts/                 NotoSansKR
│  └─ graphics/              build_assets.py 결과: index.json, units/, map/, maps/B01–B03.json, 초상
└─ tools/build_assets.py, smoke_cli.py
```

## ver1 코어에서 바뀐 점

- 이동 결과 이벤트에 실제 경로(`Event.Path`, 양 끝 포함)를 담는다. 클레임과 걷기 연출이 이것을 쓴다.
- `ai.Next`: `ai.Step`이 적용할 명령을 돌려준다.
- 화면용 읽기 전용 조회: `Engine.OfficerStats`(전투 밖 장수 능력치), `CounterDamage`(반격 최대 피해),
  `Threat`(다음 행동의 공격 가능 칸), `DuelFor`(이 공격이 일기토를 일으키는지).
- 일기토 명령(`duel`)을 없앴다. 지정된 두 장수가 인접해서 무기로 서로 공격하면 그 공격 대신 일기토가 벌어진다.
- `Session.ResolvePhase`: 현재 진영의 페이즈를 AI로 한 명령씩 끝까지 해결하고 리플레이용 행동 목록을 돌려준다.
  결과는 `ai.Step`을 반복한 것과 같다(테스트). 끝나면 자동 저장한다.
- 규칙·콘텐츠·저장 형식은 그대로다. CLI 스모크 결과가 ver1과 같다(391 요청, 라운드 10/10/16).

## 클레임 리플레이 (`internal/replay`, `internal/gui/player.go`)

- 행동 기록 = 한 부대의 차례(연속한 명령과 그 이벤트). 진영 종료·일기토·승리는 장벽이다.
- 클레임 = 행동 부대와 출발 칸, 이동 경로의 모든 칸, 이벤트가 가리키는 부대(대상·반격자·퇴각)와 그 칸.
- `replay.Link`가 점유 장부(클레임 → 마지막 등록 행동)로 선행을 채운다. 재생은 선행이 모두 끝난 행동을
  해결 순서대로, 출발 간격을 최소 0.2초 두고 시작한다. 행동은 대상의 피격 흔들림·퇴각 깜빡임까지 끝나야 완료다.
- 적 페이즈(설정 AI 자동), 자동 진행(P)의 아군 페이즈, "적 진영 전체 진행"에 쓴다.
  플레이어의 명령과 "적 AI 한 명령"은 행동 하나짜리 리플레이다. 재생 중에는 명령을 받지 않는다.
- 화면 상태(`unitVis`: 위치·높이·방향·동작·표시 병력)는 규칙 상태와 따로 두고, 재생이 끝나면 관측 상태로 맞춘다.

## 에셋

| 대상 | 원본 | 비고 |
|---|---|---|
| 병종·장수 13종 | `tools/spritetool/assets/ver2-units/` | ver3 7종 + ver4 추가 6종(사마·유비·간옹·장각·화웅·여포) |
| 지형 타일·장식 | `assets/terrain.png`, `objects.png`, `catalog.json` | ver3 타일셋 그대로 |
| 마을·성벽·성내 | `tools/spritetool/assets/ver4-structures/` | `tools/unit3d/structures.py`. 성벽 칸은 10px 높고 그 위 유닛도 올려 그린다 |
| 전투 맵 | `assets/content/campaign.json`의 `Tiles` | 에디터 지형 규칙(`ver3/tools/exportmap`)으로 경계·장식 생성 |
| 초상 | ver1 `assets/graphics/portraits.png`, `enemies.png` | 대사·정보창용 |

장수는 고유 도트(유비·관우·장비·간옹·장각·화웅·여포), 나머지는 병종 도트를 쓴다. 아군은 촉(금빛),
적은 B01 황건적, B02·B03 동탁·여포 진영색으로 바꾼다.

## 남은 일

- 전장은 맵을 2배로 그리고 병종은 원래 도트 크기로 그린다(`field.go`의 `K`). 병종·그림자가 맵 대비 절반 크기다.
- 공격이 빗나가면 대상이 막기 동작을 하고 방어막이 "팡" 하고 뜬다. 타격·화계·수계·회복·버프·상태 이상은
  입자·발광·지면 고리·빛기둥으로 그린다(`effects.go`). 큰 피해에는 화면이 짧게 흔들린다.
- 시나리오는 장면으로 진행한다(`assets/scenes.json`, `scene.go`): 작은 맵에 평복 장수들이 서서 걷고, 포권·건배·
  놀람·끄덕임을 하고, 말하는 장수 오른쪽 위에 말풍선이 뜬다. 클릭하면 연출을 넘긴다.
- 병법·아이템 이펙트는 맵 위 절차적 이펙트(화살·베기·화계·수계·회복·버프·상태)까지다. 전투 컷인 화면은 없고,
  일기토는 결과 패널만 보여 준다. 상점·배치 칸 선택은 코어 규칙이 없어 화면도 없다.
- 0.9.6 확장 중 시차만 넣었다. 읽기/쓰기 구분, 구간 단위 클레임은 없다.
- 실제 마우스·키 입력으로 캠페인 전체를 진행한 검수 기록은 아직 없다(`-shots` 자동 진행 캡처로 세 전투 완료까지 확인).
