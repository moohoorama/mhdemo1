# 결과: 연노병 특성 변경 정리

## 한 일
- 요청: 연노병 특성 풀에서 독려를 빼고, 적과의 거리가 길수록 원거리 물리 피해가 증가하는 특성을 넣는 결정을 정리. 명칭·비용·증가율·상한은 미정.
- 기존 `result/20261126/repeating-crossbow-long-range/summary.md`를 확인했다. 결정 기록으로서 내용이 요청과 맞다.

## 검증
- 현재 데이터 `ver5/assets/data/classes.json`(repeating_crossbow Pool)에는 아직 `exhort`(독려, 비용 700)가 있고, 거리 비례 특성은 없다. 요약의 "현재 게임 데이터는 수정하지 않았다"와 일치. 기록: `test/20261126-repeating-crossbow-long-range/test-log.md`(`crossbow-pool-current`, PASS).
- 이번 요청은 결정을 기록하는 범위이므로 데이터는 바꾸지 않았다. 신규 특성의 이름·비용·증가율이 미정이라 실제 데이터 반영은 그 값이 정해진 뒤에 가능하다.

## 변경 파일
- 없음(이 폴더의 result.md만 추가).
