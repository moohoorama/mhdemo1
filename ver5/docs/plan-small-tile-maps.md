# 계획: 모든 장면 맵을 작은 칸 규칙으로 (2026-11-14)

사용자: "전부 작은칸 단위로 렌더링되게, 그 규칙이 어긋난 부분 없이 잘 렌더링 되도록, 깔끔하게 만들자."
목표는 장면 맵 하나하나가 작은 칸 데이터(땅 글자·높이·물체·성벽 줄)에서 한 벌의 규칙(`internal/scenemap`, `assets/graphics/scene.json`)으로만 그려지는 것이다. 맵마다의 예외 코드는 없앤다.

## 1. 형식 1 장면 맵 목록 (37개)

| 맵 | 지금 쓰는 곳 | 형식 2 사본 | 특이 글자 |
|---|---|---|---|
| S_army_camp | 2·14장 장면 | | |
| S_castle_out, S_castle_out_open | 2·4·12장 장면 | | `b` 해자 다리, 성벽 줄 |
| S_castle_town_burnt | (빌드 스크립트·문서만) | | 성벽 줄 |
| S_checkpoint_closed | (빌드 스크립트·문서만) | | 높은 땅, 성벽 줄 |
| S_council | 3·5·6·7·11·15·17장 장면 | S_council | |
| S_garden | 17장 장면 | | |
| S_hall, S_house, S_inn, S_market, S_office, S_palace, S_jail_street (S_pass는 11-14 삭제) | 문서·테스트·빌드 | | S_jail_street 성벽 줄 |
| S_mansion, S_tavern, S_tent, S_tent_lux, S_tool_shop, S_weapon_shop, S_zhuo_office, S_zhuo, S_peach_grove(11-14 삭제: 안 쓰는 S_peach_grove 사본) | zhuo.py·compose.py·테스트 | 각 `…2` | S_zhuo 성벽 줄 |
| S_road | 3·5·7장 장면 | | |
| S_vil_burnt, S_vil_peace | 테스트·빌드 | | |
| S_camp, S_peach, S_village | 테스트(`scene_maps_test.go`), `uidemo_scenemap.go` 설명 | | 작은 칸 데이터 없음(옛 비네이티브 맵) |

S_camp·S_peach·S_village는 어느 장면도 쓰지 않는다. 변환하지 않고 지웠다(사용자 11-14). S_hill도 쓰는 장면이 없어 함께 지웠다.

## 2. 변환

- `tools/scene_maps/scene_v2.py`(옛 맵 → 형식 2: 땅 글자, 높이, 물체 기준점, 성벽 기둥 → `swall` 줄, 성문 → 성문 물체)를 제자리(같은 ID)로 돌린다. 장면·탐험 데이터의 맵 ID는 바뀌지 않는다.
- 특이 글자는 형식 2에 이미 있는 규칙으로 옮긴다. 규칙이 없는 것은 새 규칙(그림 포함)을 만들거나, 맞출 수 없으면 사용자 결정으로 남긴다.
  - `b` 다리: 작은 칸마다 다리 판(작은 칸 크기 그림, 기존 `bridge_u/v`를 반으로 줄인 것과 같은 크기)
  - `c` 성벽 블록: 성벽 줄(`swall`)
  - `h` 절벽: 높이 규칙
- 묶음마다 옛/새 전체 캡처를 비교해 다른 점을 고친다.
- 형식 2 사본이 있는 맵(S_zhuo/S_zhuo 등)은 둘 다 형식 2가 되어 내용이 겹친다. 하나로 합칠지는 사용자 결정.

## 3. 판 단면을 칸 규칙표로

지금의 `diorama.py`는 맵마다 그림 묶음을 굽는다(맵별 json·png, 신선도 검사, `_walkmap` 훅). 이것을 규칙표로 바꾼다.

- `scene.json` `board`:
  - `sections`: 땅 글자 → 단면 집안(family). 그림은 `<family>_pu_<k>`·`<family>_pv_<k>`(작은 칸 하나 폭, 네 갈래, 칸마다 해시로 고름). 높은 칸 밑은 `under`.
  - `ends`: 판 양 옆 끝의 테두리 한 줄.
  - 깊이는 모든 집안이 같다(56px).
- 구조물 단면:
  - 성벽 집안(`walls`)의 `cut`: 앞 변에 닿은 끝 칸에 `<cut>_pu`·`_pv`
  - 물체 성벽(`pwall_u` 등)은 `scene.json` `cuts`(물체 이름 → 단면 그림)
  - 높이 규칙의 `cut`: 성벽 통로가 앞 변에 닿을 때, 줄의 처음/가운데/끝 칸 단면
- 규칙은 `scenemap.Build`가 칸마다 놓는다. 단면은 평면층 끝(바닥과 함께 한 번 굽힘), 구조물 단면은 그 조각의 그림에 붙는다. 게임과 뷰어가 같은 계획을 그린다.
- 그림은 `tools/scene_maps/diorama.py`가 화소 키트로 그려 `assets_candidate/tiles`에 쓰고, `tools/build_extra.py`가 `map/extra.png`에 싼다.
- 맵별 판 파일(`maps/board/`), `field_board.go`의 판 불러오기·그리기, 신선도 테스트, `_walkmap` 굽기 훅은 지운다.

## 4. 형식 1 그리기 길 정리

장면 맵이 모두 형식 2가 되면 다음을 지운다.
- 장면용 형식 1 코드: `index.json maps`의 형식 1 읽기, `small_props`·`small_faces`·`highland`·`small_snap`·`cliff_repeat`, `trimToBoard`, 형식 1 성벽 기둥 그림자 대체
- 형식 1 빌드 스크립트: `tools/scene_maps/build.py`·`compose.py`·`export.mjs`, 그것들만 쓰는 도우미

전투 맵(`content.MapDocument.Graphics`)이 쓰는 네이티브 그리기는 남긴다.

## 5. 순서

1. 옛 캡처(37개 전체 보기)
2. 변환 묶음 1(방 15개) → 묶음 2(야외 평지) → 묶음 3(성벽·다리·절벽). 묶음마다 새 캡처와 비교
3. 규칙표·단면 그림·plan 연결, 맵별 판 지우기
4. 형식 1 코드·스크립트 지우기, 테스트(`go test ./internal/gui/ ./internal/scenemap/ ./internal/content/`, `TestSceneWalksStayOnOpenGround`)
5. 문서(`presentation-scene.md`, `tools-build.md`, `history.md`, 스킬 표), 비교 페이지

S_courtyard·S_rampart·S_rampart_siege도 쓰는 장면이 없어 지웠다(사용자 11-14).
S_army_camp_hungry·S_turban_camp도 쓰는 장면이 없어 지웠다(사용자 11-14).
