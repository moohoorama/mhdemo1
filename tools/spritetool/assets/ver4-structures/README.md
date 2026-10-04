# ver4 맵 구조물

ver4 전투 맵의 마을(`v`)·성벽(`c`)·성내(`i`) 칸 그림. 생성: `python3 tools/unit3d/structures.py`
(검토 장면: `tools/unit3d/`에서 `python3 -c "import structures; structures.scene()"` → `output/ver4-structure-scene.png`).
느낌은 사용자가 준 mhero-structures 참고 그림(초가집·돌바닥·성벽 절벽면)을 따르되 현재 32×16 마름모와 병종 도트에 맞게 다시 그렸다.

| 스프라이트 | 내용 | 그리는 층 |
|---|---|---|
| `village` | 13×12 데포르메 초가집 4채를 마름모로 묶음. 방향 고정(긴 벽 SW, 박공·문 SE) | 장식(유닛과 발밑 y 정렬, 같은 칸 유닛보다 먼저) |
| `floor` | 성내 밝은 돌바닥, 바닥 높이, 칸당 2줄 엇갈린 돌 | 지면 |
| `wall_00`–`wall_15` | 성벽: 10px(`rise`) 높인 어두운 돌바닥. 성벽이 아닌 이웃 쪽 변(마스크)에 성가퀴, 앞쪽(SE·SW)에는 돌 쌓은 벽면 | 장식 |

- 시트 `structures.png`: 셀 40×48 가로 나열, 순서는 `structures.json`의 `sprites`. 기준점 (21,39) = 칸 마름모 중심.
- 성벽 마스크 비트 i는 `wall_mask_sides[i]`(nw, ne, se, sw) 쪽 이웃이 성벽이 아님을 뜻한다. 맵 좌표로 nw=(u−1,v), ne=(u,v−1), se=(u+1,v), sw=(u,v+1).
- 벽돌과 성가퀴는 칸 중심에서 8px 주기로 반복하므로 이웃 칸과 이어진다. 성벽 칸 위 유닛은 `rise`만큼 올려 그린다.
- 팔레트는 병종과 같은 기사 v6 키(`tools/unit3d/render.py`). 알파 0/255.
