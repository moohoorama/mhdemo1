"""request/ 폴더에 새 md 요청이 올 때까지 기다렸다가 내용을 돌려주는 MCP 서버.

모델은 이 도구를 호출해 요청을 받고, 세션 안에서 직접 수행한 뒤 result.md를 쓴다.
서버 자체는 파일 확인과 이동만 하므로 기다리는 동안 토큰을 쓰지 않는다.

이동·기록·테스트 단계 규칙은 requestflow.py가 맡는다 (스킬 CLI와 같은 코드).
요청을 받으면 process/<name>/progress.md에 1분마다 실제 시각으로 진행을 남기는 heartbeat가
세션과 분리된 프로세스로 뜬다. 작업이 끝나면 begin_test -> run_test -> complete_request 순서로
요청 md를 test/ 로, 모든 필수 테스트 통과 후에만 result/ 로 옮긴다.
"""

import sys
import time
from pathlib import Path

from mcp.server.fastmcp import FastMCP

sys.path.insert(0, str(Path(__file__).resolve().parent))
import requestflow as rf  # noqa: E402

REQUEST_DIR = rf.root_dir() / "request"
POLL_SECONDS = 2
RECOVER_SECONDS = 60

mcp = FastMCP("request-watch")


def _pending() -> list[Path]:
    REQUEST_DIR.mkdir(parents=True, exist_ok=True)
    return sorted(p for p in REQUEST_DIR.glob("*.md") if p.is_file())


@mcp.tool()
def list_pending() -> list[str]:
    """request/ 에서 아직 처리되지 않은 md 요청 파일 이름 목록을 돌려준다."""
    return [p.name for p in _pending()]


@mcp.tool()
def wait_for_request(timeout_seconds: int = 240) -> dict:
    """새 md 요청이 request/ 에 생길 때까지 기다린다.

    요청이 오면 process/<name>/ 로 옮기고(덮어쓰지 않음, 이름이 겹치면 -2 접미사) 진행 기록과
    1분 heartbeat를 시작한 뒤 요청 내용을 돌려준다. 돌려준 name 을 이후 도구에 그대로 쓴다.
    시간 안에 요청이 없으면 status 가 "timeout" 이다. 그때는 다시 호출해 계속 기다린다.
    """
    timeout = max(10, min(int(timeout_seconds), 600))
    deadline = time.monotonic() + timeout
    next_recover = 0.0

    while True:
        if time.monotonic() >= next_recover:
            rf.recover()
            next_recover = time.monotonic() + RECOVER_SECONDS
        for md in _pending():
            p = rf.claim(md)
            if p is None:
                continue
            return {
                "status": "request",
                "name": p.name,
                "process_path": str(p.process_md),
                "result_dir": str(p.result_dir()),
                "progress_path": str(p.progress),
                "content": p.process_md.read_text(encoding="utf-8"),
                "flow": "작업 중 status로 현재 상태를 남기고, 구현이 끝나면 begin_test, "
                "테스트는 run_test로 실행·기록, 필수 테스트가 모두 통과하면 complete_request.",
            }
        if time.monotonic() >= deadline:
            return {"status": "timeout", "waited_seconds": timeout}
        time.sleep(POLL_SECONDS)


@mcp.tool()
def set_status(name: str, state: str, doing: str = "", next_step: str = "", blocker: str = "") -> str:
    """요청의 현재 상태를 남긴다 (heartbeat가 1분마다 이 내용을 진행 기록에 싣는다).

    state: working / testing / done / failed / blocked. doing 수행 내용, next_step 다음 단계, blocker 장애.
    """
    rf.set_status(name, state, doing, next_step, blocker)
    return "ok"


@mcp.tool()
def begin_test(name: str) -> str:
    """구현이 끝났을 때 요청 md를 process/<name>/ 에서 test/<name>/ 으로 옮기고 test-log.md를 만든다."""
    return str(rf.begin_test(name).test_md)


@mcp.tool()
def run_test(name: str, label: str, command: list[str], required: bool = True, cwd: str = "") -> dict:
    """테스트 명령을 실제로 실행하고 시각·명령·종료 코드·출력 끝을 test/<name>/test-log.md에 기록한다.

    같은 label을 다시 실행하면 최근 결과만 완료 판정에 쓰인다(재시험). 실패하면 원인을 note로 남기고 고친 뒤 재실행한다.
    """
    code = rf.run_test(name, label, command, required=required, cwd=cwd or None)
    return {"exit": code, "verdict": "PASS" if code == 0 else "FAIL"}


@mcp.tool()
def note_test(name: str, text: str) -> str:
    """test-log.md에 시각을 붙여 버그·원인·수정 내용 같은 메모를 남긴다."""
    rf.note(name, text)
    return "ok"


@mcp.tool()
def complete_request(name: str, no_test_reason: str = "") -> dict:
    """모든 필수 테스트의 최근 결과가 PASS일 때만 요청 md를 result/YYYYMMDD/<name>/ 으로 옮긴다.

    통과하지 못하면 옮기지 않고 test/ 에 실패 상태를 기록한다. 비코드 요청은 no_test_reason(테스트가 필요 없는 이유)을 준다.
    """
    try:
        return {"completed": True, "path": str(rf.complete(name, no_test_reason))}
    except ValueError as e:
        return {"completed": False, "reason": str(e)}


@mcp.tool()
def fail_request(name: str, reason: str, next_action: str) -> str:
    """필수 검증이 미실행·불가능·실패일 때 test/ 에 머물며 사유와 다음 조치를 기록한다."""
    rf.fail(name, reason, next_action)
    return "ok"


if __name__ == "__main__":
    rf.recover()
    mcp.run()
