# 출전 카드 축소와 도트 스크롤바 시안

이 문서가 맡는 범위: 출전 준비 카드의 크기와 열 수(PC 5열·모바일 3열), 그리고 공통 스크롤바의 도트 시안 5종.
앞선 설계는 `docs/scroll-window.md`, 승인된 A~E 스크롤바 시안은 거부됐다.

## 출전 카드 (정정)

사용자 뜻: 열을 늘리려고 출전 부대 칸이나 출전 준비 창을 넓히지 않는다. 카드 자체를 작게 해 원래 왼쪽 칸(`leftMax = x + w*3/10`)에 맞춘다.

- 되돌린 것: `sortie.go`의 `sortieSheetSize`·`sortieSideW`·`sortieDetailW`(창 폭 확장), `leftMax`를 원래 식으로 되돌림.
- 유지: `sortieCols`(플랫폼별 5/3), `sortieColumns`(칸 수 계산), 스크롤·터치 변경.
- 카드 크기(UI px, 원래 식 기준 칸 폭 = 2·패딩 + 카드 + 메달 돌출):

| | 카드 | 메달 | 스왑 | 패딩 | 칸 폭 | 열 | 왼쪽 칸 안쪽 폭 |
|---|---|---|---|---|---|---|---|
| PC (구 cebd66d) | 104×124 | 56 | 20 | 6 | 121 | 5 목표 | (창폭 960 기준) 266 |
| PC (신) | 44×52 | 26 | 14 | 3 | 52 | 5 | 266 (5×52=260) |
| 모바일 (구 cebd66d) | 120×124 | 84 | 40 | 6 | 139 | 3 목표 | 282 (2칸 최소 304 기준) |
| 모바일 (신) | 70×84 | 48 | 24 | 3 | 80 | 3 | 266 (3×80=240) |

- 모바일 메달 이름은 48 px에서만 쓴다. PC 메달(26 px)에는 이름을 넣지 않는다(글자가 5 px가 된다).
- 모바일 스왑 표시의 터치 판정은 `touchH`(48) 정사각형이다(`sortieSwapHit`). PC는 기존처럼 그림 + 6 px 여백.
- 카드끼리·다른 패널과 겹치지 않는다. 칸 수는 `sortieColumns`가 줄여 준다(창이 좁으면 열이 줄고 카드 크기는 그대로).

## 도트 스크롤바 시안 5종

- 원본: `tools/spritetool/assets/ver5-ui/seed.py`(`scroll_{a..e}_{track,thumb}`)가 10×10 아스키 격자를 `pixels.json`에 쓰고, `settings.json`에 등록. `build_effects.py`의 `build('ver5-ui','ui/ui')`로 `assets/graphics/ui/ui.png`·`.json`을 구운 것이다(기존 조각 좌표는 그대로, 뒤에 덧붙임).
- 조각은 창 테두리와 같은 방식(1 art px = 1 UI px, 최근접). 4 px 모서리 9등분으로 높이만 늘린다. 중간 두 행은 같은 행이어야 한다.
- 썸 조각은 두 프레임(보통·마우스가 올라가거나 끌 때).
- 시안: A 금테 쪽매(금테 홈 + 금 블록), B 구슬 줄(둥근 끝 구슬), C 두루마리 축(나무 축 + 종이 띠), D 옥 장식(칠 바탕 + 금 점), E 점무늬(점선 천 + 금 체커).
- `scrollbar_style.go`: 옛 A~E(죽간·금테 홈·그립·눈금·먹)를 빼고 `barArt(prefix)`로 교체. `barFlat`(현행)은 트랙 회색을 직접 그린다(`table.go`의 중복 트랙 삭제).
- 최종 선택하지 않는다. 데모 `-ui scrollbar`가 프레임마다 한 시안을 보인다.

## 검증 계획

- `go build ./...`, `go build -tags mobile ./internal/gui/`, `go vet ./internal/gui/`.
- 테스트(직렬): `TestSortieColumns`, `TestSortieCardFitsRoom`, `TestSortieCardColumnsOnScreen`, `TestSortieCardTapAndDrag`, `TestScrollBarArtSprites` (두 태그).
- 캡처: PC 기본, 폰은 `UIDEMO_TAGS=mobile` + `-size 2400x1080 -uiscale 130`(모바일 빌드 태그는 게스트 빌드에 `UIDEMO_TAGS`로 전달).
