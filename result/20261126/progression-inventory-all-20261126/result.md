# 결과: 시나리오 전 범위 신규 요소 조사 (재확인)

## 한 일
- 요청: 1~11장에 그치지 않고 현재 등록된 모든 장·전투의 새 장수·보물·특성·조건부 보상을 정리하고, 연의·가상 분기를 구분하며 전체 스크립트 출력을 보존한다.
- 기존 결과 `result/20261126/progression-inventory-all/`(summary.md, all-chapters.md, 06:33 작성)을 재확인했다. 새로 쓰거나 고치지 않았다.

## 검증
- `cd ver5 && python3 tools/scenario_doc/progression.py`(인자 없이 전 장) 출력을 `result/20261126/progression-inventory-all/all-chapters.md`와 비교: 동일(SAME), 종료 코드 0. 기록은 `test/progression-inventory-all-20261126/test-log.md`의 `rerun-all-compare`.
- summary.md의 몇 항목을 전체 출력에서 찾아 대조했다: 16장 방천화극(요극), 24장 전 이후 유성추, 25v 청홍검(청공). 모두 출력과 일치.
- 한계: summary.md 전체(30장 표)를 항목별로 대조하지는 않았다. 스크립트가 조건·분기를 판정하지 않으므로 표의 "◆ 조건부" 표시가 원본과 맞는지는 이번 확인 범위 밖이다.

## 변경 파일
- 없음. 이 폴더의 result.md만 추가했다.
