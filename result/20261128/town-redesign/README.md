# 성·마을 재설계안 (2026-11-28)

시나리오 탐험의 성·마을 장면 맵 19곳을 지역 건물(20261127·20261128 작업)로 다시 깐 안. 게임 데이터는 그대로이고,
미리보기 루트(작업 폴더 root/, ver5/assets 복사본)에만 적용했다. 미리보기 어드민: `cd root && scenario-admin -root . -addr 127.0.0.1:18766`.

- `current/`, `redesign/`: 맵 전체 렌더 (maplight)
- `redesign.json`: 맵별 바뀐 것(건물 수, 담장 줄, 물러선 건물, 옮긴 출구, 대장간)
- `tools/redesign.py`: 재설계 생성 (사람·출구 칸 비우기, 출구 건물 문 앞으로, 대장간 바깥으로)
- `tools/check_cells.py`: 사람·출구 칸이 덮였는지, `tools/walkcheck_main.go`(= ver5/cmd/_walkcheck): 걸어서 닿는지

점검: 덮인 칸 0/625, 못 닿는 곳 0(현재판도 0), 어드민 경고 4→4(기존 초상 경고), 다른 줄 맞닿음 0.

엔진 변경(저장소, 커밋 전): 그림자 마스크가 붙은 줄 조각도 땅 그림자를 그림 — internal/gui/assets.go·field_scene.go,
internal/maplight/render.go·shadow.go. 기존 장면 맵 101개 전후 렌더 픽셀 동일.

남은 점: 실제 반영(그림을 assets_candidate로 옮겨 build_extra, scene.json, 맵·탐험 파일), 마당 소품, 밭·논 땅 글자,
배치를 손으로 다듬기(자동 배치라 길 따라 줄 맞추기는 안 함).
