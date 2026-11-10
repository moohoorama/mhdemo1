# 진영 맵 시안 (2026-10-28)

탐험 지역의 진영 셋을 넓게 다시 짓는 시안. 진영마다 A·B 두 안을 그렸고, 사용자가 고른 셋을 게임에 넣었다.
비교 페이지: https://claude.ai/artifact/6smDzyhTK56B7bbdGvDpwH

| 진영 | 고른 안 → 게임 맵 | 고르지 않은 안 (여기 보관) |
|---|---|---|
| 1장 의병 주둔지 | A 성벽 아래 주둔지 → `S_militia_camp` (성벽에 붙은 나무는 뺌) | B 마을이 내준 초가 `militia_b.json` |
| 2장 노식의 진영 | B 전영·중군·후영 일렬 → `S_luzhi_camp` | A 사각 진영 `luzhi_a.json` |
| 5·6·7장 연합 진영 | A 십자 대로 → `S_coalition_camp` (진영 사이 길을 세 칸으로 넓히고 문을 큰길 쪽으로) | B 강 따라 일렬 `coalition_b.json` |

- `gen_camps.py`가 여섯 안을 모두 이 폴더에 다시 쓴다(`*_a.json`·`*_b.json`, 맵 ID는 `S_draft_<이름>`). 게임에 넣은 셋은
  그 뒤 손으로 고쳤으므로(큰 막사 `command_tent`, 군량 수레 `grain_cart`) `assets/graphics/maps/`의 파일이 원본이다.
- 여기 맵은 `assets/` 밖이라 게임·시험이 읽지 않는다. 보려면 `index.json`의 `maps`에 등록한 사본에서 `go run ./cmd/_scene`으로 연다.
- 고르지 않은 안을 쓰게 되면: 파일을 `assets/graphics/maps/S_<이름>.json`으로 옮겨 `id`를 바꾸고 `index.json`에 등록한 뒤,
  `go run ./cmd/_walkmap -fmt S_<이름>`으로 맵 파일 모양을 맞추고 탐험 파일의 사람·출구 자리를 `_walkmap`으로 다시 잡는다.
- 좌표는 작은 칸, 원점 (0, 0). 위쪽(작은 u+v) 끝이 화면 위.

## 광릉 진영 (2026-11-09)

15장 S#1 `c15-grain`. `gen_guangling.py`가 두 안을 쓴다. A 강가의 패잔 진영 → `S_guangling_camp`(이름·ID만 바꿔 복사, 들머리 길목을 넓히고 깃발을 목책 끝으로), B 불탄 마을 터 `guangling_b.json`은 보관. 비교 페이지: https://claude.ai/artifact/4rJgi9DxFJGe5T7fQpX2NW
