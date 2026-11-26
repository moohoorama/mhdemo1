# 승급 레벨 확인과 정정

현행 `ver5/assets/data/game.json`의 `PromoteLevels`는 [20,40]이며 `PointsPerLevel`은 100이다. `internal/core/formation.go`의 promotion은 현재 병종 Tier로 해당 값을 선택해 레벨을 검사한다. `internal/content/spawn.go`의 PromotedAt도 같은 값을 사용한다. 따라서 중급은 20레벨, 상급은 40레벨부터다. 플레이어 승급에는 인수 등 조건도 필요하다.

`internal/core/engine.go`의 addCharacterAt은 초기 특성치를 100×(레벨−1)로 계산한다. 이 기준으로 20레벨 1,900, 40레벨 3,900이며 실제 남은 점수는 습득에 쓴 점수에 따라 다르다.

이전 답변에서 문서의 낡은 15·30을 인용하고 1,400·2,900이라 잘못 주장했다. 이전 병종 설계 기록도 정정했다. 게임 코드는 수정하지 않았다.
