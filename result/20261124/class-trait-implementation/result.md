# 병종·장수 특성 적용 결과

## 적용
- `assets/data/classes.json`: 문관 병종 풀·NPC 계획, 궁기병·노기병 사거리·능력치와 저지 사격, 상급 능력치 증가, 방진·수성 및 권성 반격강화와 권사·흉적 응급처치.
- `assets/data/traits.json`, `internal/content/vocab.go`, `internal/core/rules.go`, `engine.go`: 능력치 증가(부관 중복 합산), 방진·수성·지공·저지 강화. 비용과 궁기병 능력치 수치는 구현 가안이다. 수성은 장비 반영 이후·상태 강화 이전에 평가한다.
- `internal/core/combat.go`, `engine.go`, `internal/gui/events.go`: 실제 피해에서 쌓이는 보복, 중첩 표시 및 자기 차례 공격이나 턴 종료에 소멸. 동일 상태 최대 99는 기존 저장 검증 상한에 따른 임시 제한이다.
- `internal/core/combat.go`, `internal/gui/player.go`: 반계는 책략 처리·연출 경로를 사용하고 재반계는 막는다. 기존의 원 책략 빗나감 시 발동 규칙은 유지했다. 범위 책략은 반계 시전자 기준 원 시전자 칸을 중심으로 시전하는 가안이다.
- `assets/data/characters_story.json`: 허사·황보숭·번궁 화공, 정욱 격파, 하후연 저지 강화, 법정·장완·비위 인물 데이터. 법정 네 번째 특성과 실제 합류·출전 이벤트는 미정. 하후연 상위 해금은 Lv40 가정.
- `assets/scenario/hero/story16.json`, `story27.json`, `story27v.json`, `docs/fullstory.md`: 큰 이야기 종료 카드의 1막·2막 표기.
- `internal/gui/uidemo_counterspell.go`, `internal/core/class_reform_test.go`, `internal/content/class_pools_test.go`: 반계 재현 데모 및 전투·병종 시험.

## 검증
- 관련 특성·반계·GUI 단위 시험 통과: `go test ./internal/content ./internal/core ./internal/gui -run 'TestClassPoolsFollowTraitDesign|TestFormationAndDeputyStatIncrease|TestReprisalAccumulatesOnlyDamageAndExpires|TestReflectedCriticalSpellLeavesMark|TestReflectReturnsTheSpell|TestSplitCounters|TestOfficerTraits|TestUIContentFits' -count=1`.
- 추가 보복 공격·반격 구분 시험 통과.
- 전체 core 패키지는 `TestEveryCampaignDuelStartsWhenItsOfficersMeet/jiangxia-zhaoyun-zhangwu` InvalidDeployment, `TestNewAt/the_story_before_counts_as_played` 손건 관련 실패, `TestJumpIntoTheOtherTelling/c27v-start` UnknownWaypoint. 공유 작업 트리에 별도 시나리오 변경이 있고 인과 관계는 미확인.
- content 소스 순수성 검사는 `internal/gui/uidemo_scrollbar.go`의 `c25-battle` 문자열로 실패. 새 반계 데모에서 나온 유사 오류는 해결.
- `go run ./cmd/_balance -player ai -stage B10 -seeds 1`: 0/1 승리. 단일 시드라 비교 근거로 부족.
- VM 캡처는 Mac 잠금으로 `no monitor was found at initializeGLFW` 실패. 반계·막 종료 카드의 실제 화면 미검증.

## 남은 사항
- 궁기병의 기마사격과 병종별 3·5·5의 나머지 후보는 미구현. 심공 10%도 전역 약화와 별도 병종 특성 사이에서 미결이라 변경하지 않았다.
- 법정 네 번째 특성, 세 인물 합류·출전과 초상, 방술사 명칭 변경은 미정.
- 반계의 범위 효과·애니메이션·연쇄와 보복의 면역·범위·다단, 부관 조합 및 전체 밸런스의 추가 검증 필요. Mac 잠금 해제 후 화면 캡처 필요.
