# 지역별 성 안 건물 시안 (2026-11-27)

castle_town_concept/huabei_castle.png와 지역 표(화북·중원·회남·강남·파촉)를 보고 장면맵 성 안 건물을 만든 시안. 게임에는 아직 넣지 않음.

- `regions/<지역>/`: 건물 PNG(원본 크기)와 meta.json(크기·pivot=발자리 중심·footprint 소타일)
  - cottage_1~4 (4×4), smithy·tavern·shop (6×4), yamen (8×3), mansion_1·2 (6×4), mansion_3 (8×3), mansion_4 (4×4), field_dry/field_paddy (4×4)
- `regions/<지역>_town.png`: 성 안 배치 예
- `v1/`: 1차 시안(좁은 바닥, A 도트 직접 / B 3D 모형)
- `tools/town_dot2.py`: A안 2차 — 지역 설정(REGIONS: 팔레트, 지붕 경사·처마·형태, 벽 재료, 고상, 밭/논)
- `index.html`: 비교 페이지

재현 (assets_candidate/tools의 gen_pixel_castle 렌더러 사용):

```
cd tools && python3 town_dot2.py ../regions && python3 diorama2.py ../regions
```

남은 점: 고른 세트를 assets_candidate/tools로 옮겨 tiles.json·scene.json footprints·build_extra 연결, 게임 캡처 확인.
굴뚝 연기·화덕 불빛·홍등은 scene.json attachments로. 지역 표의 성벽·수목·생활 소품은 이번 범위 밖.
