# ver5 UI 그림

UI가 쓰는 그림이다. 형식은 `ver5-fx`와 같다(레이어, 팔레트, 투명도, `gen.py` 부드러운 레이어). 배율이 그림마다 다르다.

- 그림 4px = UI 1px, 게임이 부드럽게 축소한다. 흰색으로 그려 게임이 물들인다. 이웃 그림이 축소에 섞여 들지 않게 둘레에 8px 투명 여백을 둔다.
  - `sort_mark`: 목록 정렬·HUD 접기의 작은 삼각형(위, 아래). `scroll_mark`: 스크롤 버튼의 삼각형(위, 아래).
  - `icon`: 명령 메뉴 아이콘(공격 검, 병법 책, 도구 주머니, 대기 모래시계). 12 UI px 칸, 앵커가 그 칸의 위 왼쪽.
- 그림 1px = UI 1px, 최근접.
  - `window_rim`: 모든 창의 검은 가장자리·금색 이중 테두리·모서리 장식을 17×17 창에 그린 것. 게임이 모서리 8px로 9조각 내어 늘린다(`skin.go`의 `nineSlice`). 바탕은 단색 채우기다.
  - `window_glow`: 창 위쪽 빛 띠 8단, 한 줄이 한 단(게임이 바탕 폭과 높이의 1/16로 늘림).
- 두루마리 배율(그림 1px = UI 4px): `scroll_curl`(종이가 굴대 쪽으로 말리며 어두워지는 6단, 왼쪽용, 오른쪽은 뒤집음), `scroll_grain`(굴대 나뭇결 한 점).
- 게임 화소(맵 위): `bar_frame`(HP바 칸), `bar_fill`(적, 아군), `bar_trail`(흰 잔상), `bar_ticks`(4등분 눈금), `bar_mp`(MP 줄). 게임이 남은 양만큼 가로로 잘라 얹는다.

`seed.py <스프라이트>...`가 첫 판을 쓰고 손질은 `pixels.json`에서 한다. ver5 디렉터리에서 `uv run --with pillow python tools/build_effects.py`로
`assets/graphics/ui/ui.png`와 `.json`을 굽는다.
