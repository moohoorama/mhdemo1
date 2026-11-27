# 타이틀 영상 맵: S_guandu_field, S_chibi_wreck

- `ver5/tools/scene_maps/guandu_field.py` → `S_guandu_field` (40×64, origin [-60,-60] 작은 칸, 판 u −30~69, v −30~93)
- `ver5/tools/scene_maps/chibi_wreck.py` → `S_chibi_wreck` (24×17, origin [-60,-60], 판 u −30~53, v −30~46)
- index.json maps에 두 줄 추가, docs/presentation-scene.md 표·docs/history.md 기록
- 칸 표: guandu_cells.txt, wreck_cells.txt (게임 칸 (u,v)의 작은 칸 (2u,2v): `.` 설 수 있음, `#` 막힘, `o` 물체 기준점)
- 캡처: *_fit.png(맵 전체), *_z2.png(배율 2)
- 시험: scenemap TestEncode의 S_title_* 실패는 다른 세션의 미추적 파일(내 맵은 통과), gui TestSceneMapsLoadAndDraw 통과
