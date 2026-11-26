# 기마사격 확인·적용 결과

## 결론
이 요청의 작업은 같은 날 앞선 처리(result/20261125/mounted-archery/result.md, design.md)에서 이미 완료되어 있었다. 이번에는 코드를 새로 바꾸지 않고 현재 상태만 다시 확인했다.

## 기마사격이란
자기 차례에 3칸 이상 이동한 뒤 직접 하는 원거리 물리 공격(사격 무예 포함)의 명중률 +15%p. 반격·협동 등 자동 공격에는 적용하지 않고, 명중률 상한과 원거리 면역은 기존 규칙을 따른다. 부관에게서 배운 특성도 전달된다. 피해량은 바꾸지 않는다.

## 적용되지 않았던 이유
2026-11-24 병종 특성 적용(result/20261124/class-trait-implementation/result.md) 때 궁기병 아이디어를 확정이 아닌 후보로 남겨 "미구현"으로 기록했다. 사용자 수용을 확정으로 판단하지 않은 것이 누락 사유다.

## 현재 적용 상태
- assets/data/traits.json: mounted_shot "기마사격" (talent, 비용 700, 효과 15)
- assets/data/classes.json: 궁기병 학습 풀과 궁기병·노기병 NPC 계획(30레벨)
- internal/core/mounted_shot_test.go (신규), internal/content/class_pools_test.go
- docs/logic-traits.md 설명, docs/history.md 기록
- 비용 700과 NPC Lv30은 임시 가안.

## 검증
`go test -p 1 ./internal/core ./internal/content -run 'TestMountedShot|TestClassPoolsFollowTraitDesign' -count=1` 결과 core ok, content ok (이번에 재실행). 앞선 처리에서는 gui 시험도 통과했다. 전체 테스트와 실전 밸런스는 수행하지 않았다.
