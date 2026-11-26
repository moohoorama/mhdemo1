# 새 세션 요청 처리 및 MCP 대기 안내 결과

## 확인

- `tools/request_mcp/server.py`의 `wait_for_request(timeout_seconds=240)`: 요청이 오면 `request/`의 md를 `process/<name>/`으로 옮기고, 내용과 `result/YYYYMMDD/<name>/` 경로를 돌려준다.
- 대기 시간은 `max(10, min(값, 600))`으로 10~600초로 제한된다. 시간이 지나면 `status: "timeout"`이 오고, 그때 다시 호출해 계속 기다린다.
- 결과 폴더 이름은 `result`다(`response`가 아님).
- 이 MCP 도구는 현재 세션에 연결되어 있다. 새 세션에서는 서버를 등록해야 쓸 수 있다.

## 새 세션에서 쓸 프롬프트 (제안)

```
~/orca/mhdemo1/request/ 에 새 md 요청이 생기면 처리해줘.
1. request-watch의 wait_for_request(timeout_seconds=240)을 호출해 요청을 기다린다. 도구가 없으면 먼저 tools/request_mcp/server.py를 MCP 서버로 등록하고, 같은 이름의 처리기가 없는지 확인한다.
2. status가 "request"이면 content를 읽고 result_dir 아래 result.md를 만든다. 요청의 작업을 수행하고 결과·검증·한계를 한글로 적는다.
3. status가 "timeout"이면 다시 호출해 계속 기다린다. 내가 멈추라고 하면 멈춘다.
```

## 한계

- 이 대화에서 확인한 것은 서버 코드와 요약 문서다. 새 세션에서 실제로 도구가 등록되는지는 확인하지 못했다.
