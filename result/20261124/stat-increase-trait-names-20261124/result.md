# 상급 병종 단련 특성 명칭 확인 결과

## 결론

요청의 전제와 현재 데이터가 맞지 않는다. `traits.json`(190개)과 `classes.json`에 "단련" 특성이 없다. 명칭 변경 대상이 데이터에 없으므로 바꾸지 않았다.

## 확인

- `ver5/assets/data/traits.json`: 이름에 "단련"을 포함한 항목 0개. 키도 `train`, `drill`, `boost`를 포함한 게 없다.
- `ver5/assets/data/classes.json`: "단련" 문자열 0회.
- 앞선 병종 특성 감사(`result/20261124/class-trait-additions-audit/audit.md`)도 단련이 어디에도 없다고 기록했다.

## 기존 명칭 기록과 관계

- `result/20261124/stat-increase-trait-names/notes.md`는 공격력 증가·방어력 증가 같은 이름으로 바꾸는 방향과 "주장·부관 각각 +10%, 합산 +20%" 규칙을 적었다. 이 규칙은 아직 데이터에 구현되어 있지 않다.
- 명중 관련 특성은 공격명중률증가(`accuracy_up`), 방어확률증가(`block_up`)로 이미 있다.

## 확인 필요

- 단련이 아직 데이터에 추가되지 않은 제안 특성인지, 다른 이름으로 존재하는지.
- 단련을 새로 만든다면 명칭과 합산 규칙을 데이터에 어떻게 넣을지.
