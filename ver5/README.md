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
