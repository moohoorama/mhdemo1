"""requestflow 자동 테스트. 임시 디렉터리를 루트로 쓰고 heartbeat 간격을 줄여 실제 분리 프로세스로 검증한다.

실행: python3 -m unittest tools/request_mcp/test_requestflow.py -v  (저장소 루트에서)
"""

import json
import os
import re
import signal
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
os.environ["REQUEST_FLOW_HEARTBEAT_SECONDS"] = "1"

import requestflow as rf  # noqa: E402

TS = re.compile(r"\| (\d{4}-\d\d-\d\d \d\d:\d\d:\d\d) KST\(Asia/Seoul, UTC\+09:00\) \| heartbeat \|")


def make_root() -> Path:
    root = Path(tempfile.mkdtemp(prefix="rf-test-"))
    (root / "request").mkdir()
    (root / "process").mkdir()
    return root


def wait_for(cond, timeout=10.0):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        if cond():
            return True
        time.sleep(0.1)
    return False


def kill_heartbeat(p: rf.Paths) -> None:
    try:
        pid = int(p.lock.read_text().strip())
        os.kill(pid, signal.SIGTERM)
    except (FileNotFoundError, ValueError, ProcessLookupError):
        pass


class FlowTest(unittest.TestCase):
    def setUp(self):
        self.root = make_root()
        os.environ["REQUEST_FLOW_ROOT"] = str(self.root)
        self.addCleanup(self.cleanup)

    def cleanup(self):
        for d in (self.root / "process").glob("*"):
            kill_heartbeat(rf.Paths(d.name, self.root))
        time.sleep(0.3)

    def req(self, name, text="# req\n"):
        f = self.root / "request" / f"{name}.md"
        f.write_text(text, encoding="utf-8")
        return f

    def test_claim_moves_and_records_start(self):
        f = self.req("a")
        p = rf.claim(f)
        self.assertFalse(f.exists())
        self.assertEqual(p.process_md.read_text(), "# req\n")
        self.assertIn("process/a/a.md 이동", p.progress.read_text(encoding="utf-8"))
        self.assertEqual(p.stage(), "process")

    def test_two_claimers_one_wins(self):
        f = self.req("race")
        results = []
        threads = [threading.Thread(target=lambda: results.append(rf.claim(f))) for _ in range(6)]
        [t.start() for t in threads]
        [t.join() for t in threads]
        won = [r for r in results if r]
        self.assertEqual(len(won), 1)

    def test_same_name_never_overwrites(self):
        p1 = rf.claim(self.req("dup", "first\n"))
        p2 = rf.claim(self.req("dup", "second\n"))
        self.assertEqual(p1.name, "dup")
        self.assertEqual(p2.name, "dup-2")
        self.assertEqual(p1.process_md.read_text(), "first\n")
        self.assertEqual(p2.process_md.read_text(), "second\n")

    def test_heartbeat_writes_real_timestamps_each_interval(self):
        p = rf.claim(self.req("hb"))
        rf.set_status("hb", "working", doing="x를 하는 중", next_step="y")
        self.assertTrue(wait_for(lambda: p.progress.read_text(encoding="utf-8").count("| heartbeat | 단계=process") >= 3))
        rows = TS.findall(p.progress.read_text(encoding="utf-8"))
        self.assertGreaterEqual(len(rows), 3)
        self.assertIn("수행: x를 하는 중", p.progress.read_text(encoding="utf-8"))

    def test_heartbeat_is_single_per_request_and_independent_per_request(self):
        a = rf.claim(self.req("one"))
        b = rf.claim(self.req("two"))
        self.assertFalse(rf.ensure_heartbeat("one"))
        rf.set_status("one", "working", doing="A")
        rf.set_status("two", "working", doing="B")
        self.assertTrue(wait_for(lambda: "heartbeat | 단계" in a.progress.read_text(encoding="utf-8")))
        self.assertTrue(wait_for(lambda: "heartbeat | 단계" in b.progress.read_text(encoding="utf-8")))
        self.assertNotIn("수행: B", "\n".join(l for l in a.progress.read_text(encoding="utf-8").splitlines() if "heartbeat | 단계" in l))
        starts = [l for l in a.progress.read_text(encoding="utf-8").splitlines() if "heartbeat | 시작" in l]
        self.assertEqual(len(starts), 1)

    def test_heartbeat_continues_through_test_stage_and_stops_on_done(self):
        p = rf.claim(self.req("full"))
        rf.begin_test("full")
        self.assertEqual(p.stage(), "test")
        self.assertFalse(p.process_md.exists())
        self.assertTrue(wait_for(lambda: "단계=test" in p.progress.read_text(encoding="utf-8")))
        self.assertEqual(rf.run_test("full", "ok", [sys.executable, "-c", "print('hi')"]), 0)
        dst = rf.complete("full")
        self.assertTrue(dst.exists())
        self.assertFalse(p.test_md.exists())
        self.assertEqual(p.stage(), "result")
        self.assertTrue(wait_for(lambda: p.lock.exists() and rf._lock_is_free(p)))
        text = p.progress.read_text(encoding="utf-8")
        self.assertIn("완료", text)
        self.assertTrue((dst.parent / "test-log.md").exists())
        self.assertTrue((dst.parent / "LINKS.md").exists())
        self.assertTrue(p.progress.exists())

    def test_complete_refused_when_required_test_failing_then_passes_after_fix(self):
        p = rf.claim(self.req("bug"))
        rf.begin_test("bug")
        flag = self.root / "flag"
        cmd = [sys.executable, "-c", f"import sys,os; sys.exit(0 if os.path.exists({str(flag)!r}) else 1)"]
        self.assertEqual(rf.run_test("bug", "t1", cmd), 1)
        with self.assertRaises(ValueError):
            rf.complete("bug")
        self.assertTrue(p.test_md.exists())
        self.assertEqual(p.stage(), "test")
        self.assertIn("실패", (p.test_dir / "STATUS.md").read_text(encoding="utf-8"))
        flag.write_text("fixed")
        rf.note("bug", "원인: flag 없음. 수정: flag 생성")
        self.assertEqual(rf.run_test("bug", "t1", cmd), 0)
        rf.complete("bug")
        self.assertEqual(p.stage(), "result")
        log = (p.test_dir / "test-log.md").read_text(encoding="utf-8")
        self.assertIn("[t1] FAIL", log)
        self.assertIn("[t1] PASS", log)

    def test_complete_refused_without_any_test(self):
        rf.claim(self.req("none"))
        p = rf.begin_test("none")
        with self.assertRaises(ValueError):
            rf.complete("none")
        self.assertTrue(p.test_md.exists())

    def test_non_code_request_needs_reason(self):
        rf.claim(self.req("doc"))
        p = rf.begin_test("doc")
        dst = rf.complete("doc", no_test_reason="문서만 바꿈, 실행할 코드 없음")
        self.assertTrue(dst.exists())
        self.assertIn("테스트 불필요 사유", (p.test_dir / "test-log.md").read_text(encoding="utf-8"))

    def test_optional_failure_does_not_block_but_required_does(self):
        rf.claim(self.req("opt"))
        rf.begin_test("opt")
        fail_cmd = [sys.executable, "-c", "import sys; sys.exit(3)"]
        ok_cmd = [sys.executable, "-c", "pass"]
        rf.run_test("opt", "must", ok_cmd)
        rf.run_test("opt", "extra", fail_cmd, required=False)
        self.assertTrue(rf.complete("opt").exists())

    def test_complete_keeps_existing_result_file(self):
        rf.claim(self.req("coll"))
        p = rf.begin_test("coll")
        p.result_dir().mkdir(parents=True, exist_ok=True)
        (p.result_dir() / "coll.md").write_text("OLD", encoding="utf-8")
        self.assertEqual(p.stage(), "result")
        rf.run_test("coll", "x", [sys.executable, "-c", "pass"])
        self.assertEqual((p.result_dir() / "coll.md").read_text(), "OLD")

    def test_recover_restarts_dead_heartbeat_only_for_active_new_flow(self):
        p = rf.claim(self.req("rec"))
        legacy = self.root / "process" / "legacy"
        legacy.mkdir()
        (legacy / "legacy.md").write_text("old", encoding="utf-8")
        self.assertTrue(wait_for(lambda: p.lock.exists() and p.lock.read_text().strip()))
        kill_heartbeat(p)
        self.assertTrue(wait_for(lambda: rf._lock_is_free(p)))
        self.assertEqual(rf.recover(), ["rec"])
        self.assertTrue(wait_for(lambda: not rf._lock_is_free(p)))
        self.assertFalse((legacy / "progress.md").exists())
        self.assertIn("recover", p.progress.read_text(encoding="utf-8"))

    def test_failed_state_stops_heartbeat_and_blocks_recover(self):
        p = rf.claim(self.req("fl"))
        rf.begin_test("fl")
        rf.fail("fl", "필수 테스트 미실행", "다음에 실행")
        self.assertTrue(wait_for(lambda: rf._lock_is_free(p)))
        self.assertEqual(rf.recover(), [])
        self.assertTrue(p.test_md.exists())

    def test_cli_run_test_records_start_end_exit(self):
        rf.claim(self.req("cli"))
        rf.begin_test("cli")
        code = rf.main(["run-test", "cli", "--label", "c", "--", sys.executable, "-c", "print(1)"])
        self.assertEqual(code, 0)
        rec = json.loads((self.root / "test" / "cli" / "test-results.jsonl").read_text().splitlines()[0])
        self.assertEqual(rec["verdict"], "PASS")
        self.assertLessEqual(rec["start"], rec["end"])
        self.assertIn("+09:00", rec["start"])


if __name__ == "__main__":
    unittest.main()
