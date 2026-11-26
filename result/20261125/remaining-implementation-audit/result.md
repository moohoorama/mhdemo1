# 미구현 조사 결과

코드·데이터 변경 없음. 조사 결과는 같은 폴더 summary.md.

## 한 일
- 이전 결과(class-trait-implementation 등)의 "남은 사항"을 현행 classes.json·traits.json·characters_story.json·scenario 참조와 대조.
- 기존 summary.md가 심공 10%(05:27)와 법정 산악지형보조(05:04) 적용 전에 쓰여 구식이어서 해당 항목을 해소로 갱신.

## 미결 설계
- 병종 풀 3·5·5 나머지(권사 3개 등), 방술사·선술사 개명, 3막 이름·종료 카드, 법정·장완·비위 합류·출전 이야기(scenario에 id 참조 없음)·초상(portraits에 파일 없음).

## 구현했으나 검증 남음
- 반계 범위 책략·연쇄·연출, 보복 다단·범위·면역, 부관 조합·밸런스, 임시 수치(비용 700, NPC Lv30/40, 보복 상한 99).
- 이전 core 테스트 실패 3건(강하 일기토, 손건, c27v)과 scrollbar 데모 순수성 검사는 이번에 재실행 안 함.
- GUI 캡처(반계·막 종료 카드)는 Mac 잠금으로 실패했고 재시도 안 함.

## 검증
- 테스트·빌드는 실행하지 않음(조사 전용).

## 변경 파일
- result/20261125/remaining-implementation-audit/summary.md (갱신), result.md (신규)
