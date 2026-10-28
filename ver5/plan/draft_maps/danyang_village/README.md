# 단양 고순의 마을 시안 (2026-10-28)

4장 첫 장면(`c04-start`)의 마을. 세 안을 `gen.py`가 이 폴더에 쓴다(맵 ID `S_draft_dy_*`). 비교 페이지: https://claude.ai/artifact/4sa3t5CVtM2NnsdgFk5C7N

| 안 | 파일 | |
|---|---|---|
| A 어귀 앞 | `dy_gate.json` | 고르지 않음 |
| B 마을 마당 | `dy_square.json` | 고르지 않음 |
| C 산자락 마을 전체 | `dy_whole.json` | **고름** → `assets/graphics/maps/S_danyang_village.json` (id·이름만 바꿔 복사) |

C안을 고치면 `python3 gen.py` 뒤 `dy_whole.json`을 id `S_danyang_village`, 이름 `단양 산자락 마을`로 바꿔 다시 복사한다.
`../pengcheng_field/gen.py`가 이 `gen.py`의 도구(`Map`)를 가져다 쓴다.
