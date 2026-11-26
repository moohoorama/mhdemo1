# 구릉 지형보조·심공 적용 결과

이번 실행에서는 코드를 새로 바꾸지 않았다. 같은 요청의 이전 처리분이 이미 작업 트리에 반영되어 있어 내용과 시험만 다시 확인했다. 상세 설계와 결과는 `result/20261125/hills-and-life-drain/`의 design.md, result.md에 있다.

## 반영 내용 (작업 트리 기준)
- 구릉(초원·황무지)에 `Highland` 지형 속성을 두어 산악지형보조가 지형 배율 +0.05를 받게 했다. 매복(Ambush) 속성은 건드리지 않았고, 난전 치명타 판정(`rules.go`)은 매복 또는 구릉이면 적용되는 기존 구조를 유지한다.
- 심공(`drain`)의 `life_steal`을 0.5에서 0.1로 바꾸고 설명을 10%로 고쳤다. 권사·의적 학습 풀에 추가했고 NPC 계획도 반영했다.

## 관련 파일
- ver5/internal/content/terrain.go
- ver5/assets/data/traits.json
- ver5/assets/data/classes.json
- ver5/docs/logic-traits.md
- ver5/internal/content/class_pools_test.go
- ver5/internal/content/officer_traits_data_test.go
- ver5/internal/core/officer_traits_test.go
- ver5/internal/core/traits_test.go

## 검증
- `go test -p 1 ./internal/content ./internal/core -run 'TestLifeStealIsAvailableToPugilistAndOutlaw|TestLifeStealRecoversTenPercentOfDamage|TestOfficerReformBatch|TestOfficerTraits|TestTerrainFootingRaisesTheFactorEverywhere|TestClassPoolsFollowTraitDesign|TestFazhengLearnsMountainFooting|TestAmbushCrit' -count=1` 결과: content, core 모두 ok.
- 전체 전투 밸런스 시뮬레이션과 전체 테스트는 실행하지 않았다.
