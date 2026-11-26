# 수공 행군·물소리 작업 완료 여부 (상태 조회)

결론: 완료됨. 진행 중인 작업 없음. 코드·파일은 바꾸지 않았다.

근거: `result/20261123/20261123-suigong-march-water-audio/result.md` (17:45).
- 행군: 장면 엔진 걷기(칸 단위 walkTo, 3.57칸/초)와 보병 step·기병 hoof 발소리 구현. TestSuigongMarch 통과.
- 둑 붕괴 후 2배: surgeRate=2 반영(둑 2.5->1.25초, 여울 4.4->2.2초, 흰색 컷+2.2초에 완전 흰색).
- 물소리: flood-surge/crash/spray 새 cue 3개로 재구성, 영상 믹스 클리핑 없음.
- 검증: go vet, `go test -p 1 -run 'TestSuigong|Surge' ./internal/gui/` 통과, 945프레임 캡처·더빙 성공. 전체 테스트는 안 돌림.
- 한계(해당 result 기재): 귀로 직접 듣지 못하고 수치로만 확인. 발 닿는 순간과 소리의 프레임 단위 일치는 아님.
- 산출물: 위 폴더의 suigong.mp4, 시트 png 3장, suigong.sfx.

다른 수공 subagent 작업: 17:45 이후 새 result 없음, 진행 중으로 보이지 않음.
요청이 지정한 `suigong-march-status/summary.md`는 이미 존재(17:51).
