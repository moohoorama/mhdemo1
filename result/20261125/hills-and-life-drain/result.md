# 산악지형보조·심공 적용 결과

- 산악지형보조는 기존 숲·산에 더해 초원 구릉과 황무지 구릉에서 지형 배율 +0.05가 된다. 구릉에 독립적인 Highland 지형 속성을 부여해 매복(Ambush)을 바꾸지 않았으므로 난전 치명타 조건은 유지된다. 같은 칸에서 두 조건을 만족해도 보정은 한 번이다. 기존 이동 비용과 지형 배율 자체는 변경하지 않았다.
- 심공의 물리 피해 회복량을 50%에서 10%로 변경하고 권사·의적 학습 풀에 추가했다. 권사·권성·의적 NPC 특성 계획에도 반영했다. 여몽도 동일 심공을 사용하므로 10%가 적용된다.
- 지형 데이터, 관련 설명과 단위 시험을 갱신했다.
- `go test ./internal/content ./internal/core -run 'TestLifeStealIsAvailableToPugilistAndOutlaw|TestLifeStealRecoversTenPercentOfDamage|TestOfficerReformBatch|TestOfficerTraits|TestTerrainFootingRaisesTheFactorEverywhere|TestClassPoolsFollowTraitDesign|TestFazhengLearnsMountainFooting|TestAmbushCrit' -count=1` 통과. 양쪽 구릉의 보조 적용, 기존 매복 판정 유지, 심공 수치·배정·회복을 확인했다. 전체 전투 밸런스 시뮬레이션은 수행하지 않았다.
