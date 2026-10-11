# ver5

삼국지 SRPG. ver4를 정리한 판이다. 엔진(`internal/`)에는 인물·병종·아이템·스킬·특성 이름이 하나도 없고,
게임 내용은 모두 `assets/data/`의 데이터 묶음이다. 스프라이트는 도트 에디터 YAML을 빌드한 `.spr`만 쓰며
(픽셀별 색 태그), 진영색·옷색·말색은 캐릭터 데이터가 정한다. 특성은 `docs/trait-design.md`의 규칙형 특성이다.
설계는 [docs/ver5-design.md](docs/ver5-design.md).

```sh
cd ver5
go run ./cmd/srpg-gui                      # 그래픽 (1280×850)
go run ./cmd/srpg-cli                      # 사람용 CLI, -json 으로 JSON Lines
go run ./cmd/srpg-sim -seeds 10            # AI 대 AI 캠페인 시뮬레이션
go test ./... && go vet ./...
python3 tools/spritebuild/build.py         # tools/doteditor/units/*.yaml → assets/sprites/*.spr
(cd tools/doteditor && python3 -m http.server)  # 도트 에디터: http://localhost:8000
```

## 구조

```
ver5/
├─ cmd/srpg-{gui,cli,sim}
├─ internal/
│  ├─ content   데이터 묶음 로더·검증, 엔진 어휘(vocab.go)
│  ├─ core      규칙 엔진
│  ├─ ai, session, storage, cli, sim, replay
│  ├─ sprite    .spr 로더, 태그 팔레트, 그림자, 시트 캐시
│  ├─ gui       Ebitengine 화면
│  └─ testdata/mini   엔진 테스트용 최소 데이터 묶음 (다른 게임으로도 엔진이 돈다는 증거)
├─ assets/
│  ├─ data/       게임 데이터 (README.md 참고)
│  ├─ sprites/    .spr
│  ├─ graphics/   타일셋·구조물·소품·맵 그림 데이터·초상
│  ├─ scenes.json 장면 연출 (GUI 전용)
│  └─ fonts/
├─ tools/
│  ├─ doteditor/   도트 에디터와 units/*.yaml 원본
│  ├─ spritebuild/ yaml → .spr 빌드
│  └─ smoke_cli.py
└─ docs/
```

## 문서

| 문서 | 내용 |
|---|---|
| [ver5-design.md](docs/ver5-design.md) | ver5 목표, 데이터 묶음 구조, 엔진 변경 |
| [sprite-binary.md](docs/sprite-binary.md) | `.spr` 형식, 태그 색 규칙 |
| [trait-design.md](docs/trait-design.md) | 특성·책략 설계 (구현 대상) |
| [claim-replay.md](docs/claim-replay.md) | 클레임 리플레이: 규칙은 턴제, 화면은 동시 재생 |
| design.md, scenario*.md, script-c1.md, officers.md, ui-design.md | ver4에서 이어받은 설계·시나리오 문서 (일부 용어는 ver4 기준) |

## ver4와 다른 점

- 인물 = `Characters`. 병사는 `Template` 정의 하나를 스테이지가 여러 번 배치하고, 엔진이 번호 붙은 임시 인물을 만든다.
- 파티·패배 조건·편성은 `Lord`/`Playable` 속성, 시작 노드·다음 노드는 데이터(`Campaign.Start`, `Stage.Next`, 준비 노드의 `Stage`)다.
- 아이템은 `Effect`(`hp`/`mp`/`promote`), 스킬은 `Requires`(필요 특성), 특성은 `Effects`(엔진 효과 키)로 동작한다.
- 레벨업으로 특성치가 쌓이면 학습창이 열리고, 닫기 전까지 다른 명령은 받지 않는다(`Learning`).
- 스프라이트는 신형 7종(경보병·경기병·궁병·문사·유비·관우·장비)과 비전투 3종만 있다. 나머지 인물은 가장 가까운 병종 도트에
  진영색과 `Look.Palette`로 구분한다.
