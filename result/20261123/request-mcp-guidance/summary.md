# 새 세션 요청 처리와 MCP 대기 안내

기존 `tools/request_mcp/server.py`를 확인했다. `wait_for_request`는 request의 md를 기다려 process/<name>/으로 옮기고, 내용과 result/YYYYMMDD/<name>/ 경로를 돌려준다. 결과 폴더 이름은 response가 아닌 result다. 대기 시간은 10~600초이며 timeout 후 재호출해야 한다. 현재 대화에 이 MCP 도구는 연결되어 있지 않다. 새 세션에 서버를 등록하고 도구 사용 가능 여부 및 중복 처리기 유무를 먼저 확인하는 프롬프트를 제안했다.
