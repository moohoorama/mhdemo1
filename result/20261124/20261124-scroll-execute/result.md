# 공통 스크롤·출전 배치 구현과 스크롤바 시안 5종 결과

## 요약

- 공통 스크롤 입력(드래그·관성·자식 버튼 탭/드래그 구분)을 구현하고 목록 화면에 연결했다.
- 출전 부대 카드: 모바일 3열, PC 5열. 실제 레이아웃 테스트로 두 빌드 태그에서 확인.
- 스크롤바 시안 5종(+ 현행 참고)을 실제 게임 화면에서 캡처해 비교 페이지를 만들었다. **최종 채택 안 함.**
- 전체 `internal/gui` 테스트: 두 번 실행, 두 실패(`TestScenarioNodesAreStaged`, `TestStoryManagementDeputy`)는 이번 변경과 무관해 보이는 다른 세션의 미커밋 스토리·장면 데이터 문제로 판단(확인 한계 아래 참고).
- 커밋 ID와 push 여부는 아래 "커밋" 절 참고.

## 설계

- `ver5/docs/scroll-window.md` (설계 문서, 구현과 맞춤)
- 핵심: 스크롤 영역(`wheelArea`)이 자기 버튼 범위를 기록한다. 영역 안 버튼을 누르면 바로 실행하지 않고 드래그 보류 상태로 두었다가, 움직이지 않고 놓으면 탭(실행), 8px 넘게 움직이면 스크롤로 처리한다. 빠르게 놓으면 관성이 남은 거리를 이어 간다.

## 변경 파일

- `ver5/internal/gui/table.go` — tableList의 fling/end, 드래그 속도 추적, updateFlings, 스크롤바 grab 버튼
- `ver5/internal/gui/ui_kit.go` — wheelArea 범위·행 높이, beginScroll/endScroll, scrollAreaOf, pressButton, runButton, pressArea, listRows 재작성
- `ver5/internal/gui/skin.go` — button.grab
- `ver5/internal/gui/game.go` — leftClick이 pressButton/pressArea 사용, drawScreen에서 updateFlings, 전투 기록 관성 필드
- `ver5/internal/gui/editor.go`, `duel_gallery.go` — 자체 누름 루프가 pressButton 사용
- `ver5/internal/gui/officer_cards.go` — 특성 선반 영역 공통화, 칩의 중첩 pressTable 제거
- `ver5/internal/gui/battle_log.go` — 본문 드래그 관성 (glideLog)
- `ver5/internal/gui/waypoints.go` — 드래그 관성, 스크롤바 썸 드래그
- `ver5/internal/gui/sortie.go` — sortieSheetSize(출전 시트 폭), 열 수에 맞는 그리드 폭
- `ver5/internal/gui/sortie_unified.go` — sortieColumns(avail), 그리드 열 수
- `ver5/internal/gui/platform_desktop.go` — sortieCols = 5
- `ver5/internal/gui/platform_mobile.go` — sortieCols = 3
- `ver5/internal/gui/scrollbar_style.go` (새 파일) — 스크롤바 6종(현행 + A~E)
- `ver5/internal/gui/uidemo_scrollbar.go` (새 파일) — 시안 비교 캡처 데모 (`-ui scrollbar`)
- `ver5/internal/gui/uidemo_insu.go` — pressLabel이 손을 뗀 뒤 탭을 실행
- `ver5/internal/gui/browser_test.go` — 행 누름을 pressButton으로 바꿈
- `ver5/internal/gui/scroll_test.go` (새 파일) — 테스트
- `ver5/docs/scroll-window.md` (새 파일) — 설계 문서
- `ver5/docs/history.md` — 항목 추가
- `result/20261124/20261124-scroll-execute/` — 이 결과 폴더

변경하지 않은 것: `ver5/internal/gui/uidemo_zz_tmp_*.go`(다른 세션), 다른 세션의 assets·tools·core 미커밋 변경.

## 화면별 스크롤 감사