- 전투 시뮬레이터(skirmish)와 옛 ver2·ver3 스프라이트 체인은 가져오지 않았다.

## 맵에디터

`go run ./cmd/srpg-gui` 실행 후 시작 메뉴의 **맵에디터**에서 전투맵을 선택한다.
지면·구조물은 기존 배율의 절반으로 그리고, 게임 칸과 나무·유닛 크기는 유지한다.
대타일 하나는 소타일 2×2이며, 단독 최다 종류가 게임 지형이 된다(2:1:1도 변경).
2:2나 1:1:1:1은 이전 지형 판정을 유지한다. 외곽은 편집 가능하지만 전투에서는 어둡고 이동할 수 없다.

- 좌클릭 드래그: 칠하기. 타일 탭의 우클릭: 지형 가져오기.
- 브러시 단축키: `1`~`9`, `0`, `q`, 이후 `w e r t y…`. `Tab`: 타일/병력 탭 전환.
- 마우스 휠: 확대/축소. 우드래그·가운데 버튼·`Space`+좌드래그: 지도 이동. 지도 가장자리에 커서를 대도 이동한다. **영역 맞춤**: 전체 전투 영역 보기.
- 금색 네 꼭짓점: 전투 영역 크기 변경(각 변 4~64칸). 놓으면 항상 예/아니오를 묻는다. 예를 누르면 영역 밖 배치도 삭제한다.
- 병력 탭: 아군 배치 가능 칸 또는 적군 선택. 일반병은 연속 배치, 장수는 한 번만 배치 가능하다.
  기존 적을 클릭하면 병종·레벨·방향·현재 HP/MP·특성·장비를 편집한다. 우클릭은 제거한다.
  장수 선택에는 계열 필터와 진영별 초상화 표가 있으며, 없는 병종은 비활성화된다.
- `Ctrl`/`Cmd`+`Z`: 실행 취소. `Ctrl`/`Cmd`+`Shift`+`Z`: 다시 실행. `Ctrl`/`Cmd`+`S`: 저장.
- **저장**은 검증한 맵 문서를 교체하고 캠페인에 적용한다.
- **시험 전투**는 출전 편성부터 시작하며 캠페인 진행·저장 경로와 분리된다. 우상단 버튼으로 에디터에 돌아온다.

`assets/data/maps/<ID>.json` 한 파일을 에디터와 게임이 함께 사용한다.
대타일 판정과 화면용 지면은 로드 시 메모리에서 계산한다. 별도 내보내기나 빌드 명령은 없다.
새 ID로 저장한 맵은 에디터에서 시험할 수 있으며, 캠페인에 새 전투로 연결하려면 시나리오 노드를 별도로 추가한다.
같은 ID로 저장한 기존 맵은 이후 캠페인 전투부터 수정본을 사용한다.
규칙 버전과 콘텐츠 해시가 변경되므로 이전 자료로 만든 세이브·리플레이는 호환 검사를 통과하지 못한다.

```sh
go test ./internal/mapedit -cover
go test ./internal/mapedit -bench BenchmarkGraphics -benchtime 1000x -benchmem
```

맵 문서 형식과 저장 동작은 [docs/maps.md](docs/maps.md),
설계·검증 기록은 [_mapeditor.md](_mapeditor.md)에 있다.

브러시 크기는 소타일 기준 **1×1 / 2×2 / 4×4 / 8×8**로 선택한다.
방향은 실제 유닛 그림을 클릭해 **북동·남동·남서·북서**로 회전한다.
장수 팝업에서 병종과 진영별 인원수를 볼 수 있으며, 특성 창은 해당 장수의 보유 특성만 표시한다.
특성을 가리키면 설명이 나온다. 병종·장비로 부여되는 특성은 해당 병종·장비를 통해 변경한다.

