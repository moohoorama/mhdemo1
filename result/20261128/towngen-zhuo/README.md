# 탁현 생성안 (2026-11-28)

원칙(사용자 11-28)대로 장면 맵을 다시 짜는 생성기 towngen의 첫 맵. 미리보기 루트에만 적용, 게임 데이터 그대로.
- generated.png(생성판), current.png(현재), swap_version.png(앞 치환판), zones.png(구역 배치도)
- roles_S_zhuo.json: 방문 건물·역할, 세운 건물·문 칸, 구역 블록, 점검
- tools/towngen.py: `python3 towngen.py ROOT S_zhuo huabei` (reg/·lines/ 필요: town_dot2.py, town_lines.py)
- 거울판(_r): town_dot2.make_mirrored — 거울 빛으로 그리고 좌우로 뒤집음
