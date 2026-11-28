# 지역별 성 안 장면 (2026-11-28)

20261127/huabei-town-buildings의 지역 건물(A안 2차)을 실제 장면 맵에 놓고 maplight(시나리오 어드민과 같은 렌더러)로 그린 비교.
게임 데이터(ver5/assets)는 바꾸지 않았다.

| 새 맵 | 원본 | 지역 |
|---|---|---|
| S_zhuo | S_zhuo (탁현) | 화북 (땅을 누렇게) |
| S_luoyang | S_luoyang (낙양) | 중원 |
| S_xuzhou_town | S_xuzhou_town (서주) | 회남 |
| S_xiangyang | S_xiangyang (양양) | 회남 |
| S_demo_jiangnan | S_xiangyang 지형 | 강남 |
| S_demo_bashu | S_zhuo 지형 | 파촉 |

- `current/`, `new/`: 맵 전체 렌더
- `tools/build_root.py`: ver5/assets를 복사한 미리보기 루트를 만들고 지역 건물을 extra 시트·scene.json footprints에 넣는다
- `tools/remap.py`: 옛 집·관아 → 지역 건물, 가장 가까운 빈자리에 배치, 빈 땅 채우기(FILL)
- `index.html`: 비교 페이지

재현:

```
SP=<작업 폴더>; python3 town_dot2.py reg; python3 build_root.py reg root; python3 remap.py root
cd root && <ver5에서 빌드한 _maplight> S_zhuo out.png
cd root && <ver5에서 빌드한 scenario-admin> -root . -addr 127.0.0.1:18766
```

주의: 미리보기 루트의 assets/data 등은 심볼릭 링크로 두면 scenario-admin이 "rules incomplete"로 꺼진다(폴더를 훑을 때 링크를 안 따라감). 복사해야 한다.

남은 점: 배치는 자동이라 길 따라 줄 맞추기·마당 연출은 손으로 다듬어야 함. 지역별 성벽·수목 없음. 실제 반영 시 assets_candidate로 옮기고 build_extra, 장면 걷기 칸 확인.

## 2차 (11-28): 건물은 집만, 담장·울타리는 줄 타일

- `tools/town_dot2.py`: 건물 함수의 마당 요소(`yard_from` 뒤)를 빼고 집 몸체에 맞춰 바닥을 다시 잡음(`house_only`). 지붕 경사 낮춤, `tools/shade.py`로 처마 밑 그늘·해 그림자.
- `tools/town_lines.py`: 지역별 줄 타일 `<지역>_twall`(기와 담장), `_ewall`(흙담), `_fence`(울타리) 각 16조각(마스크 1 -u, 2 -v, 4 +u, 8 +v), 대문 `_tgate_u/v`, `_egate_u/v`(2칸 물건). scene.json `walls`에 `{"piece": 이름}`으로 등록.
- `tools/remap.py`: 관청·저택은 기와 담장+대문, 초가는 흙담+사립문 / 울타리 / 없음을 번갈아, 집 앞 2칸·옆뒤 1칸 마당을 둘러 `walls` 줄로 기록.
- `v2_embedded/`: 담장이 건물 그림에 박혀 있던 직전판, `parts/`: 지역별 부품 시트.

주의(11-28): 엔진의 줄 타일은 이웃 칸에 **종류가 달라도** 줄이 있으면 이어졌다고 본다(plan.go `closed`). 저택 담장을 성벽 바로 옆 칸에 두면 성벽의 성가퀴·그늘 조각이 빠진다. remap은 담장 테두리를 기존 줄과 한 칸 띄우고, 자리가 없으면 담장 없이 집만 놓는다.
