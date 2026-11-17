"""Advance learned policies through genuine undefeated campaign frontiers."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import time


def read_json(path: Path):
    try:
        return json.loads(path.read_text())
    except (OSError, ValueError):
        return None


def write_json(path: Path, data) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n")
    temporary.replace(path)


def progress(report: dict) -> int:
    wins = 0
    for stage in report["Stages"]:
        if not stage["Won"] or stage["Tries"] != 1:
            break
        wins += 1
    return wins


def completed(report: dict) -> bool:
    return (report.get("Cleared") is True
            and report.get("ExpectedBattles") == 20
            and len(report["Stages"]) == 20 and progress(report) == 20)


class Job:
    def __init__(self, directory: Path, command: list[str]):
        directory.mkdir(parents=True, exist_ok=True)
        self.directory = directory
        self.model = directory / "model.json"
        self.log = directory / "train.log"
        write_json(directory / "command.json", command)
        with self.log.open("w") as output:
            self.process = subprocess.Popen(
                command + ["-model", str(self.model)], stdout=output,
                stderr=subprocess.STDOUT, start_new_session=True)

    def stop(self) -> None:
        if self.process.poll() is None:
            os.killpg(self.process.pid, signal.SIGTERM)
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(self.process.pid, signal.SIGKILL)
                self.process.wait()


def counts(base: Path) -> dict:
    fresh = battles = teachers = 0
    for path in base.rglob("*.log"):
        text = path.read_text(errors="replace")
        fresh += len(re.findall(r"^collect \d+ ", text, re.M))
        fresh += 4 * len(re.findall(r"^full campaign dev after ", text, re.M))
        teachers += len(re.findall(r"^teacher \d+/\d+ victory=", text, re.M))
        rows = re.findall(r"^frontier (\d+)/\d+ victory", text, re.M)
        if rows:
            battles += int(rows[-1])
    fresh += len(list((base / "verification").glob("*.json")))
    return {"fresh_campaign_evaluations": fresh,
            "frontier_training_battles": battles, "teacher_battles": teachers}


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--trainer", required=True)
    parser.add_argument("--autoplay", required=True)
    parser.add_argument("--initial", type=Path, required=True)
    parser.add_argument("--stage", default="B12")
    parser.add_argument("--record", type=int, default=11)
    parser.add_argument("--seed", type=int, default=310004)
    parser.add_argument("--base", type=Path, default=Path("reports/autoplay/goal20"))
    parser.add_argument("--content", default="assets/data")
    args = parser.parse_args()
    base = args.base
    base.mkdir(parents=True, exist_ok=True)
    empty = base / "no-teachers.json"
    empty.write_text("[]\n")
    current, stage, record, seed = args.initial, args.stage, args.record, args.seed
    cycle = 0
    attempted: set[str] = set()
    state = {"status": "running", "highest_unbeaten": record,
             "stage": stage, "model": str(current), "seed": seed}
    jobs: list[Job] = []
    try:
        while True:
            cycle += 1
            run = base / f"{stage.lower()}-cycle{cycle:03}"
            common = [args.trainer, "-content", args.content, "-initial", str(current),
                      "-stage", stage, "-stage-preparation", "-goal-inputs",
                      "-target-streak", "20", "-par", "2"]
            teacher = Job(run / "teacher", common + [
                "-collect", "1", "-seed", str(seed), "-teachers", "12",
                "-teacher-risk", ".4", "-teacher-xp", "0", "-teacher-threat", "officers",
                "-teacher-explore", ".1", "-teacher-lord-last", "-warm", "120",
                "-episodes", "128", "-self-imitate", "8", "-progress-reward", ".25"])
            jobs = [teacher]
            pool = Path(str(teacher.model) + ".frontiers.json")
            while not pool.exists():
                if teacher.process.poll() is not None:
                    raise RuntimeError(f"frontier collection failed: {teacher.log}")
                time.sleep(2)
            for index, (temperature, rate) in enumerate([(".25", ".0003"), (".45", ".001")]):
                jobs.append(Job(run / f"neural{index}", common + [
                    "-pool", str(pool), "-teacher-data", str(empty), "-warm", "0",
                    "-episodes", "256", "-temperature", temperature, "-rate", rate,
                    "-epochs", "4", "-self-imitate", "8", "-progress-reward", ".25",
                    "-seed", str(600001 + cycle * 10000 + index * 1000)]))
            distilled: set[str] = set()
            advanced = False
            while True:
                for job in list(jobs):
                    score = read_json(Path(str(job.model) + ".best.eval.json"))
                    if not score or score["Max"] <= record:
                        continue
                    source = Path(str(job.model) + ".best.json")
                    raw = source.read_bytes()
                    try:
                        json.loads(raw)
                    except ValueError:
                        continue
                    digest = hashlib.sha256(raw).hexdigest()
                    if digest in attempted:
                        continue
                    attempted.add(digest)
                    candidate = base / "candidates" / f"{digest}.json"
                    candidate.parent.mkdir(exist_ok=True)
                    candidate.write_bytes(raw)
                    report_path = base / "verification" / f"{digest}.json"
                    report_path.parent.mkdir(exist_ok=True)
                    test_seed = max(score["Reports"], key=progress)["Seed"]
                    command = [args.autoplay, "run", "-content", args.content,
                               "-player", "neural", "-battle-model", str(candidate),
                               "-tries", "1", "-seeds", str(test_seed), "-par", "1",
                               "-out", str(report_path)]
                    with report_path.with_suffix(".log").open("w") as output:
                        subprocess.run(command, stdout=output, stderr=subprocess.STDOUT, check=True)
                    report = read_json(report_path)[0]
                    if progress(report) <= record:
                        continue
                    current, record, seed = candidate, progress(report), test_seed
                    stage = report.get("FirstDefeat", "")
                    state.update(highest_unbeaten=record, model=str(current), seed=seed,
                                 stage=stage, verification=str(report_path), counts=counts(base))
                    if completed(report):
                        state["status"] = "complete"
                    write_json(base / "supervisor.json", state)
                    print(json.dumps(state, ensure_ascii=False), flush=True)
                    advanced = True
                    break
                if advanced:
                    break
                for job in list(jobs):
                    if len([j for j in jobs if j.process.poll() is None]) >= 4:
                        break
                    for success in sorted([*job.directory.glob("model.json.success.*.json"),
                                           *job.directory.glob("model.json.success.*.json.gz")]):
                        if str(success) in distilled:
                            continue
                        snapshot = Path(str(success).removesuffix(".gz").replace(".success.", "."))
                        if not snapshot.exists():
                            continue
                        distilled.add(str(success))
                        command = common.copy()
                        command[command.index("-initial") + 1] = str(snapshot)
                        jobs.append(Job(run / f"imitate{len(distilled):03}", command + [
                            "-pool", str(pool), "-teacher-data", str(success),
                            "-episodes", "0", "-warm", "400", "-seed", str(seed)]))
                        break
                state.update(counts=counts(base), cycle=cycle, stage=stage)
                write_json(base / "supervisor.json", state)
                if all(job.process.poll() is not None for job in jobs):
                    latest = jobs[2].model
                    if read_json(latest):
                        current = latest
                    break
                time.sleep(2)
            for job in jobs:
                job.stop()
            if state["status"] == "complete":
                return
            if advanced and not stage:
                raise RuntimeError("campaign stopped before completion without a next defeated stage")
    finally:
        for job in jobs:
            job.stop()


if __name__ == "__main__":
    main()
