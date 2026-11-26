# 테스트 기록: npc-trait-prerequisites-2

요청 파일: /Users/younwookim/orca/mhdemo1/test/npc-trait-prerequisites-2/npc-trait-prerequisites-2.md
진행 기록: /Users/younwookim/orca/mhdemo1/process/npc-trait-prerequisites-2/progress.md

테스트 단계 진입: 2026-11-26 07:13:37 KST(Asia/Seoul, UTC+09:00)

## [npc-prerequisites] PASS

- 시작: 2026-11-26 07:13:37 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:13:45 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test ./internal/content ./internal/core -run TestResolveNPCPlannedPrerequisites|TestNPCPlannedPrerequisitesAtSpawn|TestOfficerPlannedTraitPrerequisites|TestOfficerOwnedTraitPrerequisites|TestClassPlannedTraitPrerequisites|TestPrerequisitesGateLearning|TestPoisonAndStrictOrderPrerequisites|TestRepeatingCrossbowPlanOrder -count=1` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/content	4.979s
ok  	srpg/internal/core	1.533s
```

## [spawn-equipment-skills] PASS

- 시작: 2026-11-26 07:14:39 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:14:44 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test ./internal/core -run ^TestNPCPlannedPrerequisitesAtSpawn$ -count=1` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 0

```
ok  	srpg/internal/core	1.798s
```

## [content-core-packages] FAIL

- 시작: 2026-11-26 07:13:49 KST(Asia/Seoul, UTC+09:00)
- 종료: 2026-11-26 07:15:30 KST(Asia/Seoul, UTC+09:00)
- 명령: `go test ./internal/content ./internal/core -count=1` (cwd `/Users/younwookim/orca/mhdemo1/ver5`)
- 필수: 예
- 종료 코드: 1

```
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	liubei
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"deputy_appoint", "aim_prep", "suppressing_shot", "mounted_shot", "close_shot", "dash", "accuracy_up", "grass_footing", "reload_prep", "fire_arrow_art", "rapid_art", "agility_increase"} should not contain "suppressing_shot"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	xiahouyuan
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"spirit_prep", "counter_art", "fire", "deputy_appoint", "diagonal", "first_aid", "drain", "spirit_spread", "palm_wind", "fire_boost", "counter_boost", "agility_increase"} should not contain "fire"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	fangong
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"fire", "relief", "foresight", "deputy_appoint", "spread", "water", "earth", "focus", "terrain_craft", "fire_boost", "water_boost", "earth_boost", "mind_increase"} should not contain "earth"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	fazheng
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"fire", "relief", "foresight", "deputy_appoint", "spread", "water", "earth", "focus", "terrain_craft", "fire_boost", "water_boost", "earth_boost", "mind_increase"} should not contain "water"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	xunyou
        traits_test.go:506: 
            	Error Trace:	/Users/younwookim/orca/mhdemo1/ver5/internal/core/traits_test.go:506
            	Error:      	[]string{"spirit_prep", "counter_art", "fire", "deputy_appoint", "diagonal", "first_aid", "drain", "spirit_spread", "palm_wind", "fire_boost", "counter_boost", "agility_increase"} should not contain "fire"
            	Test:       	TestOfficerReformBatch/personal_traits_do_not_repeat_inherited_class_learning
            	Messages:   	xusi
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
FAIL
FAIL	srpg/internal/core	99.299s
FAIL
```

### 2026-11-26 07:15:34 KST(Asia/Seoul, UTC+09:00)

기존 중복 금지 검사는 유지했다. 기존 데이터의 순유·법정·유비·허사·유벽·황보숭·하후연·번궁 중복 8건은 이번 요청 범위 밖이며 별도 수정이 필요하다.

### 2026-11-26 07:15:46 KST(Asia/Seoul, UTC+09:00)

실패 상태: content/core 전체 검사에서 기존 고유·병종 특성 중복 8건이 실패한다.
다음 조치: 장수별 특성 배치 수정안 결정 후 데이터를 수정하고 전체 검사를 재실행한다.

