# TODO: 새 도트·초상을 ver4 게임에 연결

작성: 2026-10-05. `/plan`으로 이 작업을 계획·진행하기 위한 메모다. 이 파일만 읽어도 시작할 수 있게 사실과 위치를 적었다.

## 목표

서장~1장(B01~B06, 대본은 `ver4/script-c1.md`)에 쓰려고 만든 전투 도트, 시나리오(평복) 도트, 수묵 담채 초상을 ver4 게임에 넣는다. 끝나면 게임에서 다음이 보여야 한다.

- 전투: 새 장수들이 고유 도트(또는 병종 변형 도트)와 진영색으로 나온다.
- 시나리오 장면: 평복 도트가 NW·SW만 있고 NE·SE는 좌우반전으로 그려진다. 출연자마다 진영색이 다르다(조조는 위 파랑).
- 대화창·편성·일기토: 장수 키별 수묵 초상이 나온다.

## 먼저 읽을 것

- `AGENTS.md`: 원본·런타임 위치 규칙. 일러스트는 `tools/illustrations/`, 도트는 `tools/spritetool/assets/`.
- `tools/spritetool/assets/ver2-units/README.md`: 도트 세트 목록(키, 외형, 공격), 평복 2방향 규칙, 진영색
- `tools/illustrations/README.md`: 초상 화풍, 장수 키, 파일 위치
- `ver4/README.md`, `ver4/scenario-system.md` 3절(장면 연출: 출연자 진영색, 2방향 도트), `ver4/ui-design.md`
- `.claude/skills/unit-sprite` 6단계: 게임 연결은 사용자가 요청했으므로 이번에 진행한다. 커밋은 요청이 있을 때만 한다.

## 현재 상태 (확인된 사실)

| 항목 | 위치 | 상태 |
|---|---|---|
| 전투 도트 원본 | `tools/spritetool/assets/ver2-units/<키>/` | 8방향 × 6동작. 새로 21종 |
| 평복 도트 원본 | `tools/spritetool/assets/ver4-civilians/<키>/` | **NW·SW 2방향만**(`frames.json`의 `directions`). 19종. 예전 8방향 PNG(N, NE, E, SE, S, W)가 일부 폴더에 남아 있으나 쓰지 않는다 |
| 진영색 표 | `tools/spritetool/assets/ver2-units/factions.json` | 위(밝은 파랑), 촉(**초록**, 금빛에서 바뀜), 오, 황건적, 동탁, 공손찬, 원소, 이민족 |
| 초상 원본 | `tools/illustrations/portraits/<키>.png` | 496×496, 36명. 유비는 `tools/illustrations/style-samples/4-ink-wash.png`(1254×1254) |
| 패키징 | `ver4/tools/build_assets.py` | 전투 도트는 `tools/unit3d/build.py`의 `UNITS`(=`looks.chosen()` 포함), 평복은 `civilian.chosen()`을 읽어 `ver4/assets/graphics/units/`와 `index.json`으로 묶는다. 초상은 아직 묶지 않는다 |
| GUI 도트 선택 | `ver4/internal/gui/view.go:21` `heroArt` | 유비·관우·장비·간옹·장각·화웅·여포만 고유 도트, 나머지는 `classArt`(병종 도트) |
| GUI 적 진영색 | `view.go:18` `stageFaction` | 스테이지 단위(B01 황건, B02·B03 동탁). 장수별 진영은 없다 |
| GUI 장면 출연자 | `internal/gui/scene.go:118-130` `newActor` | 모두 `faction: "shu"`, 기본 방향 `"S"` |
| 장면 데이터 | `ver4/assets/scenes.json` | `cast`·`face`에 정방향 `"S"`, `"N"`이 4곳 있다 |
| GUI 초상 | `view.go:253` `portrait()` | `portraits.png`(유비·관우·장비·간옹), `enemies.png`(여포·화웅·장각·황건) 4열 시트에서 잘라 쓴다. 그 외 인물은 황건 얼굴로 대체된다 |
| 도트 그리기 | `internal/gui/assets.go:288` `UnitArt.sheet(faction)`, `:341` `Frame(faction, dir, anim, i)` | 행(row)은 `rows`의 방향 순서. 없는 방향 처리 없음 |

## 할 일

### 1. 에셋 재패키징
- `python3 ver4/tools/build_assets.py`를 실행한다(Pillow, PyYAML, Go 필요).
- `index.json`에 다음이 들어갔는지 확인한다.
  - 새 전투 도트 21종
  - 평복 `civ_*` 19종. 평복의 `rows`는 `["NW","SW"]`여야 한다.
- 그림자 캐시(`shadows.yaml`)가 새 유닛도 만드는지 확인한다.

### 2. 평복 2방향 + 좌우반전
- `UnitArt.Frame`에서 `rows`에 없는 방향을 처리한다. NE는 NW, SE는 SW를 좌우반전해 그린다.
  - 반전은 그리기 단계(GeoM 스케일 -1)에서 하고, 기준점(pivot)도 반전에 맞춘다.
  - 전투 도트는 8방향이 다 있으니 영향이 없어야 한다.
- `scenes.json`의 정방향(`S`, `N`, `E`, `W`)을 대각(SW, NW, SE, NE)으로 바꾼다. `newActor`의 기본 방향 `"S"`도 `"SW"`로 바꾼다.
- 장면 데이터를 검증할 때 대각이 아닌 방향을 거부하는 테스트를 추가한다(`scenario_test.go` 등).

