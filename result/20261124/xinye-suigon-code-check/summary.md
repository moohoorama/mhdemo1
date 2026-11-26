# 신야 수공 본편 적용 확인

작업 트리에서 story25의 c25-battle이 B25를 가리키고 stage25의 suigong 이벤트가 surge와 조홍을 지정한다. GUI cutscene과 field_surge_show에서 수공 연출을 실행하고 ch25의 두 둑 장면은 S_baihe_dike를 사용한다. 수공 데이터 파일은 디스크에 존재하며 시나리오 파일은 수정 상태, 수공 에셋 폴더는 untracked이다. `go test -p 1 -count=1 -run 'TestSuigong|TestSurge' ./internal/gui ./internal/core` 통과. 새 GUI 캡처나 전투 전체 플레이는 이번 확인에서 수행하지 않았다.
