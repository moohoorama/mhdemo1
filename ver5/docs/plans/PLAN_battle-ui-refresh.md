# 전장 UI 개편 (docs/ui-mockup 적용)

시안 `docs/ui-mockup/index.html`의 확정 항목을 기능 하나에 커밋 하나로 넣는다. 마음에 안 드는 기능은
그 커밋만 `git revert`로 되돌린다. 그래서 커밋끼리 서로 의존하지 않게 하고, 의존이 필요하면 앞 커밋에 둔다
(폰트, 데모 하네스).

## 커밋 순서

| # | 커밋 | 주로 고치는 곳 |
|---|---|---|
| 0 | docs: 시안과 이 계획 | docs/ui-mockup, docs/plans |
| 1 | 붓·명조 폰트 + `cmd/_uidemo` 하네스 | assets/fonts, skin.go, gui/uidemo.go, cmd/_uidemo |
| 2 | 턴 전환: 붓질 + 붓글씨 턴 + 진영색 밑줄 | game.go(banner), player.go, 새 turn_banner.go |
| 3 | 피해 숫자: 붓글씨, 먹 튐, 치명 고리, 회복 빛 | player.go(say/strike), 새 damage_popup.go |
| 4 | 체력 바: 유닛 폭 기준, 4칸 눈금, 흰 잔상 | field.go(unit bar), player.go(unitVis) |
| 5 | 명령 메뉴: 위에서 펼침, 아이콘, 금빛 강조, 하위 목록 병법치·설명 | command.go(drawMenu) |
| 6 | UI 크기 설정 80–160% (모바일 기본 130%) | skin.go(uiScale), windows.go(settings), session settings |
| 7 | 맵 밖 타일: 어둡게 한 풀 + 소품, 안개, 경계 테, 구름 그림자 | field_native.go(background/exterior) |
| 8 | 상단 바: 옻칠 바, 턴 원패, 승패 두루마리, 세로 이름패 | hud.go(topBar) |
| 9 | 이름패 누르면 그 장수 가운데로 + 선택 | hud.go, view.go(카메라 이동 보간) |
| 10 | 적 차례 카메라: 동시에 움직이는 적들 가운데에 가까운 적 | player.go, view.go |
| 11 | 대화창: 도트 부품 두루마리 (펼침 easeOutBack, 접힘 easeInCubic) | scenario.go, scene.go, assets/graphics/ui/scroll |
| 12 | 일기토 컷인: 가운데 띠, 사선으로 맞물리는 초상, 이름 찍기 | duel_scene.go, 새 duel_cutin.go |
| 13 | 장수 첫 등장: 먹 번짐 초상 + 이름·자·소개 | scene.go, scenes.json 단계 |
| 14 | 전투 로그 글자 색 (피해 주색, 회복 초록, 장수 이름) | battle_log.go |
| 15 | 대상 고르는 단계에서 대상 아닌 칸 클릭 = 취소 | command.go(mapClick) |

순서는 바뀔 수 있다. 각 커밋은 `go test ./internal/gui/... && go vet ./...`가 통과해야 한다.

## 되돌리기

각 기능 커밋은 `git revert <커밋>` 하나로 빠지도록 만든다.

- 공용 도우미(`ui_anim.go`의 이징·키프레임·글자 이미지·줄바꿈, `camera_pan.go`의 카메라 이동 보간, 폰트,
  `cmd/_uidemo` 하네스)는 1번에 모은다.
  1번은 다른 기능 모두의 바탕이라 되돌리지 않는다.
- 기능마다 `Game` 필드와 `advance()` 호출을 서로 떨어진 줄에 둔다. 데모는 기능별 `uidemo_<이름>.go`에
  `init()`으로 등록한다. 그래서 옆 기능과 같은 줄을 건드리지 않는다.
- 위에 지은 기능은 함께 되돌린다: **6(UI 크기)은 5(명령 메뉴) 위에**, **9(이름패 누르기)는 8(상단 바) 위에**
  있다. 5를 빼려면 6, 5 순으로, 8을 빼려면 9, 8 순으로 revert 한다.

## 폰트

Ebitengine `text/v2`는 폰트를 주지 않고 TTF/OTF를 읽기만 한다. 지금은 `assets/fonts/NotoSansKR.ttf` 하나다.

- 붓글씨: Nanum Brush Script (OFL). 턴 글자, 피해 숫자, 일기토 이름, 장수 첫 등장 이름.
- 명조: Noto Serif KR Bold/Black (OFL). 상단 바 제목, 두루마리 대사, 이름패.
- APK 크기 때문에 `pyftsubset`으로 KS X 1001 한글 2,350자 + 영숫자·문장부호만 남긴다.
  스크립트는 `tools/subset_fonts.py`, 원본 다운로드 경로와 명령을 README에 적는다.

## cmd/_uidemo 와 검증

`go run ./cmd/_uidemo -ui <이름> -out <dir> [-frames N] [-every K]`

- `internal/gui/uidemo.go`가 실제 `Game`을 만들고(`editorTrial`로 B01 전투에 바로 진입) 기능별 준비 함수로
  상태를 맞춘 뒤, 매 프레임 `advance(1/60)` + `Draw`를 하고 K프레임마다 PNG로 저장한다.
  게임과 같은 코드 경로를 그대로 쓰므로 데모가 맞으면 게임도 맞다.
- 완료 조건: 데모 프레임을 시안의 같은 순간(헤드리스 Chrome 캡처)과 나란히 놓고 배치·색·타이밍이 같은지 본다.
  다르면 고치고 다시 뽑는다. 같아진 뒤에 커밋한다.
- `_` 접두어라 `go build ./...`와 테스트 대상에서 빠진다(확인용 임시 도구 규칙).

## 하지 않는 것

- 전투 규칙·저장 형식은 건드리지 않는다. UI 크기 설정은 세션 설정(기기별)에만 둔다.
- 한자는 화면에 쓰지 않는다.
