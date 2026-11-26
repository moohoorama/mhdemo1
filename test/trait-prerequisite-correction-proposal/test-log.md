# 테스트 기록: trait-prerequisite-correction-proposal

요청 파일: /Users/younwookim/orca/mhdemo1/test/trait-prerequisite-correction-proposal/trait-prerequisite-correction-proposal.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/trait-prerequisite-correction-proposal/progress.md

테스트 단계 진입: 2026-11-26 06:41:01 KST(Asia/Seoul, UTC+09:00)

## [병종고유중복] FAIL

- 시작: 2026-11-26 06:41:12 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 06:41:15 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test ./internal/core -run ^TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning$ -count=1` (cwd `ver5`)
- 필수: 예
- 종료 코드: 1

```
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	xiahouyuan
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"spirit_prep", "counter_art", "fire", "deputy_appoint", "diagonal", "first_aid", "drain", "spirit_spread", "palm_wind", "fire_boost", "counter_boost", "agility_increase"} should not contain "fire"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	xusi
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"spirit_prep", "counter_art", "fire", "deputy_appoint", "diagonal", "first_aid", "drain", "spirit_spread", "palm_wind", "fire_boost", "counter_boost", "agility_increase"} should not contain "fire"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	fangong
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"deputy_appoint", "iron_armor", "ranged_ward", "formation_guard", "diagonal", "fire", "brace_prep", "castle_defense", "siegecraft", "composure", "castle_mind", "defense_increase"} should not contain "deputy_appoint"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	liubei
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"fire", "relief", "foresight", "deputy_appoint", "spread", "water", "earth", "focus", "terrain_craft", "fire_boost", "water_boost", "earth_boost", "mind_increase"} should not contain "earth"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	fazheng
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"counter_art", "earth", "plunder", "deputy_appoint", "venom", "ambush_crit", "diagonal", "first_aid", "vital_strike", "crit_boost", "relief", "attack_increase", "drain"} should not contain "earth"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	liupi
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"deputy_appoint", "iron_armor", "ranged_ward", "formation_guard", "diagonal", "fire", "brace_prep", "castle_defense", "siegecraft", "composure", "castle_mind", "defense_increase"} should not contain "fire"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	huangfusong
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"fire", "relief", "foresight", "deputy_appoint", "spread", "water", "earth", "focus", "terrain_craft", "fire_boost", "water_boost", "earth_boost", "mind_increase"} should not contain "water"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	xunyou
FAIL
FAIL	srpg/internal/core	1.114s
FAIL
```

### 2026-11-26 06:41:25 KST(Asia/Seoul, UTC+09:00)

실패 상태: 복구한 중복 금지 테스트가 현재 데이터의 8건 중복을 검출함; 사용자가 요청한 범위는 수정안 제시까지임
다음 조치: 수정안 결정 후 특성 데이터를 재배치하고 중복·선행 검사를 함께 통과시킨다

