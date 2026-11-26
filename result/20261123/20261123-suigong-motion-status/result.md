# 수공 움직임 수정 완료 여부 (상태 조회)

코드·파일은 바꾸지 않았다. 진행 중인 수공 작업은 없다(가장 최근 수공 result가 `20261123-suigong-motion`, 이후 `20261123-changban-redcliff`는 다른 주제).

## 결론: 세 가지 모두 완료로 보고됨
`result/20261123/20261123-suigong-motion/result.md` 기준.
- 흰색 페이드: 완료. 물 장면 4.4초로 연장, 흰색 1초(smoothstep), 컷+4.4초에 전 픽셀 #ffffff 확인, 정지한 물 프레임 없음.
- 둑 파편 회전: 완료. 판자를 강체 상자로 풀어 yaw/pitch/roll 회전(tools/floodsim.py), 굽기 후 프레임 750 이후 각도 다양함 확인. 한계: 판자끼리·둔덕 충돌 없음, 수류에 맞춘 모델이며 강체 해석 아님.
- 조조군 행군: 완료. 1.25칸/초 NW 실제 이동, 걷기 0.35배, 지도 폭 40, `beforeDraw` 훅 추가.

## 확인한 산출물 (존재 확인)
suigong.mp4(19.32초), ford_fade_sheet.png, dam_rotation_sheet.png, march_sheet.png, frames/, suigong.sfx, result.md

## 검증 (해당 보고서 기록, 이번에 재실행하지 않음)
- go vet ./internal/gui/ 통과; go test -p 1 -run 'TestSuigong|Surge' ./internal/gui/ 통과(TestSuigongMarch 포함); 실제 캡처 성공.
- 미확인: 귀로 들은 소리 균형, 폰 해상도, 25장 본편 연결, 전 프레임 픽셀 차이 비교.

## 참고
요청 문서는 결과 경로를 `suigong-motion-status/summary.md`로 적었고 같은 내용의 summary.md가 이미 있다(17:22). 이번 result.md는 지시된 폴더에 별도 작성.
