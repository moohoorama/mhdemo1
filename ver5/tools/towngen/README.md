# towngen — 성·마을 장면맵 생성

규칙으로 성·마을 장면 맵을 다시 짜고 지역 건물로 채운다. 절차와 원칙은 스킬 `.claude/skills/towngen/SKILL.md`.

```
tools/towngen/batch.sh --art                 # 지역 건물·줄 타일 그림부터 (work/reg, work/lines) + 전체 맵
tools/towngen/batch.sh S_zhuo:huabei         # 한 맵만 (그림은 있는 것 사용)
python3 tools/towngen/page_all.py            # 비교 페이지 (result/…/towngen-all)
tools/towngen/scene_compare.sh               # 스토리 장면을 옛/새 맵에 그려 비교 (work/cap/pair)
(cd tools/towngen/work/root && ../bin/scenario-admin -root . -addr 127.0.0.1:18766)   # 미리보기 어드민
```

| 파일 | 하는 일 |
|---|---|
| town_dot2.py | 지역 건물(초가집 4·대장간·주점·도구점·관청·저택 4·밭/논), 반전판 `_r`, 그림자 마스크 |
| town_lines.py | 담장·흙담·울타리 줄 타일 16조각, 대문 |
| town_dot.py, shade.py | 공용 페인트, 처마 밑 그늘·해 그림자·땅 그림자 마스크 렌더 |
| build_root.py | 미리보기 루트(ver5/assets 복사) + 그림을 extra 시트·scene.json에 |
| towngen.py | 한 맵 생성 + 탐험 파일 출구 고치기 + roles 기록 |
| stage.py | 장면 무대: 선 자리·걷는 길, 사람을 가리는 물건, 장면이 기대던 옛 집, 광장 |
| apply.py | 게임에 반영: 그림 → assets_candidate·build_extra, scene.json, 맵(_mapfmt), 탐험 병합, 2장 대장간 장면 |
| scene_compare.sh | 장면을 옛/새 맵에 그려 비교 (코드 사본 둘, scenespots) |
| batch.sh, page_all.py, check_cells.py | 전체 실행, 페이지, 점검 |

생성물은 `work/`(git 밖, `TOWNGEN_WORK`로 바꿀 수 있음). 생성의 출발점은 towngen 이전 커밋 f96ad23의 맵·탐험·장면(`work/base`, `TOWNGEN_BASE`). 2026-11-28 게임에 반영(`apply.py`).
