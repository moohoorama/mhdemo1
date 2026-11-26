# 검증 결과: 병종 특성 및 이동 보너스 정리 문서의 현재 구현 주장

변경한 코드·데이터·기존 문서는 없다. 모든 판정은 코드와 데이터를 읽어 얻었다.

## 주장별 판정

| 주장 | 판정 | 근거 |
|---|---|---|
| 돌진은 이동 칸당 공격력 +5% | 확인 | `ver5/assets/data/traits.json:239-247` (`charge_strike`, `charge_power: 0.05`, "이번 차례에 이동한 칸 수 × 공격력 +5%"). 적용은 `ver5/internal/core/rules.go:78` (`FxChargePower*u.Steps`). |
| 이동 칸 수는 이동 시 기록됨 | 확인 | `ver5/internal/core/engine.go:837` (`u.Steps = len(path) - 1`). |
| 돌진은 아군 차례 종료 시 초기화되어 적 반격에 영향 없음 | 확인 | `ver5/internal/core/combat.go:1286-1287` (`u.WalkedLastTurn = u.Steps > 0; u.Steps = 0`, 주석: 돌진·기습은 자기 차례 걸음만 센다). 차례 시작 쪽에서도 `combat.go:1348` 에서 0으로 만든다. |
| 이를 검증하는 기존 테스트 | 확인 | `ver5/internal/core/traits_test.go:395-406` `TestChargeEndsWithTheWalkersTurn`. 아래 실행 결과 PASS. |
| 중기병·친위대에 돌진이 있다 | 불일치 | 친위대만 있다: `ver5/assets/data/classes.json:485-513` (Pool에 `charge_strike`, 512행). 중기병(`classes.json:315`)의 Pool은 `deputy_appoint, counter_ward, magic_ward`, Plan은 `grass_footing, dash` 뿐이라 돌진이 없다(340-353행). 문서는 이를 "제안"으로 적었으므로 문서 자체의 오류는 아니지만, "현재 있다"로 읽으면 틀리다. |
| 전차대는 돌진이 없다 | 불일치 | 전차대에 돌진이 있다: `classes.json:154-199` (Pool 180행 `charge_strike`, Plan Lv40 199행). 문서의 "전차대에서는 제거"는 제안으로서 타당하고, 현재는 있다. |
| 방진이 문서의 "밀집대형"의 현재 이름 | 확인 | `traits.json:1886-1894` (`formation_guard`, 이름 "방진", 주변 8칸 아군 1기당 방어력 +4%, `formation_defense: 0.04`). 중보병 Pool에 있다(`classes.json:91-95`). 비용은 이 구간에서 1000이다. 문서의 비용 800 제안은 변경안이다. |
| 저지 사격이 문서 "견제"의 현재 이름 | 확인(이름 충돌 주의) | `traits.json:1786-1793` (`suppressing_shot`, "저지 사격", 직접 원거리 물리 첫 타격 명중 시 다음 차례 이동력 -2, `ranged_slow: 2`). 현재 궁기병(Pool·Plan `classes.json:390,405`)과 노기병(Plan `classes.json:474`)에 있다. 문서가 말한 장창병은 현재 없다. 주의: 이름 "견제"는 이미 `traits.json:1252` `hinder`("디버프계 책략을 쓴다", 책략 계열)가 쓰고 있어, 개명하면 이름이 충돌한다. |
| 원거리 물리 피해 30% 감소 특성 존재 | 확인 | `traits.json:1484-1492` `ranged_ward`, 이름 "원거리피해감소", `ranged_ward: 0.3`. 적용 `combat.go:65`. 현재 소유 병종은 중보병(Pool, `classes.json:94`)과 근위병(Plan, `classes.json:255`)이고 연노병이 아니다. 문서는 연노병이 아니라 방패술이 대체할 대상으로 적었으므로 일치한다. 거울(`traits.json:1494-1503`)이 -50%를 더한다. |
| 연노병 Pool 5개, 노병 Pool 4개 | 확인 | `classes.json`의 `repeating_crossbow`(632행) Pool 5개, `crossbow` Pool 4개(스크립트로 집계). |
| 방패술이 존재한다 | 코드에 없음 | `assets/data/*.json` 전체에서 "방패술"과 `shield_art`를 검색해 0건. 문서도 "이름·효과 변경 제안"으로 적었다. 현재는 원거리피해감소가 해당 효과다. |
| 기세 | 코드에 없음 | `traits.json`에 "기세" 이름이 없다. 문서는 만들지 않는다고 적었으므로 일치한다. |

## 불일치 요약
1. 전차대에 돌진이 현재 있다(`classes.json:180,199`). 요청이 말한 "전차는 없다"는 틀렸고, 문서의 "제거"는 제안이다.
2. 중기병에는 돌진이 현재 없다(`classes.json:340-353`). 친위대에만 있다.
3. 문서의 "견제" 개명은 기존 책략 특성 `hinder`(이름 "견제", `traits.json:1252`)와 충돌한다.
4. 저지 사격 현재 소유는 궁기병·노기병이고, 장창병은 없다.
5. 방패술은 아직 없고 원거리피해감소(`ranged_ward`)가 현재 효과다. 소유는 중보병·근위병이다.

## 테스트
`RF begin-test` 로 test/ 단계로 이동한 뒤 아래를 실행했다.

```
requestflow.py run-test 20261126-class-trait-movement-revision --label charge-ends --cwd /Users/younwookim/orca/mhdemo1/ver5 -- go test -p 1 -v -run TestChargeEndsWithTheWalkersTurn ./internal/core
```
출력(test/.../test-log.md 에서 발췌):
```
=== RUN   TestChargeEndsWithTheWalkersTurn
--- PASS: TestChargeEndsWithTheWalkersTurn (0.01s)
PASS
ok  	srpg/internal/core	0.628s
```
종료 코드 0, PASS.
