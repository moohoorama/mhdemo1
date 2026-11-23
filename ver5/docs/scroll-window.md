# 공통 스크롤 영역 설계

이 문서가 맡는 범위: 스크롤바가 있는 목록 화면의 입력(휠·본문 드래그·스크롤바·터치 관성)과, 자식 버튼 위에서 시작한 누름의 탭·드래그 구분.
승인된 시안은 `result/20261123/scroll-window-proposal/`(HTML 동작 시안, 게임 구현 아님)이다.

## 문제

- 목록 버튼은 누르는 순간 실행된다(`leftClick`). 출전 부대 카드처럼 자식 버튼이 있는 영역에서 누르면 드래그가 시작되지 않았다. 목록 행 버튼만 드래그를 시작했고 `listRows`는 `pick`이 없으면 행 버튼 자체를 만들지 않았다.
- 터치를 놓은 뒤 관성이 없었다.
- 목록마다 입력 처리가 달랐다(드래그 상태, 휠 처리, 스크롤 위치 갱신).

## 규칙

1. 스크롤 영역은 그릴 때 등록한다(`wheelArea`). 등록 시점의 버튼 번호 범위 `[from, to)`를 기록한다. 영역 안에 그려진 모든 버튼(머리글·배경·행·자식 버튼·스크롤바)이 이 범위에 든다.
2. 영역 배경은 버튼이 아니다. 어떤 버튼도 누름을 받지 않은 자리에서 누르면 `pressArea`가 영역의 드래그를 시작한다(버튼끼리 겹치지 않게 유지하는 `TestNoOverlappingButtons` 규칙 때문).
3. 누름이 영역 안의 버튼에 걸리면 `pressButton`이 버튼을 바로 실행하지 않고 `pressTable(l, rowH, click)`로 드래그 보류 상태를 만든다.
   - 손가락·마우스가 `dragStartPx`(8px)를 넘게 움직이면 드래그: 목록이 따라온다. 버튼은 실행되지 않는다.
   - 움직이지 않고 놓으면 `click`을 실행한다(짧은 누름 = 선택·부관 선택 등 기존 동작).
4. 스크롤바 조각(`grab`)은 누르는 즉시 동작한다(썸 드래그 또는 트랙을 누른 쪽으로 한 쪽 넘김). 보류하지 않는다.
5. 놓을 때 빠르게 움직이고 있었다면 관성이 남은 거리를 이어 간다(`tableList.fling`, 행/프레임). 매 프레임 마찰로 줄고, 목록 끝(`tableList.end`)이나 처음에 닿으면 멈춘다. 새 누름이 오면 관성을 끊는다.
6. 관성·드래그는 입력을 소비하지 않는다(`updateFlings`는 `false`). 관성 중 누름은 관성을 끊고 정상 누름으로 처리된다.
7. 전투 기록(`battle_log.go`)은 역순 목록(아래가 최신)이므로 본문 드래그 방향을 줄 위치 방향에 맞추고, 같은 관성 상수를 쓴다. 썸 드래그에는 관성을 주지 않는다.

## 변경 파일

- `ver5/internal/gui/table.go`: `tableList`에 `fling`·`end`, `tableDrag`에 속도 추적, `updateFlings`, `pressTable` 보류 규칙.
- `ver5/internal/gui/ui_kit.go`: `listRows`가 배경·행 버튼을 항상 등록하고 영역을 기록. `wheelArea`에 범위·행 높이.
- `ver5/internal/gui/skin.go`: `button.grab`.
- `ver5/internal/gui/game.go`: `leftClick`이 `pressButton`을 쓰고 빈 영역 누름은 `pressArea`, `drawScreen`이 `updateFlings`를 호출.
- `ver5/internal/gui/editor.go`, `duel_gallery.go`: 자체 누름 루프가 `pressButton`을 쓴다.
- `ver5/internal/gui/officer_cards.go`: 특성 칩의 `pressTable` 중첩 호출을 평범한 실행으로 바꾸고 영역 등록을 공통화.
- `ver5/internal/gui/battle_log.go`: 본문 드래그 관성.
- `ver5/internal/gui/waypoints.go`: 드래그 관성, 스크롤바 썸 드래그.
- `ver5/internal/gui/sortie_unified.go`, `platform_*.go`: 출전 부대 열 수(모바일 3·PC 5).
- `ver5/internal/gui/table.go`의 `tableBar`: 스크롤바 시안 선택(`scrollBarStyle`, 기본 현행).
- `ver5/internal/gui/uidemo_scrollbar.go`(새 파일): 스크롤바 시안 5종을 같은 출전 화면에서 캡처하는 데모.
- `ver5/internal/gui/scroll_test.go`(새 파일): 테스트.

## 알려진 한계

- 목록은 행 단위로 움직인다(픽셀 단위 위치 없음). 드래그 중 행이 칸 단위로 뛴다. 공통 방식을 유지하며 픽셀 오프셋 렌더링은 하지 않았다.
- 터치 관성의 속도·마찰은 60fps 기준 상수다. 실제 폰 감각은 기기에서 확인해야 한다.
- 휠 관성은 없다(기존 동작 유지).