전투에서는 피격과 방어 성공에도 상대 공격 경험치의 50%를 얻는다(`Rules.ExpDefenseShare`).
전투 로그는 한 화면 프레임에 한 항목씩 표시하며 다음 전투로 넘어가면 기록과 대기 큐를 비운다.
상단 드래그로 높이를 바꾸고, 휠 또는 스크롤바로 이전 기록을 볼 수 있다.
독·속박·봉격 등은 머리 위의 9×9 도트로, 강화·약화는 5×6 색상 화살표로 표시한다.
HP바→MP바→상태 아이콘 순서이며, 중독의 유닛 색조는 약하게만 적용한다.
공격은 빨강, 방어는 파랑, 정신은 초록, 순발·사기는 노랑, 이동은 흰색이다.
상태 원본·설정은 `../tools/spritetool/assets/ver5-status/`, 런타임은 `assets/graphics/status/`다.
`uv run --with pillow python tools/build_status_icons.py`로 수제 픽셀 원본을 재빌드한다.
성내는 병력 회복 없이 병법치 회복만 2배이며, 지형 정보에서 두 효과를 따로 표시한다.

```sh
SRPG_RENDER_TEST=1 go test ./internal/gui -run TestRenderedEditorAndStatuses -count=1
```

이 선택적 렌더링 테스트는 실제 창을 열고 `reports/editor-regression/`에 화면을 저장한다.

## 아군 배치

출전 준비에서 **출전**을 누르면 지도에서 배치할 수 있다. 아래 장수 카드를 선택한 뒤
밝은 빈칸을 클릭한다. 배치한 장수나 그 카드를 다시 클릭하면 배치를 해제한다.
출전 명단을 모두 배치한 뒤 **전투 시작** 또는 `Enter`/`E`로 전투를 시작한다.
고정 아군은 카드 목록과 배치 해제 대상에서 제외한다.

- `Tab`: 다음 미배치 장수. 우클릭/`Esc`: 선택 취소.
- 하단 휠·좌우 버튼: 장수 목록 가로 스크롤. 지도 드래그·확대/축소는 기존과 같다.
- `Space`: 배치 자동 진행 한 명령. `P`: 자동 진행.
- 배치 중에도 저장·불러오기가 가능하다. 배치 지면은 밝고 선택 장수의 빈칸만 1초 주기로 강조된다.
- CLI: `place 장수 x y`, `unplace 장수`, `deployment_finish`, `legal [장수]`.
  `auto`는 남은 장수를 배치하고 전투 시작 단계에서 멈춘다.

구현 계획과 검증 기록은 [아군 배치 계획](docs/plans/PLAN_ally-deployment.md)에 있다.

## 일기토 연출

전투 중 지정된 장수가 인접해 공격하면 전용 일기토 화면을 재생한다. B01의 장비–등무와 관우–정원지는 각각 다른 한 합 승부, B02는 화웅의 연타와 강타를 관우가 쉽게 막은 뒤 직선 추격해 베는 승부, B03은 장비·관우·유비 각각의 여포 일기토다. 장비는 대등한 격전, 관우는 강공, 유비는 버티기를 보이고 나머지 두 형제가 합세하자 여포가 퇴각한다. 감상 화면에는 건너뛰기 버튼만 표시한다. `Enter`·`Space`·`Esc`로도 일기토만 건너뛸 수 있고, `F`로 배속을 바꾼다. 종료 후 전투맵에서 사망·퇴각 등의 후속 결과를 보여 준다.

전개·대사·시간은 `assets/duels.json`, 규칙 결과는 기존 스테이지 데이터가 정한다. 시작 메뉴의 **일기토**에서 여섯 승부를 바로 감상할 수 있다. `go run ./cmd/srpg-gui -duel B01-duel-1`로 직접 열 수도 있다. 감상 모드는 캠페인 진행·저장 상태를 변경하지 않는다. [일기토 연출 문서](docs/duel-animation.md) 참고.

일기토 제작은 프로젝트 스킬 `.agents/skills/duel-authoring/SKILL.md`를 참고한다. 예제에 없는 동작도 이야기 요구에 맞게 확장한다. `srpg-gui`에서 `-duel-debug`를 지정하면 저작용 상황 설명과 조작을 표시한다.
