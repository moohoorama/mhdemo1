# 장면 이동 막별 분류와 앞선 전투 결과 잘림 수정 결과

## 요약

1. 장면 이동 왼쪽 목록을 막 탭 **1막 / 2막 / 2막-연의 / 2막-가상**으로 나눴다. 분류는 `scenario.json` 각 장의 `Act` 데이터에서 읽는다.
2. 오른쪽 "앞선 전투 결과"(22건)가 창 밖과 닫기 버튼 위로 잘리던 문제를 스크롤 영역(휠·본문 드래그·스크롤바 썸·관성)과 줄바꿈으로 고쳤다. 모두/없음·개별 토글·이동 시 플래그 적용은 유지했다.

설계 기록: `ver5/docs/waypoint-groups.md` (구현 중·후에 작성했다. 지시는 "먼저 작성"이었으나 순서는 지키지 못했다).

## 분류 기준 (실제 데이터 확인)

- 장 구성: `scenario.json` 순서 01~27, 25v, 26v, 27v (battle 노드는 장마다 하나).
- `fullstory.md`: 공통부 1~24장, 「1막 난세도래」는 16장 끝에서 닫힘(17장부터 2막), 연의·가상 갈래는 24장 융중 선택(`c24-mode`)과 25장 `c25-mode`에서 갈림.
- 결과: 01~16 = 1막, 17~24 = 2막, 25~27 = 2막-연의, 25v~27v = 2막-가상. 지점 60곳(장 시작 30 + 전투 30) 모두 정확히 한 막에 들어간다.
- 판단 사항: 25장 첫머리(c25-sick-news~c25-b08)는 두 갈래가 함께 지나지만, 장 시작 지점으로 이동하면 `flag-mode-virtual`이 없는 연의 쪽 25장으로 들어가므로(`core.NewAtWith`, 기존 `TestJumpIntoTheOtherTelling`이 같은 전제) 25장 시작을 2막-연의에 뒀다. 이 배치가 마음에 들지 않으면 알려 주세요.
- 이동 동작 자체는 바꾸지 않았다(`jump`, `core.NewAtWith` 그대로) — 잘못된 분기로 가는 경로가 새로 생기지 않는다.

## 변경 파일

- `ver5/assets/scenario/hero/scenario.json` (각 장에 `Act` 30줄)
- `ver5/internal/content/content.go` (`Chapter.Act`), `ver5/internal/content/scenario.go` (`Waypoint.Act`, `Data.WaypointActs()`)
- `ver5/internal/gui/waypoints.go` (막 탭, `scroller` 공통화, 결과 영역 스크롤·줄바꿈·`n/22`, 결과 행은 놓을 때 토글)
- `ver5/internal/gui/game.go` (메뉴에서 열 때 `g.waypoints = waypointList{}` 한 줄)
- `ver5/internal/gui/uidemo_waypoints.go` (두 목록 스크롤, 2막-가상 탭 전환)
- 새 파일: `ver5/internal/content/waypoint_acts_test.go`, `ver5/internal/gui/waypoints_test.go`, `ver5/docs/waypoint-groups.md`
- `ver5/docs/history.md` (항목 추가)

다른 세션의 미커밋 변경(game.go 등 기존 수정 hunk 포함)은 건드리지 않았다. `validate.go`는 `Act` 필수 검사를 넣었다가 조각 시나리오 테스트 14건이 깨져 되돌렸다(변경 없음).

## UI 동작

- 막 탭은 현재 막이 금색 밑줄. 막마다 스크롤 위치를 기억한다. 메뉴에서 창을 새로 열면 1막·맨 위로 초기화(기존 `top = 0`과 같은 규칙). 난이도 창을 열었다 닫는 왕복은 위치를 유지하도록 만들었으나 화면으로는 확인하지 못했다(아래).
- 결과 목록: 레벨 줄·`앞선 전투 결과 n/22`·모두/없음은 고정, 그 아래만 스크롤. 문구는 칸 너비에 맞춰 줄바꿈(예: 조인과 싸우기 전에 … 두 줄), 행 높이가 늘어난다.
- 두 스크롤 영역은 각자 자기 영역에서 시작한 누름만 받는다(목록 드래그가 결과를 건드리지 않음, 그 반대도 마찬가지).

