# 설계: 궁기병 최소 사거리와 근접사격

## 조사 결과
- 모든 기본 공격 사거리는 `core/rules.go`의 `rangedBand`(lo~hi) 한 곳에서 정한다. `weaponReach`/`reachesFrom`/`inRange`(공격·반격·협동·AI·이동 후 사거리), `rangeFor`/`AttackRange`(GUI 표시·AI 추정)가 모두 이를 지난다. 원거리 판정(`rangedAttack`)은 병종 `Attack==ranged`이면 거리와 무관하게 원거리이므로 거리 1에서도 원거리 피해·명중·반격 규칙이 그대로 적용된다.
- 궁기병(horse_archer)·노기병(horse_crossbow)만 `MinRange: 1`(데이터)이고 궁병 계열·발석거는 2. 같은 계열 승급은 노기병 하나뿐이다.
- 특성은 `traits()`가 주장+부관(생존) 특성 id를 집합으로 합치고 `fx`가 id당 한 번만 합산하므로, 주장·부관 양쪽 보유 시 중복 없음, 부관 사망/해임 시 즉시 소멸(`!d.Dead`, Deputy 해제)한다.

## 변경
1. `classes.json`: horse_archer·horse_crossbow `MinRange` 1 -> 2 (최대 사거리 3 유지, 다른 병종 불변).
2. `traits.json`: `close_shot`(근접사격, Cost 800, 효과 `close_shot: 1`). 궁기병·노기병 학습 풀에 추가.
3. `content/vocab.go`: `FxCloseShot = "close_shot"`을 효과 키 목록에 등록.
4. `core/rules.go rangedBand`: `Attack==ranged` 병종에서 `close_shot`이 있으면 최소 거리를 1(스킬 자체 Min은 유지)로. 근접 병종은 이 분기에 들지 않으므로 무의미한 사격 능력이 생기지 않는다. 기사(mounted_range) 경로·책략 사거리(`skillBand`)는 건드리지 않는다.
5. 효과가 `rangedBand`에 있어 반격·AI·GUI·사거리 표시는 자동 반영된다(별도 수정 없음, 테스트로 확인).
6. 문서: logic-traits.md(표·특성), logic-combat.md(사거리), history.md. 테스트: class_pools_test 갱신, core 신규 테스트.
