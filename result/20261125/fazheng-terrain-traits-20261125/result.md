# 법정 산악지형보조 / 지형 특성 확인

## 한 일
요청 파일은 기존 기록(result/20261125/fazheng-terrain-traits/summary.md)을 가리키는 한 줄이었다. 코드를 확인한 결과 적용은 이미 작업 트리에 들어 있어 이번에 새로 고친 파일은 없다.

- assets/data/characters_story.json: 법정 Traits·Learn = crush, earthcraft, reprisal, mountain_footing (네 번째 고유 특성 확인).
- assets/data/traits.json: mountain_footing "산악지형보조", ambush_factor 0.05, 설명 "구릉·숲에 서면 지형 보정 +5%(공격·방어 모두)".
- internal/core/rules.go:141: 현재 코드는 `props.Ambush || props.Highland`에서 작동한다. 구릉에서도 작동하므로 summary.md의 "구릉에서는 작동하지 않는다"는 문장은 현재 코드와 맞지 않는다(오래된 서술). 설명문(구릉·숲)이 맞다.

## 지형 보조 특성 (traits.json 기준)
- 초원지형보조: 맨 초원 +5%
- 산악지형보조: 구릉·숲 +5% (법정 등)
- 전지형보조: 모든 지형 +5%
- 수영: 물·여울·물웅덩이 +20%, 수계 책략 피해 절반
- 답파(지형 하한 100), 공성(적 지형 상한 100), 험지이동, 지형 조성, 지형정비는 summary.md 참고.
보유자 전체 목록은 이번에 다시 뽑지 않았고 summary.md를 그대로 따른다(재검증하지 않음).

## 검증
`go test -p 1 ./internal/content ./internal/core -run 'TestFazhengLearnsMountainFooting|TestOfficerTraitDecisions|TestOfficerTraits' -count=1`
- ok srpg/internal/content
- ok srpg/internal/core