### 3. 장면 출연자별 진영색
- `scenes.json`의 `cast`에 진영을 줄 수 있게 한다. 예: `"조조": [x, y, "SW", "wei"]`. 생략하면 `shu`.
- 다른 방법: 장수 → 진영 표를 GUI에 둔다.
- 어느 쪽이든 `scenario-system.md` 3절과 맞춘다.

### 4. 장수 → 도트 매핑
- `heroArt`를 아래 표의 장수 전원으로 넓힌다.
- 적 진영색을 장수별로 정할 수 있게 한다. 지금은 스테이지 단위다.
  - B04 이후에는 원소군과 공손찬 우군이 한 맵에 나온다.
  - 지금 런타임 스테이지는 B01~B03뿐이므로, 표와 구조만 준비하고 B04~B06 데이터는 범위 밖이다.

### 5. 초상
- `build_assets.py`에서 `tools/illustrations/portraits/*.png`와 유비 기준 이미지를 256×256(제안)으로 줄여 `ver4/assets/graphics/portraits/<키>.png`로 낸다. `index.json`에 `portraits: {장수 이름: 파일}`을 넣는다.
- `portrait()`를 장수 이름 → 파일 조회로 바꾼다.
  - 초상이 없는 인물은 `scenario-system.md` 3절대로 이름표만 띄운다(황건 얼굴로 대체하지 않는다).
  - 이름 없는 황건병은 `turban_soldier`를 쓴다.
- 예전 `portraits.png`·`enemies.png`는 새 초상으로 대체되면 제거한다. 사용처를 모두 확인한 뒤에 지운다.
- 초상은 수묵 배경이 있는 정사각형이다. 대화창(`scenario.go:126-130`, 140px)과 편성창(60·120px)에서 잘림·비율을 확인한다.

### 6. 검증
- `cd ver4 && go test ./... && go vet ./...`
- `go build -o bin/srpg-cli ./cmd/srpg-cli && python3 tools/smoke_cli.py`: 규칙에는 영향이 없어야 한다.
- `go run ./cmd/srpg-gui -audit /tmp/shots -shots 600,1200`로 장면(도원결의 등)과 전투 화면을 캡처해 직접 본다.
  - 평복의 NE·SE 반전
  - 장비의 초록 진영색
  - 초상 표시
  - 대화창 레이아웃

### 7. 문서
- `ver4/README.md` 에셋 표: 도트 종수, 초상 출처, 진영색 변경
- 필요하면 `ver4/ui-design.md`와 `ver4/design.md` 4부의 초상·그래픽 항목

## 장수 이름 ↔ 키 (도트 · 평복 · 초상)

| 이름 | 키 | 전투 도트 | 평복 | 초상 | 진영(제안) |
|---|---|---|---|---|---|
| 유비 | liubei | ○ | ○ | 기준 이미지 | shu |
| 관우 / 장비 / 간옹 | guanyu / zhangfei / jianyong | ○ | ○ | ○ | shu |
| 손건 | sunqian | — (병종 strategist) | ○ | ○ | shu |
| 전예 | tianyu | ○ (고정 청색) | ○ | ○ | shu |
| 번궁 | fangong | ○ (무도가, 맨손 `jab`) | ○ | ○ | shu |
| 경무 / 관순 | gengwu / guanchun | ○ | ○ | ○ | shu |
| 장세평 / 소쌍 | zhangshiping / sushuang | ○ (상단, 공격 없음) | ○ | ○ | 미정(상단 중립색) |
| 한영 / 곽적 | hanying / guoshi | ○ | — | ○ | shu(우군) |
| 조운 | zhaoyun | ○ | ○ | ○ | gongsun |
| 공손찬 | gongsunzan | **없음(후보 미제작)** | ○ | ○ | gongsun |
| 엄강 | yangang | ○ | — | ○ | gongsun |
| 원소 | yuanshao | ○ (도보, 새 금빛 관모) | ○ | ○ | yuan |
| 문추 / 안량 / 장합 | wenchou / yanliang / zhanghe | ○ | 문추만 ○ | ○ | yuan |
| 순우경 / 국의 / 고람 | chunyuqiong / quyi / gaolan | ○ | — | ○ | yuan |
| 전풍 / 심배 | tianfeng / shenpei | ○ | — | ○ | yuan |
| 원술 | yuanshu | — | ○ | ○ | yuan |
| 조조 | caocao | — | ○ | ○ | wei |
| 동탁 / 이유 | dongzhuo / liru | 이유는 병종 strategist | ○ | ○ | dong |
| 여포 / 화웅 | lubu / huaxiong | ○ | — | ○ | dong |
| 이각 / 호진 | lijue / huzhen | — (병종 cavalry) | — | ○ | dong |
| 장각 | zhangjiao | ○ | — | ○ | turban |
| 정원지 / 등무 | chengyuanzhi / dengmao | ○ | — | ○ | turban |
| 황건병(이름 없음) | turban_soldier | — (병종 bandit) | — | ○ | turban |

## 범위 밖 (별도 작업)

- 시나리오 노드 시스템, 회의장, 상점, 전투 이벤트(`ver4/scenario-system.md`)
- B04~B06 스테이지 데이터, 우군 NPC 진영, 상단(비전투) 병종, 무도가 병종의 규칙 추가
- 공손찬 전투 도트 후보
- 이벤트 CG, 배경 일러스트(`tools/illustrations/events/`, `backgrounds/`)

## 정할 것

- 상단(장세평·소쌍)의 진영색: 지금은 진영 키 그대로다. 중립(베이지·갈색) 진영을 새로 둘지 정한다.
- 초상의 런타임 크기(256 제안). 대화창에서 얼굴 방향(일부 초상은 오른쪽이나 정면을 봄)을 맞출지, 그대로 둘지.
