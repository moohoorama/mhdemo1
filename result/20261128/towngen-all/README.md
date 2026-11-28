# 성·마을 생성안 (2026-11-28)

탁현(towngen-zhuo)에서 승낙받은 규칙으로 성·마을 장면 맵 22곳을 다시 짠 안. 미리보기 루트에만 적용, 게임 데이터 그대로.
- current/, generated/: 맵 렌더, roles/: 맵별 역할·구역·점검·바뀐 출구
- tools/: towngen.py(생성기), batch.sh(전체), build_root.py, town_dot2.py·town_lines.py·shade.py(그림), page_all.py
- 점검: 22곳 모두 검사 통과(지킬 칸 덮임 0, 역할 건물 누락 0), 걸어서 못 닿는 곳 0
