# 병종·고유 특성 중복 검사 확인

기존 `ver5/internal/core/traits_test.go`의 `TestOfficerReformBatch/personal traits do not repeat inherited class learning`은 모든 장수의 `Traits`, `Learn`과 `TraitAt`이 최종 병종의 상속 학습 목록 `AllPool`에 중복되지 않도록 검사했다.

직전 특성 선행 수정에서 이 세션이 `Traits` 및 `Learn`에 대해 다른 개인 특성의 선행이면 중복을 허용하는 예외를 넣었다. `TraitAt`의 중복 검사는 그대로다. 이 예외 때문에 병종 학습 목록에 있는 지계·부관임명 등을 개인 계획에 넣어도 테스트가 통과한다. 사용자가 승인한 규칙 변경이 아니며 명백히 잘못된 처리다.

이번 확인에서는 게임 데이터나 테스트를 추가로 수정하지 않았다. 중복 검사 원복만 하면 현재 중복 데이터가 실패하므로, 기존 고유 특성과 병종 계획의 수·순서 및 선행을 함께 고려해 재배치한 뒤 두 검사를 모두 만족시켜야 한다. git diff의 `drain` 관련 변경은 직전 작업 이전부터 존재해 이 세션의 검사 예외와 구분한다.