| 화면 (코드) | 휠 | 스크롤바 드래그 | 본문 드래그 | 터치 관성 | 자식 버튼 위 드래그 / 짧은 탭 | 비고 |
|---|---|---|---|---|---|---|
| 출전 부대 카드 (sortie_unified listRows) | ○ | ○ | ○ | ○ | ○ / ○ | 카드·부관 메달·교체 버튼 포함 |
| 출전 장수 표 (drawTable) | ○ | ○ | ○ | ○ | ○ / ○ | 머리글은 고정, 머리글 위 드래그도 스크롤 |
| 부관 창 (deputy_window drawTable) | ○ | ○ | ○ | ○ | ○ / ○ | 코드 경로 동일, 화면 테스트는 아래 참고 |
| 장수 명부 (officer_window roster·officerList) | ○ | ○ | ○ | ○ | ○ / ○ | 코드 경로 동일 |
| 장비 창 목록 (equipment listRows) | ○ | ○ | ○ | ○ | ○ / ○ | 코드 경로 동일 |
| 상점 (shop drawTable) | ○ | ○ | ○ | ○ | ○ / ○ | updateShopList의 휠은 중복이나 무해 |
| 소지품·인수 (insu drawTable) | ○ | ○ | ○ | ○ | ○ / ○ | 코드 경로 동일 |
| 물자 (party goods drawTable) | ○ | ○ | ○ | ○ | ○ / ○ | 코드 경로 동일 |
| 저장 창 (save_window drawTable) | ○ | ○ | ○ | ○ | ○ / ○ | 코드 경로 동일 |
| 지도 목록 (browser listRows) | ○ | ○ | ○ | ○ | ○ / ○ | TestBrowserLists 통과 |
| 배치 목록 (deployment listRows) | ○ | ○ | ○ | ○ | ○ / ○ | 코드 경로 동일 |
| 대결 갤러리 (duel_gallery listRows) | ○ | ○ | ○ | ○ | ○ / ○ | 자체 누름 루프도 pressButton 사용 |
| 편집기 선택 목록 (editor listRows) | ○ | ○ | ○ | ○ | ○ / ○ | 자체 누름 루프도 pressButton 사용 |
| HUD·명부 장수 표 (unitTable drawTable) | ○ | ○ | ○ | ○ | ○ / ○ | 휠은 HUD 본문 위에서 동작 |
| 장수 카드 특성 선반 (officer_cards shelfCards) | ○ | ○ | ○ | ○ | ○ / ○ | 칩 탭은 놓을 때 실행 |
| 전투 기록 (battle_log, 역순) | ○ | ○ | ○ | ○ (본문 드래그만) | — | 썸 드래그에는 관성 없음, 최신 줄 기준 역순 유지 |
| 경유지 목록 (waypoints) | ○ | ○ (신규) | ○ | ○ (신규) | 놓을 때 행 선택 | 스크롤바 썸 드래그를 새로 추가 |

확인 수준: 공통 경로는 단위 테스트와 실제 레이아웃 테스트로 확인. 화면별로는 코드 경로가 같다는 것까지만 확인했고, 각 화면을 실제로 조작한 자동 테스트는 출전 화면·지도 목록·편집기 목록·장수 특성 탭(TestSortieDeputy)뿐이다. 경유지 화면은 자동 테스트가 없다.

## 출전 열 수 (빌드 태그별)

- 데스크톱 빌드(기본 태그): 2400×1080 @130%, 1280×850 @100% 모두 5열 (TestSortieCardColumnsOnScreen)
- 모바일 빌드(`-tags mobile`): 2400×1080 @130%, 1280×850 @100% 모두 3열 (같은 테스트, 모바일 태그로 실행)
- 좁은 창: `sortieColumns(avail)`가 가용 폭에 맞춰 열을 줄인다(테스트 TestSortieColumns). 화면 캡처는 좁은 창에 대해 찍지 않았다.
- 기존에는 `leftMax = x + w*3/10`으로 2열이 나왔고, 이를 바꾸며 출전 시트 폭을 PC 1135·모바일 960 이상으로 넓혔다. 세부 치수는 `sortie.go`의 `sortieSideW`, `sortieDetailW`에 있다.
- `sortieCols`는 빌드 태그마다 한 번만 선언되며 두 빌드 모두 컴파일된다.

