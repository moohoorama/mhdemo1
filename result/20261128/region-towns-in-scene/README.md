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
