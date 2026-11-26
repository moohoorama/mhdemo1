# 공통 스크롤·출전 배치 진행 상태 확인 결과

## 결론

이 대화에서 `20261123-scroll-sortie-layout` 작업은 시작하지 않았다. 작업 트리에는 같은 영역을 건드린 미커밋 변경이 있으므로, 다른 세션이 진행 중일 가능성이 높다. 완료로 볼 근거는 없다.

## 근거

- 참조 보고서 `result/20261123/scroll-sortie-status/summary.md`: 요청은 process로 옮겨졌지만 결과 보고서는 접수·미착수 상태다. 범위 순서를 묻고 끝났다.
- `result/20261123/20261123-scroll-sortie-layout/result.md`: 접수·미착수 상태이며 범위 질문이 남아 있다.
- `ver5` 작업 트리(git status):
  - 수정: `internal/gui/sortie.go`, `internal/gui/sortie_deputy_test.go`, `internal/gui/sortie_unified.go`
  - 신규(미추적): `internal/gui/platform_desktop.go`, `internal/gui/platform_mobile.go`
- 실행 중인 작업: `_uidemo-vm` 캡처(`jiejiao-ford-ambush`)와 `go test ./internal/core ./internal/content`. 다른 세션이 돌리고 있다.

## 하지 않은 것

- 해당 파일들을 건드리지 않았다. 다른 세션의 미커밋 변경을 보존하기 위해서다.
- 테스트와 캡처를 돌리지 않았다. 동시에 돌리면 캡처가 서로 겹칠 수 있다.

## 확인이 필요한 점

- 이 미커밋 변경이 같은 요청을 처리 중인 세션의 것인지 확인해야 한다.
- 범위 순서(출전 배치 먼저인지, 공통 스크롤 먼저인지) 답이 있어야 이 요청을 진행할 수 있다.
