"""request -> process -> test -> result 흐름을 안전하게 옮기고 기록하는 stdlib 전용 모듈/CLI.

단계(요청 md가 있는 곳)
  request/<name>.md            대기
  process/<name>/<name>.md     구현·작업 중
  test/<name>/<name>.md        테스트 중 (실패하면 여기 머문다)
  result/YYYYMMDD/<name>/      모든 필수 테스트 통과 후에만

기록
  process/<name>/progress.md   1분마다 실제 시각과 상태를 덧붙이는 진행 기록 (끝까지 보존)
  process/<name>/status.json   작업자가 `status`로 남긴 현재 수행·다음·장애
  test/<name>/test-log.md      실행한 테스트마다 시각·명령·결과·원인·수정
  test/<name>/test-results.jsonl  기계가 읽는 테스트 결과 (완료 조건 판정용)

heartbeat는 작업 세션과 별개의 분리된 프로세스라 긴 테스트 중에도 계속 돌고,
flock으로 요청당 하나만 돈다. 시간이 지났다는 이유만으로 진척을 꾸며 쓰지 않는다:
작업자의 마지막 status 시각이 오래됐으면 그 사실을 그대로 적는다.
"""

import argparse
import datetime
import fcntl
import json
import os
import shutil
import signal
import subprocess
import sys
import threading
import time
import uuid
from pathlib import Path
from zoneinfo import ZoneInfo

DEFAULT_ROOT = "/Users/younwookim/orca/mhdemo1"
TZ = ZoneInfo("Asia/Seoul")
TERMINAL_STATES = {"done", "failed", "blocked"}
STALE_STATUS_SECONDS = 300


def root_dir() -> Path:
    return Path(os.environ.get("REQUEST_FLOW_ROOT", DEFAULT_ROOT))


def heartbeat_interval() -> float:
    return float(os.environ.get("REQUEST_FLOW_HEARTBEAT_SECONDS", "60"))


def now() -> datetime.datetime:
    return datetime.datetime.now(TZ)


def stamp(t: datetime.datetime | None = None) -> str:
    t = t or now()
    return t.strftime("%Y-%m-%d %H:%M:%S") + " KST(Asia/Seoul, UTC+09:00)"


def short_stamp(t: datetime.datetime | None = None) -> str:
    return (t or now()).strftime("%Y-%m-%d %H:%M:%S KST")


class Paths:
    def __init__(self, name: str, root: Path | None = None):
        self.root = root or root_dir()
        self.name = name
        self.request = self.root / "request" / f"{name}.md"
        self.process_dir = self.root / "process" / name
        self.process_md = self.process_dir / f"{name}.md"
        self.progress = self.process_dir / "progress.md"
        self.status = self.process_dir / "status.json"
        self.meta = self.process_dir / "meta.json"
        self.lock = self.process_dir / ".heartbeat.lock"
        self.test_dir = self.root / "test" / name
        self.test_md = self.test_dir / f"{name}.md"
        self.test_log = self.test_dir / "test-log.md"
        self.test_results = self.test_dir / "test-results.jsonl"

    def read_meta(self) -> dict:
        return json.loads(self.meta.read_text(encoding="utf-8"))

    def result_dir(self) -> Path:
        return Path(self.read_meta()["result_dir"])

    def stage(self) -> str:
        if self.meta.exists() and (self.result_dir() / f"{self.name}.md").exists():
            return "result"
        if self.test_md.exists():
            return "test"
        if self.process_md.exists():
            return "process"
        if self.request.exists():
            return "request"
        return "unknown"


def move_no_clobber(src: Path, dst: Path) -> None:
    """src를 dst로 옮긴다. dst가 이미 있으면 덮지 않고 FileExistsError. 동시에 둘이 해도 한 쪽만 성공한다."""
    dst.parent.mkdir(parents=True, exist_ok=True)
    os.link(src, dst)
    try:
        os.unlink(src)
    except FileNotFoundError:
        pass


