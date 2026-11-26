# 요청 기록 파일 열기 오류 - 결과

## 원인
- 이전 답변이 가리킨 `request/20261123-request-recording.md`는 사라진 것이 아니라 **request-watcher가 옮긴 것**이다. 스킬 절차 2a가 새 md를 `request/`에서 `process/<name>/`으로 `mv`하므로, 처리가 시작되면 `request/`는 비고 파일은 `process/<name>/<name>.md`에만 있다.
- 지금 확인: `request/`는 비어 있고, `process/20261123-request-link-repair/`, `process/20261123-request-recording/`, `process/20261123-request-recording-r2/`, `process/20261123-result-inventory/`에 각 요청 파일이 있다.
- 따라서 `request/...` 경로를 링크로 보여 주면 거의 항상 "파일 없음"이 된다. 이전 작업이 파일을 "다시 만든" 것은 임시 대응이었고, 그 사본도 watcher가 다시 옮긴다.

## 바꾼 것
- `/Users/younwookim/orca/mhdemo1/AGENTS.md`의 요청 기록 규칙에 한 단락 추가: 요청 기록을 안내할 때는 `request/`가 아니라 존재를 확인한 `process/<name>/<name>.md` 경로를 링크한다.

## 참고(미수정)
- 요청 파일이 결과 폴더를 `result/20261123/request-link-repair/`(접두어 없음)로 적었지만 watcher는 `20261123-request-link-repair/`를 만든다. 두 이름이 따로 생겨 `request-recording`, `result-inventory`도 폴더가 이중으로 있다. 어느 쪽을 정본으로 할지는 정하지 않았다(질문: 접두어 있는 폴더로 통일할까요?).
- 모바일에서 열리는 링크가 로컬 경로 기반인지, 웹뷰 URL인지는 확인하지 못했다. 모바일이 `process/`나 `result/`를 서빙하는지는 별도 확인이 필요하다.

## 검증
- 코드 변경 없음, 테스트·빌드 실행 안 함. 파일 존재 여부는 `ls`로만 확인.
