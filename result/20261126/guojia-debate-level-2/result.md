# 곽가 논변 계획

논변의 선행은 혼란으로 두고 곽가의 논변을 개인 Traits에서 TraitAt 40으로 이동했다. 외교관의 계획에 혼란이 40레벨에 있으므로 40레벨부터 둘 다 얻으며, 이전 레벨에는 논변이 없다. NPC에게 선행 특성을 자동으로 추가하는 코드와 그 전용 테스트는 철회했다.

장수 8명의 병종 특성 중복은 이전 요청에 따라 개인 계획에서 제거한 상태다. 허사·순유·하후연은 앞서 지정된 고유 특성도 반영했다. 이들 중 허사·순유와 유비·황보숭·번궁·유벽·법정은 자동 보완을 없애자 개인 계획의 선행 시점이 잘못되었음이 드러났다. 병종 공통 계획 또는 개인 특성 시점에 추가 조정이 필요하다. 전 장수 계획 선행 검사는 이 7명 때문에 실패하므로 완료 처리하지 않는다.

곽가 1·39·40·41레벨 경계, 고유·병종 중복, 병종 계획 선행, 학습 선행 및 기존 특성 효과 검사는 통과했다. 상세: `test/guojia-debate-level-2/test-log.md`.

변경: `ver5/assets/data/traits.json`, `ver5/assets/data/characters.json`, `ver5/assets/data/characters_story.json`, `ver5/internal/content/trait_prerequisite_audit_test.go`, `ver5/internal/content/guojia_plan_test.go`, `ver5/internal/core/traits_test.go`, `ver5/docs/history.md`. 자동 보완을 추가했던 `internal/content/spawn.go`, `internal/core/engine.go`의 해당 코드와 전용 테스트는 철회.
