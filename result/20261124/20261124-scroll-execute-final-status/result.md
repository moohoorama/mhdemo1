# 통합 스크롤 작업 완료 상태 확인 결과

## 결론

구현은 끝났다. 다만 완료라고 보기에는 확인되지 않은 항목이 남아 있다.

## 근거

- 커밋(main, push 안 함): `cebd66d`(스크롤·3/5열), `20c548c`(설계·시안 비교·결과·history), `5ce99e0`(커밋 ID 기록). 그 앞 `a880fcf`는 gui 체크포인트다.
- 결과 폴더 `result/20261124/20261124-scroll-execute/`: `result.md`, `scrollbar-options.html`, `capture/`. 시안 5종과 현행 참고 캡처가 있다.
- 테스트: 대상 테스트(스크롤·출전·목록) 데스크톱·모바일 태그 모두 통과. 전체 `internal/gui`는 2건 실패.
  - `TestScenarioNodesAreStaged`: 커밋 `702d2c5`의 장면 배치 누락(`c23-xinye-arrival`, `c25-sick-news`). 이번 작업과 무관.
  - `TestStoryManagementDeputy`: 다른 세션의 미커밋 `story13.json` 변경. 이번 작업과 무관.
- 워크트리: 스크롤 관련 소스 변경은 모두 커밋됐다. 남은 `uidemo_zz_tmp_*.go` 2개는 다른 세션 임시 파일이라 건드리지 않았다.

## 아직 확인하지 않은 것

- 실제 휴대폰 터치 감각과 관성 느낌.
- 경유지 화면 드래그 자동 테스트 없음, 좁은 창 출전 캡처 없음.
- `cebd66d` 단독 체크아웃은 컴파일되지 않는다. 다른 세션 미커밋 변경이 필요하다.
- 스크롤바 시안 5종은 아직 고르지 않았다.