## 테스트 결과 (실제 실행 그대로)

- `go test -p 1 -run TestWaypointActs ./internal/content` → ok
- `go test -p 1 ./internal/content` → ok (첫 실행에서 `Act` 필수 검사 때문에 14건 실패, 검사를 되돌린 뒤 ok)
- `go test -p 1 -run 'TestWaypoint' -v ./internal/gui` → 전부 PASS (TestWaypointActTabs, TestWaypointFlagsReachable, TestWaypointJumpUsesSetFlags, TestWaypointWrapLines; 데스크톱 1280x850·폰 2400x1080 두 화면)
- `go test -p 1 -run 'TestNewAt|TestJump' ./internal/core` → ok
- `go test -p 1 -run 'TestScroll|TestSortie|TestNoOverlapping|TestBrowser' ./internal/gui` → ok
- `go test -p 1 ./internal/session ./internal/autoplay` → ok
- `go build ./...` → 성공 (`go vet ./internal/...`는 출력 없음)

테스트가 검증하는 것: 막별 분류와 경계(c16-battle=1막, c17-walk=2막, c24-battle=2막, c25-battle·c27-redcliff-battle=연의, c25v-start·c25v-battle·c27v-battle=가상), 지점 누락·중복 없음, 막마다 마지막 행까지 스크롤해 눌러 마지막 지점이 선택되고 난이도 창이 열림, 막별 스크롤 기억, 결과 22건이 스크롤로 전부 도달·각 행이 자기 플래그를 토글, 모두/없음, 결과 영역 드래그가 토글하지 않음, 썸 드래그가 결과만 움직임, 줄바꿈이 글자를 잃지 않고 폭 안에 듦, 결과 행을 눌러 켠 플래그로 `jump` 하면 고순이 합류하고 누르지 않으면 합류하지 않음.

## 화면 캡처 (`cmd/_uidemo-vm -ui waypoints`)

- 수정 전(결과가 닫기 버튼 위까지 잘림): `before/pc/frame-0040.png`
- 수정 후 PC 1280x850: `after/pc/frame-0040.png`(1막, 탭·결과 위쪽), `frame-0119.png`(두 목록 끝까지 스크롤, 마지막 항목이 영역 안), `frame-0140.png`(2막-가상 탭), `after/pc-wrapped/frame-0075.png`(줄바꿈 행)
- 수정 후 폰 2400x1080 배율 130: `after/phone130/frame-00{40,119,140}.png`
- 수정 후 폰 2400x1080 배율 160: `after/phone160/frame-00{70,75,80}.png`
- 직접 본 것: PC 4장(수정 전 1·후 3 + 줄바꿈 1), 폰 130 한 장(119), 폰 160 한 장(75). 나머지는 같은 데모 프레임이라 열어 보지는 않았다.

## 미확인·한계

- 실제 기기 터치(관성 체감, 두 영역 동시 터치), 마우스 휠은 확인하지 못했다. 헤드리스 캡처와 포인터 주입 테스트뿐이다. 휠은 기존 코드와 같은 방식(영역 위 40px/칸)이지만 테스트하지 않았다.
- 난이도 창을 열었다 취소하고 돌아왔을 때 탭·스크롤이 남는지는 화면으로 확인하지 않았다(코드상 상태는 `overlays` 스택 위에 유지됨).
- 폰 배율 160 캡처가 130과 레이아웃 배율이 달라 보이지 않는다(같은 폭에 보임). 배율이 실제로 적용되는지 `-uiscale` 쪽은 확인하지 않았다.
- `TestStoryDrawsFromEveryWaypoint`(전체 장면 재생)와 `internal/gui` 전체 테스트는 실행하지 않았다(8GiB 맥 기준 하나씩 돌리는 규칙, 시간). 이전 결과 보고에 있던 다른 세션 데이터 때문의 실패 2건(`TestScenarioNodesAreStaged`, `TestStoryManagementDeputy`)은 이번에 다시 보지 않았다.
- 체크박스 선 모양은 기존과 같은 `strokeRect` 반 픽셀 그리기라 캡처에서 일부 변이 흐리게 보인다(수정 전 캡처도 동일).
- 커밋·푸시는 하지 않았다.
