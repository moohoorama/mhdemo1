# 결과: 연노병 특성 조회

## 한 일
- 요청: 현재 연노병의 특성과 습득 비용, NPC 특성 계획 확인.
- 기존 정리 `result/20261126/repeating-crossbow-traits/summary.md`의 내용을 `ver5/assets/data/classes.json`(키 `repeating_crossbow`)과 `ver5/assets/data/traits.json`에 대조했다.
- 코드·데이터·기존 summary.md는 수정하지 않았다.

## 확인 결과
- 습득 후보 Pool: `exhort` 독려 700, `penetrate_art` 관통 1000, `rapid_art` 연발 1000, `far_sight` 원시 1000, `attack_increase` 공격력 증가 1000. classes.json Pool 목록과 일치.
- NPC용 Plan(classes.json `Plan`): 10 `accuracy_up` 공격명중률증가(500), 20 `fire_arrow_art` 화시(500), 30 `relief` 구휼(500), 40 `rapid_art` 연발, 49 `snipe_prep` 저격(700), 50 `penetrate_art` 관통, 60 `far_sight` 원시. 레벨과 특성 이름 모두 summary와 일치.
- 요약의 "Plan은 적·AI 동맹에 적용되며 플레이어의 자동 습득이 아니다"는 데이터만으로는 확인할 수 없다(적용 로직 미확인). 이 항목은 검증하지 않았다.
- "밀집사격·장거리 사격은 논의 중인 제안"은 두 특성 id가 현재 데이터에 없는 것으로 확인했다(제안으로 표기된 그대로).

## 변경 파일
- 없음(결과 폴더의 result.md만 추가).

## 검증
- 데이터 대조 스크립트를 `RF run-test`로 실행한 결과를 아래에 기록한다.
