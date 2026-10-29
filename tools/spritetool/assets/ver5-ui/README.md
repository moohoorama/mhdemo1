# ver5 UI 그림

UI가 쓰는 그림이다. 형식은 `ver5-fx`와 같다(레이어, 팔레트, 투명도, `gen.py` 부드러운 레이어). 배율이 그림마다 다르다.

- 그림 4px = UI 1px, 게임이 부드럽게 축소한다. 흰색으로 그려 게임이 물들인다. 이웃 그림이 축소에 섞여 들지 않게 둘레에 8px 투명 여백을 둔다.
  - `sort_mark`: 목록 정렬·HUD 접기의 작은 삼각형(위, 아래). `scroll_mark`: 스크롤 버튼의 삼각형(위, 아래).
  - `edge_arrow`: 전투 맵 가장자리 스크롤 커서(북, 북동, 동, 남동, 남, 남서, 서, 북서). 24×24 도트(`"dot": 4`)를 아스키로 손으로 찍는다 — 동·북동만 그리고 나머지는 돌리거나 뒤집은 뒤 명암(위가 외곽선이면 `l`, 아래가 외곽선이면 `d`)을 다시 넣는다. 외곽선 1도트, 비탈은 1:1 계단. 게임은 도트당 UI 2px, 최근접으로 그린다.
  - `icon`: 명령 메뉴 아이콘(공격 검, 병법 책, 도구 주머니, 대기 모래시계). 12 UI px 칸, 앵커가 그 칸의 위 왼쪽.
  - `trait_strategy`·`trait_tactic`·`trait_talent`: 특성 분류 아이콘(책략 두루마리, 전법 교차한 창·검, 특기 메달). 26×26 도트(`"dot": 4`, 22 UI px + 2도트 여백)를 회색 단계와 어두운 외곽선으로 그리고, 게임이 특성의 `Tone` 색으로 물들인다.
  - `item_<이름>`: 장비 아이콘(`equipment.json`의 `Icon`). 색을 칠한 36×36 도트를 `"dot": 4`로 그림 4px 덩어리로 키워 굽는다(32 UI px 그림 + 둘레 2도트 여백, 앵커가 그림의 위 왼쪽).
    장비 도트는 그림에서 변환한다(손으로 찍지 않는다). 지금 아이콘이나 설명으로 `codex-image` 스킬에 마젠타(#FF00FF) 바탕 그림을 그리게 하는데,
    프롬프트에 "32x32 pixel art RPG inventory icon, big crisp square pixels, few large shapes, 3-tone cel shading, no texture, dark outline, rounded silhouette"를 넣어야 줄였을 때 얼룩이 안 진다.
    긴 무기는 손잡이 왼쪽 아래 → 끝 오른쪽 위 45°, 글자·한자 없음. 그림을 512px로 줄여 `sources/<스프라이트>.png`에 두고 설정에 `"source"`를 적은 뒤
    ver5에서 `uv run --with pillow python tools/pixelize_icon.py <스프라이트>...`. 도구는 마젠타를 빼고 30도트(`"fit"`)에 맞춘 뒤, 큰 그림에서 픽셀마다 팔레트 색(Lab 거리, `"use"`로 쓸 글자 제한)이나
    외곽선 `k`(그림의 짙은 선)로 바꾸고 칸마다 가장 많은 색을 고른다. 칸의 4분의 1을 덮는 색은 외곽선을 이기고, 짙은 선만 있는 듬성한 칸은 비워 둬서 가는 시위·칼날이 산다.
    칸이 `"solid"`(기본 .5) 이상 덮여야 칠한다 — 칼·창은 .35, 활은 .2. 끝으로 외톨이 점을 지우고 1px `k` 외곽선을 두른다. 팔레트는 설정의 `layers.icon.palette`다.
- 그림 1px = UI 1px, 최근접.
  - `window_rim`: 모든 창의 검은 가장자리·금색 이중 테두리·모서리 장식을 17×17 창에 그린 것. 게임이 모서리 8px로 9조각 내어 늘린다(`skin.go`의 `nineSlice`). 바탕은 단색 채우기다.
  - `window_glow`: 창 위쪽 빛 띠 8단, 한 줄이 한 단(게임이 바탕 폭과 높이의 1/16로 늘림).
- 두루마리 배율(그림 1px = UI 4px): `scroll_curl`(종이가 굴대 쪽으로 말리며 어두워지는 6단, 왼쪽용, 오른쪽은 뒤집음), `scroll_grain`(굴대 나뭇결 한 점).
- 게임 화소(맵 위): `bar_frame`(HP바 칸), `bar_fill`(적, 아군), `bar_trail`(흰 잔상), `bar_ticks`(4등분 눈금), `bar_mp`(MP 줄). 게임이 남은 양만큼 가로로 잘라 얹는다.

`seed.py <스프라이트>...`가 첫 판을 쓰고 손질은 `pixels.json`에서 한다. ver5 디렉터리에서 `uv run --with pillow python tools/build_effects.py`로
`assets/graphics/ui/ui.png`와 `.json`을 굽는다.