def append_line(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd = os.open(path, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o644)
    try:
        os.write(fd, text.encode("utf-8"))
    finally:
        os.close(fd)


def log_progress(p: Paths, kind: str, text: str) -> None:
    text = " ".join(text.split()).replace("|", "/")
    append_line(p.progress, f"| {stamp()} | {kind} | {text} |\n")


def _ensure_progress_header(p: Paths) -> None:
    header = (
        f"# 진행 기록: {p.name}\n\n"
        "시각은 기록한 순간의 실제 시계(Asia/Seoul)다. `heartbeat`는 1분마다 자동 기록하고,\n"
        "`status`/`단계`/`테스트` 행은 작업자·처리기가 사건이 생길 때 남긴다.\n\n"
        "| 시각 | 구분 | 내용 |\n|---|---|---|\n"
    )
    try:
        fd = os.open(p.progress, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o644)
    except FileExistsError:
        return
    with os.fdopen(fd, "w", encoding="utf-8") as f:
        f.write(header)


def adopt(name: str, root: Path | None = None, result_date: str | None = None) -> Paths:
    """이미 process/<name>/ 또는 test/<name>/에 있는 요청에 진행 기록·메타를 붙인다 (여러 번 불러도 안전)."""
    p = Paths(name, root)
    p.process_dir.mkdir(parents=True, exist_ok=True)
    if not p.meta.exists():
        date = result_date or now().strftime("%Y%m%d")
        result_dir = p.root / "result" / date / name
        result_dir.mkdir(parents=True, exist_ok=True)
        meta = {"name": name, "claimed": stamp(), "result_dir": str(result_dir)}
        tmp = p.process_dir / f".meta.{os.getpid()}.tmp"
        tmp.write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")
        try:
            move_no_clobber(tmp, p.meta)
        except FileExistsError:
            tmp.unlink(missing_ok=True)
    _ensure_progress_header(p)
    return p


def claim(request_md: Path, root: Path | None = None) -> Paths | None:
    """request/의 md를 process/<name>/으로 옮긴다.

    다른 처리기가 먼저 가져갔으면 None. 같은 이름이 process/test/result에 이미 있으면
    기록을 덮지 않고 -2, -3 접미사를 붙인 새 이름을 쓴다.
    """
    root = root or root_dir()
    base = request_md.stem
    (root / "process").mkdir(parents=True, exist_ok=True)
    held = root / "process" / f".claim.{os.getpid()}.{uuid.uuid4().hex}.md"
    try:
        os.rename(request_md, held)  # 원자적: 동시에 여러 처리기가 해도 하나만 성공한다
    except FileNotFoundError:
        return None
    n = 1
    while True:
        name = base if n == 1 else f"{base}-{n}"
        n += 1
        p = Paths(name, root)
        if p.process_dir.exists() or p.test_dir.exists() or any((root / "result").glob(f"*/{name}")):
            continue
        try:
            p.process_dir.mkdir(parents=True)
        except FileExistsError:
            continue
        os.rename(held, p.process_md)
        break
    adopt(name, root)
    log_progress(p, "단계", f"request/{request_md.name} -> process/{name}/{p.process_md.name} 이동, 작업 시작")
    set_status(name, "working", doing="요청 접수", next_step="요청 읽기", root=root)
    return p


def _read_status(p: Paths) -> dict:
    try:
        return json.loads(p.status.read_text(encoding="utf-8"))
    except (FileNotFoundError, json.JSONDecodeError):
        return {}


def _status_text(data: dict) -> str:
    parts = []
    if data.get("doing"):
        parts.append(f"수행: {data['doing']}")
    if data.get("next"):
        parts.append(f"다음: {data['next']}")
    if data.get("blocker"):
        parts.append(f"장애: {data['blocker']}")
    return " / ".join(parts) or "(내용 없음)"


def set_status(
    name: str,
    state: str,
    doing: str = "",
    next_step: str = "",
    blocker: str = "",
    root: Path | None = None,
) -> None:
    """작업자가 현재 상태를 남긴다. heartbeat가 이 내용을 1분 기록에 싣는다.

    state: working / testing / done / failed / blocked
    """
    p = adopt(name, root)
    data = {
        "state": state,
        "doing": doing,
        "next": next_step,
        "blocker": blocker,
        "updated": now().isoformat(timespec="seconds"),
    }
    tmp = p.process_dir / f".status.{os.getpid()}.tmp"
    tmp.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
    os.replace(tmp, p.status)
    log_progress(p, f"status:{state}", _status_text(data))
    if state not in TERMINAL_STATES:
        ensure_heartbeat(name, p.root)


def _lock_is_free(p: Paths) -> bool:
    p.process_dir.mkdir(parents=True, exist_ok=True)
    fd = os.open(p.lock, os.O_RDWR | os.O_CREAT, 0o644)
    try:
        fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        return False
    finally:
        os.close(fd)
    return True


def ensure_heartbeat(name: str, root: Path | None = None) -> bool:
    """heartbeat가 안 돌고 있으면 세션과 분리된 프로세스로 띄운다. 새로 띄웠으면 True."""
    p = Paths(name, root)
    if not _lock_is_free(p):
        return False
    env = dict(os.environ, REQUEST_FLOW_ROOT=str(p.root))
    child = subprocess.Popen(
        [sys.executable, str(Path(__file__).resolve()), "heartbeat", name],
        start_new_session=True,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        env=env,
    )
    threading.Thread(target=child.wait, daemon=True).start()  # 장수 프로세스에서 좀비를 남기지 않는다
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline and _lock_is_free(p):
        time.sleep(0.05)
    return True


def is_active(p: Paths) -> bool:
    if not p.meta.exists() or p.stage() not in ("process", "test"):
        return False
    return _read_status(p).get("state") not in TERMINAL_STATES


def recover(root: Path | None = None) -> list[str]:
    """새 흐름으로 접수됐지만 heartbeat가 멈춘 요청(재시작·중단 후)에 heartbeat를 다시 붙인다."""
    root = root or root_dir()
    started = []
    for meta in sorted((root / "process").glob("*/meta.json")):
        p = Paths(meta.parent.name, root)
        if is_active(p) and ensure_heartbeat(p.name, root):
            log_progress(p, "단계", "heartbeat가 멈춰 있어 다시 시작(recover)")
            started.append(p.name)
    return started


def heartbeat(name: str, root: Path | None = None) -> int:
    p = Paths(name, root)
    p.process_dir.mkdir(parents=True, exist_ok=True)
    lock_fd = os.open(p.lock, os.O_RDWR | os.O_CREAT, 0o644)
    try:
        fcntl.flock(lock_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
    except OSError:
        return 0
    os.ftruncate(lock_fd, 0)
    os.write(lock_fd, f"{os.getpid()}\n".encode())

    stopping: list[int] = []
    signal.signal(signal.SIGTERM, lambda *_: stopping.append(1))
    signal.signal(signal.SIGINT, lambda *_: stopping.append(1))

    interval = heartbeat_interval()
    log_progress(p, "heartbeat", f"시작 pid={os.getpid()} 간격={interval:g}초")
    last = time.monotonic()
    next_tick = last + interval
    while not stopping:
        while time.monotonic() < next_tick and not stopping:
            time.sleep(min(0.25, max(0.0, next_tick - time.monotonic())))
        if stopping:
            break
        tick = time.monotonic()
        gap = tick - last
        last = tick
        next_tick = max(next_tick + interval, tick + interval / 2)
        stage = p.stage()
        status = _read_status(p)
        state = status.get("state", "")
        if stage in ("result", "unknown"):
            log_progress(p, "heartbeat", f"종료: 요청이 {stage} 단계")
            return 0
        text = [f"단계={stage}"]
        if status:
            text += [f"상태={state}", _status_text(status)]
            try:
                upd = datetime.datetime.fromisoformat(status["updated"])
                age = (now() - upd).total_seconds()
                if age > STALE_STATUS_SECONDS:
                    text.append(f"작업자 마지막 보고 {short_stamp(upd)} ({int(age // 60)}분 전) 이후 새 보고 없음")
            except (KeyError, ValueError):
                pass
        else:
            text.append("작업자 상태 보고 없음")
        if gap > interval * 1.5:
            text.append(f"직전 기록과 {int(gap)}초 간격(절전 또는 중단 가능)")
        log_progress(p, "heartbeat", " / ".join(text))
        if state in TERMINAL_STATES:
            log_progress(p, "heartbeat", f"종료: 상태 {state}")
            return 0
    log_progress(p, "heartbeat", "종료: 신호 수신")
    return 0


def begin_test(name: str, root: Path | None = None) -> Paths:
    """구현이 끝나면 요청 md를 test/<name>/으로 옮긴다 (process의 진행 기록은 그대로 둔다)."""
    p = adopt(name, root)
    if not (p.test_md.exists() and not p.process_md.exists()):
        move_no_clobber(p.process_md, p.test_md)
        (p.process_dir / "MOVED.txt").write_text(
            f"요청 파일은 {stamp()}에 {p.test_md} 로 옮겼다 (테스트 단계).\n"
            f"진행 기록은 이 폴더의 progress.md에 계속 남는다.\n",
            encoding="utf-8",
        )
        log_progress(p, "단계", f"process/{name}/ -> test/{name}/{p.test_md.name} 이동, 테스트 시작")
    if not p.test_log.exists():
        append_line(
            p.test_log,
            f"# 테스트 기록: {name}\n\n요청 파일: {p.test_md}\n진행 기록: {p.progress}\n\n"
            f"테스트 단계 진입: {stamp()}\n\n",
        )
    set_status(name, "testing", doing="테스트 실행", next_step="필수 테스트 전부 통과 확인", root=p.root)
    return p


def _tail(text: str, lines: int = 40) -> str:
    return "\n".join(text.rstrip("\n").split("\n")[-lines:])


def run_test(
    name: str,
    label: str,
    command: list[str],
    required: bool = True,
    cwd: str | None = None,
    root: Path | None = None,
) -> int:
    """명령을 실제로 실행하고 시작·종료 시각, 명령, 종료 코드, 출력 끝을 test-log.md에 남긴다."""
    p = adopt(name, root)
    if not p.test_log.exists():
        raise RuntimeError("test 단계가 아니다: begin-test를 먼저 실행하세요")
    start = now()
    log_progress(p, "테스트", f"시작 [{label}] {' '.join(command)}")
    try:
        proc = subprocess.run(command, cwd=cwd, capture_output=True, text=True)
        code, output = proc.returncode, proc.stdout + proc.stderr
    except FileNotFoundError as e:
        code, output = 127, str(e)
    end = now()
    verdict = "PASS" if code == 0 else "FAIL"
    record = {
        "label": label,
        "command": command,
        "cwd": cwd,
        "required": required,
        "start": start.isoformat(timespec="seconds"),
        "end": end.isoformat(timespec="seconds"),
        "exit": code,
        "verdict": verdict,
    }
    append_line(p.test_results, json.dumps(record, ensure_ascii=False) + "\n")
    append_line(
        p.test_log,
        f"## [{label}] {verdict}\n\n"
        f"- 시작: {stamp(start)}\n- 종료: {stamp(end)}\n"
        f"- 명령: `{' '.join(command)}`" + (f" (cwd `{cwd}`)" if cwd else "") + "\n"
        f"- 필수: {'예' if required else '아니오'}\n- 종료 코드: {code}\n\n"
        f"```\n{_tail(output)}\n```\n\n",
    )
    log_progress(p, "테스트", f"종료 [{label}] {verdict} (exit {code})")
    return code


def note(name: str, text: str, root: Path | None = None) -> None:
    """test-log.md에 시각을 붙여 자유 메모(버그·원인·수정·재시험 계획)를 남긴다."""
    p = adopt(name, root)
    append_line(p.test_log, f"### {stamp()}\n\n{text.strip()}\n\n")
    log_progress(p, "테스트메모", text)


def test_verdict(p: Paths) -> tuple[bool, str]:
    """라벨별 가장 최근 결과가 모두 PASS이고 필수 테스트가 하나 이상이면 통과."""
    latest: dict[str, dict] = {}
    if p.test_results.exists():
        for line in p.test_results.read_text(encoding="utf-8").splitlines():
            if line.strip():
                rec = json.loads(line)
                latest[rec["label"]] = rec
    required = [r for r in latest.values() if r["required"]]
    if not required:
        return False, "기록된 필수 테스트가 없다"
    failing = [r["label"] for r in required if r["verdict"] != "PASS"]
    if failing:
        return False, "최근 결과가 실패인 필수 테스트: " + ", ".join(failing)
    return True, f"필수 테스트 {len(required)}개 모두 최근 결과 PASS"


def complete(name: str, no_test_reason: str = "", root: Path | None = None) -> Path:
    """필수 테스트가 모두 통과했을 때만 요청 md를 result/YYYYMMDD/<name>/으로 옮긴다.

    비코드 요청은 no_test_reason(테스트가 필요 없는 이유)을 주면 되고, 이때 test-results에
    실패 기록이 없어야 한다. 통과하지 못하면 ValueError를 내고 test/에 머문다.
    """
    p = adopt(name, root)
    if p.stage() == "result":
        return p.result_dir() / f"{name}.md"
    if not p.test_md.exists():
        raise ValueError("요청이 test 단계에 있지 않다: begin-test를 먼저 실행하세요")
    ok, why = test_verdict(p)
    if not ok:
        if no_test_reason.strip() and not p.test_results.exists():
            ok, why = True, f"테스트 불필요 사유: {no_test_reason.strip()}"
        else:
            fail(name, why, "실패 원인 수정 후 재시험", p.root)
            raise ValueError(f"완료할 수 없다: {why}")
    result_dir = p.result_dir()
    result_dir.mkdir(parents=True, exist_ok=True)
    dst = result_dir / f"{name}.md"
    if dst.exists():
        dst = result_dir / f"{name}.request-{now().strftime('%H%M%S')}.md"
    move_no_clobber(p.test_md, dst)
    note(name, f"완료: {why}. 요청 파일을 {dst} 로 옮김", p.root)
    set_status(name, "done", doing="완료, result로 이동", root=p.root)
    for src, new in ((p.test_log, "test-log.md"), (p.test_results, "test-results.jsonl"), (p.progress, "progress.md")):
        if src.exists():
            shutil.copyfile(src, result_dir / new)
    (result_dir / "LINKS.md").write_text(
        f"# 연결\n\n- 요청 원본: {dst}\n- 진행 기록: {p.progress} (복사본 progress.md)\n"
        f"- 테스트 기록: {p.test_log} (복사본 test-log.md)\n- 완료 시각: {stamp()}\n",
        encoding="utf-8",
    )
    done = f"완료({stamp()}): 요청 파일은 {dst} 로 옮겼다.\n"
    (p.test_dir / "STATUS.md").write_text(done, encoding="utf-8")
    (p.process_dir / "MOVED.txt").write_text(done, encoding="utf-8")
    return dst


def fail(name: str, reason: str, next_action: str, root: Path | None = None) -> None:
    """필수 검증이 실패·미실행·불가능할 때: test/에 머물고 실패 사유와 다음 조치를 기록한다."""
    p = adopt(name, root)
    if p.test_log.exists():
        note(name, f"실패 상태: {reason}\n다음 조치: {next_action}", p.root)
    set_status(name, "failed", doing="필수 검증 미완료", blocker=reason, next_step=next_action, root=p.root)
    if p.test_dir.exists():
        (p.test_dir / "STATUS.md").write_text(
            f"실패({stamp()}): {reason}\n다음 조치: {next_action}\n(요청 파일은 이 폴더에 남아 있다)\n",
            encoding="utf-8",
        )


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    s = sub.add_parser("claim", help="request/<name>.md 를 process/<name>/ 으로 옮기고 기록·heartbeat 시작")
    s.add_argument("file")
    s = sub.add_parser("adopt", help="이미 옮겨진 요청에 진행 기록·heartbeat 붙이기")
    s.add_argument("name")
    s.add_argument("--result-date")
    s = sub.add_parser("status", help="현재 상태 남기기")
    s.add_argument("name")
    s.add_argument("state", choices=["working", "testing", "done", "failed", "blocked"])
    s.add_argument("--doing", default="")
    s.add_argument("--next", default="")
    s.add_argument("--blocker", default="")
    s = sub.add_parser("heartbeat", help="(내부) 1분 기록 루프")
    s.add_argument("name")
    s = sub.add_parser("begin-test", help="요청 md를 test/<name>/ 으로 옮기기")
    s.add_argument("name")
    s = sub.add_parser("run-test", help="테스트 명령을 실행하고 test-log.md에 기록 (명령은 -- 뒤)")
    s.add_argument("name")
    s.add_argument("--label", required=True)
    s.add_argument("--optional", action="store_true")
    s.add_argument("--cwd")
    s = sub.add_parser("note", help="test-log.md에 메모(버그·원인·수정)")
    s.add_argument("name")
    s.add_argument("text")
    s = sub.add_parser("complete", help="필수 테스트 전부 통과 시에만 result/로 이동")
    s.add_argument("name")
    s.add_argument("--no-test-reason", default="")
    s = sub.add_parser("fail", help="test/에 머물며 실패 사유 기록")
    s.add_argument("name")
    s.add_argument("--reason", required=True)
    s.add_argument("--next", required=True)
    sub.add_parser("recover", help="멈춘 heartbeat 다시 붙이기")
    s = sub.add_parser("where", help="요청의 현재 단계 출력")
    s.add_argument("name")
    argv = list(sys.argv[1:] if argv is None else argv)
    test_cmd: list[str] = []
    if "--" in argv:
        i = argv.index("--")
        argv, test_cmd = argv[:i], argv[i + 1:]
    args = ap.parse_args(argv)

    if args.cmd == "claim":
        p = claim(Path(args.file))
        print(json.dumps({
            "claimed": bool(p),
            "name": p.name if p else None,
            "process_path": str(p.process_md) if p else None,
            "result_dir": str(p.result_dir()) if p else None,
        }, ensure_ascii=False))
        return 0 if p else 1
    if args.cmd == "adopt":
        p = adopt(args.name, result_date=args.result_date)
        set_status(args.name, "working", doing="기록 시작(adopt)")
        print(p.stage())
    elif args.cmd == "status":
        set_status(args.name, args.state, args.doing, args.next, args.blocker)
    elif args.cmd == "heartbeat":
        return heartbeat(args.name)
    elif args.cmd == "begin-test":
        print(begin_test(args.name).test_md)
    elif args.cmd == "run-test":
        if not test_cmd:
            ap.error("run-test: 명령을 -- 뒤에 적는다")
        return run_test(args.name, args.label, test_cmd, required=not args.optional, cwd=args.cwd)
    elif args.cmd == "note":
        note(args.name, args.text)
    elif args.cmd == "complete":
        try:
            print(complete(args.name, args.no_test_reason))
        except ValueError as e:
            print(e, file=sys.stderr)
            return 2
    elif args.cmd == "fail":
        fail(args.name, args.reason, args.next)
    elif args.cmd == "recover":
        print(json.dumps(recover(), ensure_ascii=False))
    elif args.cmd == "where":
        print(Paths(args.name).stage())
    return 0


if __name__ == "__main__":
    sys.exit(main())
