# 출전 카드 축소와 도트 스크롤바 시안 5종 — 결과

요청: `process/20261124-scroll-sortie-pixel/20261124-scroll-sortie-pixel.md`. 설계: `ver5/docs/plan-sortie-card-scrollbar.md`.

## 1. 출전 카드 (정정)

이전 작업은 열을 늘리려고 출전 창과 왼쪽 칸을 넓혔다. 이를 되돌리고, 카드 자체를 줄여 원래 왼쪽 칸에 맞췄다.

- 왼쪽 칸 폭(원래 식): PC `x + w*3/10` (창 폭 960 기준 288, 카드 영역 안쪽 266). 모바일도 같은 식(2칸 최소 규칙은 카드가 작아져 영향 없음).
- 되돌린 것: `sortieSheetSize`, `sortieSideW`, `sortieDetailW`, 확장된 `leftMax`. `sortie.go`는 `git diff cebd66d^ -- ver5/internal/gui/sortie.go`가 빈 값이다(원래 상태).
- 유지: `sortieCols`(PC 5, 모바일 3), `sortieColumns`, 스크롤·탭 변경.

| 플랫폼 | 구 카드 | 구 칸 폭 | 구 열 | 신 카드 | 신 칸 폭 | 신 열 (코드 계산) |
|---|---|---|---|---|---|---|
| PC | 104×124, 메달 56, 스왑 20 | 121 | 5 목표, 창 확장으로 달성 | 44×52, 메달 26, 스왑 14 | 52 (패딩 3) | 5 (5×52=260 ≤ 266) |
| 모바일 | 120×124, 메달 84, 스왑 40 | 139 | 3 목표, 창 확장으로 달성 | 70×84, 메달 48, 스왑 24 | 80 (패딩 3) | 3 (3×80=240 ≤ 266) |

구 열 수의 "창 확장으로 달성"은 cebd66d의 창 폭 확장이 있어야 5·3열이 나왔다는 뜻이다. 원래 상태(확장 전)는 PC·모바일 모두 2열이었다.

실측(게임 캡처, 같은 코드): PC 1280×850에서 5열, 폰 2400×1080 @130%(모바일 빌드)에서 3열. 캡처는 `captures/`.

모바일 메달 이름은 48 px일 때만 그린다. PC 메달은 26 px이라 이름을 넣지 않는다. 모바일 스왑 판정은 `touchH`(48) 정사각형이고 PC는 그림 + 6 px 여백이다(기존과 같음).

## 2. 도트 스크롤바 시안 5종

- 원본: `tools/spritetool/assets/ver5-ui/seed.py`의 `scroll_{a..e}_{track,thumb}`(10×10 아스키 격자), `pixels.json`, `settings.json`. 굽기는 `build_effects.py`의 `build('ver5-ui','ui/ui')`로 `ver5/assets/graphics/ui/ui.png`·`ui.json`에 뒤쪽으로 덧붙였다. 기존 스프라이트 좌표는 바뀌지 않았다(확인함).
- 조각 방식은 창 테두리와 같다(1 art px = 1 UI px, 최근접, 4 px 모서리 9등분, 중간 두 행 동일). 썸은 두 프레임(보통, 올라감·끎).
- 시안: A 금테 쪽매 / B 구슬 줄 / C 두루마리 축 / D 옥 장식 / E 점무늬. 서로 다른 색과 모양(금 프레임, 둥근 구슬, 나무 축과 종이 띠, 칠 바탕과 금 점, 점선 천과 체커)이다. 최종 선택 아님.
- 코드: `scrollbar_style.go`의 옛 A~E(죽간·금테 홈·그립·눈금·먹)를 제거하고 `barArt`로 교체. `table.go`는 현행 트랙 회색을 `barFlat`으로 옮겼다(중복 그리기 제거).
- 터치 판정: 기존대로 트랙 전체 높이(폭 10 UI px). 그림을 작게 해도 판정은 바뀌지 않았다. 판정 폭을 넓히면 옆 행 버튼과 겹쳐 `TestNoOverlappingButtons`를 깰 수 있어 이번에는 넓히지 않았다. 이 점은 미확인 과제다.

## 캡처 (실제 게임, `cmd/_uidemo-vm -ui scrollbar`)

- PC: `captures/desktop/frame-0025·0055·0085·0115·0145·0175.png` (현행, A, B, C, D, E 순서, 프레임마다 한 시안)
- 폰: `captures/phone/frame-0025…0175.png` (`UIDEMO_TAGS=mobile`, `-size 2400x1080 -uiscale 130`)
- 비교 페이지: `comparison.html` (트랙 확대 4배·2배, 전체 화면, 조각 16배). 모든 그림은 게임 캡처 또는 게임 UI 시트에서 잘라 확대한 것이며 목업은 없다.

## 검증

- `go build ./...`, `go build -tags mobile ./internal/gui/`, `go vet ./internal/gui/`: 통과.
- `go test -p 1 -run 'Sortie|Scroll|Overlap|Table' ./internal/gui/`: 두 태그 모두 통과.
  - `TestSortieColumns`, `TestSortieCardColumnsOnScreen`(두 태그에서 5·3열), `TestSortieCardTapAndDrag` 통과. 두 테스트는 상수 기반이라 값만 새 크기에 따른다.
  - 추가: `TestSortieCardFitsRoom`(카드 열이 원래 칸에 들어가고 모바일 스왑 판정이 `touchH` 이상), `TestScrollBarArtSprites`(다섯 시안의 조각과 썸 두 프레임).
- 게임 캡처로 열 수와 시안을 확인했다(위).

## 미확인

- 실기기 터치: 폰에서 카드·스왑·스크롤바를 실제로 누르는 확인은 하지 않았다(헤드리스 캡처와 테스트만).
- 다른 UI 배율(70·100·160% 등)과 좁은 폰(914dp 폭, 960 미만 시트)에서 열 수 확인은 테스트하지 않았다. 코드상 `sortieColumns`가 열을 줄이므로 깨지지는 않지만 실제 모습은 보지 않았다.
- 구 상태 캡처는 찍지 않았다. 구 수치는 소스(cebd66d 전후)에서 계산한 값이다.
- 1 UI px 도트는 1:1 캡처에서 흐리게 보인다. 확대 비교는 `comparison.html`에서 확인한다.
- 스크롤바 시안의 실제 손맛(드래그, 밝기 전환)은 게임에서 직접 조작하지 않았다.

## 커밋

- 코드·자산·설계: 아래 커밋 ID를 `git log`에서 확인(작업자 보고 참조).
- 결과 폴더와 `ver5/docs/history.md`의 이 작업 절: 별도 커밋. 같은 파일에 다른 세션의 미커밋 항목이 있어 이 작업 절만 인덱스에 넣었다.