## 스크롤바 시안 5종 (+ 현행 참고)

시안 자체는 같은 출전 화면(제25장 c25-battle, 장수 표 스크롤 상태)에서 `scrollBarStyle` 값만 바꿔 찍었다. 실제 게임 화면이며 모형은 없다. 최종 채택 아님.

- 현행(참고): 반투명 회색 홈, 밋밋한 손잡이.
- A 죽간: 둥근 홈 위 베이지 알약 손잡이, 결 두 줄.
- B 금테 홈: 금색 테두리 홈, 밝은 금색 손잡이.
- C 그립: 가는 홈, 세 줄 결의 넓은 손잡이, 그림자.
- D 눈금: 5px 간격 눈금 트랙, 금색 블록 손잡이.
- E 먹: 먹색 손잡이, 왼쪽 금테, 먹 번짐 홈.

캡처:
- PC 1280×850: `capture/desktop/frame-0025.png` (현행), `-0055` (A), `-0085` (B), `-0115` (C), `-0145` (D), `-0175` (E)
- 휴대폰 참고 2400×1080 @130%(모바일 게스트 빌드 `UIDEMO_TAGS=mobile`): `capture/phone/frame-0025.png` … `-0175.png`
- 확대 비교: `capture/desktop/strip-table-bar.png`, `capture/phone/strip-table-bar.png`
- 비교 페이지: `scrollbar-options.html` (같은 폴더)

## 테스트 명령과 결과

- `go build ./...` (데스크톱): 통과
- `go build -tags mobile ./internal/gui/`: 통과
- `go vet ./internal/gui/` (데스크톱·모바일): 통과
- `go test -p 1 -count=1 -run 'TestScroll|TestSortie|TestTable|TestBrowserLists|TestNoOverlappingButtons|TestEditorChoiceLists|TestSortieDeputy' ./internal/gui`: 통과 (20초)
- 같은 명령 `-tags mobile`: 통과 (22초)
- `go test -p 1 -count=1 ./internal/gui` (전체, 데스크톱): 2회 실행. 1회차 실패 6개 → 수정 후 2회차 실패 2개: `TestScenarioNodesAreStaged`(c23-xinye-arrival, c25-sick-news에 장면 없음), `TestStoryManagementDeputy`(c13-wait-zhaoyun에서 WrongPhase). 두 테스트 모두 장면·스토리 데이터(다른 세션의 미커밋 `assets/scenes`, `story13.json`, `explore13_farewell.json` 등)와 관련되며 이번 입력 변경과 직접 관련된 호출이 아니다. 다만 HEAD 기준 비교는 못 했다(HEAD 자체가 다른 세션의 `core` 미커밋 변경 없이는 컴파일되지 않음).
- 전체 테스트를 `-tags mobile`로는 돌리지 않았다(대상 테스트만 실행).

## 실제로 확인하지 못한 것

- 실제 휴대폰 기기의 터치 감각·관성 느낌. 관성 상수(마찰 0.92, 시작 속도 3px/프레임)는 60fps 기준 추정치다.
- 실제 터치 입력: 테스트는 포인터 상태를 직접 설정했다. 손가락 탭과 드래그의 실제 경계(8px)는 기기에서 확인하지 않았다.
- 휠 관성은 없다(기존 동작 유지).
- 경유지 화면 드래그·썸 드래그의 자동 테스트는 없다. 코드만 추가했다.
- 좁은 창의 출전 열 수 캡처는 찍지 않았다(테스트만 있음).
- 두 개의 전체 테스트 실패(위 참고)는 원인을 단정하지 못했다.
- 시안 5종 중 무엇을 쓸지는 사용자가 정한다. 어느 것도 확정하지 않았다.

## 커밋

(아래 커밋 절은 최종 단계에서 채운다)
