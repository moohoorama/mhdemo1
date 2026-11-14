# /// script
# requires-python = ">=3.10"
# dependencies = ["sqlite-vec"]
# ///
"""Searchable index of the story text: docs/fullstory.md and every story, explore and stage file.

  uv run tools/storydb/storydb.py index
  uv run tools/storydb/storydb.py search "조조가 유비에게 빚을 지는 장면" [-n 8] [--source story] [--chapter 3] [--speaker 관우]
  uv run tools/storydb/storydb.py grep 형님 [--speaker 관우] [--source fullstory]
  uv run tools/storydb/storydb.py jargon [--source fullstory]   # every line using a word listed in jargon.tsv

Chunks (an S# section, the nodes between two scene nodes, one explore person, one stage event) are embedded
with bge-m3 through a local Ollama; search mixes that with substring hits. grep lists single lines.
Every result carries file:line. Run from ver5/.
"""

import argparse
import hashlib
import json
import math
import re
import sqlite3
import struct
import sys
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path

import sqlite_vec

ROOT = Path(__file__).resolve().parents[2]
DB_PATH = Path(__file__).with_name("storydb.sqlite")
JARGON = Path(__file__).with_name("jargon.tsv")
FULLSTORY = ROOT / "docs/fullstory.md"
SCENARIO = ROOT / "assets/scenario/hero"
MODEL = "bge-m3"
DIM = 1024
OLLAMA = "http://localhost:11434/api/embed"
MAX_CHUNK = 1200
HANGUL = re.compile(r"[가-힣]")
PARTICLE = re.compile(r"(에게서|에게|께서|으로|하고|이랑|까지|부터|처럼|보다|이가|은|는|이|가|을|를|의|에|와|과|도|로|만)$")
COMMON_TERM = 0.12  # a search word found in more chunks than this share says nothing about the topic


@dataclass
class Line:
    lineno: int
    speaker: str
    text: str


@dataclass
class Chunk:
    source: str
    path: str
    chapter: int | None
    title: str
    lines: list[Line] = field(default_factory=list)

    @property
    def start(self) -> int:
        return self.lines[0].lineno

    @property
    def end(self) -> int:
        return self.lines[-1].lineno

    @property
    def body(self) -> str:
        return "\n".join(f"{l.speaker}: {l.text}" if l.speaker else l.text for l in self.lines)

    @property
    def embed_text(self) -> str:
        return f"[{self.title}]\n{self.body}"

    @property
    def hash(self) -> str:
        key = f"{self.path}\0{self.title}\0{self.body}"
        return hashlib.sha1(key.encode()).hexdigest()


def load_names() -> dict[str, str]:
    names = {}
    for f in ("characters.json", "characters_story.json"):
        for cid, c in json.loads((ROOT / "assets/data" / f).read_text())["Characters"].items():
            names[cid] = c.get("Name") or cid
    return names


def split_long(chunk: Chunk) -> list[Chunk]:
    if len(chunk.body) <= MAX_CHUNK:
        return [chunk]
    out, cur, size = [], [], 0
    for line in chunk.lines:
        if cur and size + len(line.text) > MAX_CHUNK:
            out.append(Chunk(chunk.source, chunk.path, chunk.chapter, chunk.title, cur))
            cur, size = [], 0
        cur.append(line)
        size += len(line.text) + 1
    if cur:
        out.append(Chunk(chunk.source, chunk.path, chunk.chapter, chunk.title, cur))
    return out


# fullstory.md: top sections sit between ==== rules, parts between ---- rules, chapters between #### rules
# (1부) or on a bare "N. " line (2부 on), and inside a chapter every "S#n", "[스토리]"-style heading or "근거:" paragraph starts a chunk.
SPEAKER_LINE = re.compile(r"^\s*([가-힣·]{1,8}):\s*(.+)$")
CHAPTER = re.compile(r"^(\d+)\.\s")
LABELS = {"개요", "맵", "근거", "보상", "목표", "상태", "표기", "전투", "결과", "승리", "패배", "부목표"}


def parse_fullstory(names: dict[str, str]) -> list[Chunk]:
    rel = str(FULLSTORY.relative_to(ROOT))
    raw = FULLSTORY.read_text().splitlines()
    chunks: list[Chunk] = []
    section = chapter_title = ""
    chapter: int | None = None
    cur: Chunk | None = None
    last_speaker = ""

    def flush():
        nonlocal cur
        if cur and cur.lines:
            chunks.extend(split_long(cur))
        cur = None

    def start(title: str):
        nonlocal cur
        flush()
        head = " / ".join(t for t in (section, chapter_title, title) if t)
        cur = Chunk("fullstory", rel, chapter, head)

    i = 0
    while i < len(raw):
        s = raw[i]
        rule = s[:1] * 20 if s[:1] in "=#-" else None
        if rule and s.startswith(rule) and i + 2 < len(raw) and raw[i + 2].startswith(rule):
            title = raw[i + 1].strip()
            if s[0] == "=":
                section, chapter_title, chapter = title, "", None
            elif s[0] == "#":
                m = CHAPTER.match(title)
                chapter, chapter_title = (int(m.group(1)) if m else None), title
            else:
                chapter_title, chapter = title, None
            start("")
            i += 3
            continue
        stripped = s.strip()
        m = CHAPTER.match(s)
        if m and section.startswith("3."):
            chapter, chapter_title = int(m.group(1)), stripped
            start("")
        elif stripped.startswith("S#") or (stripped.startswith("[") and "]" in stripped[:20]) or stripped.startswith("근거:"):
            start(stripped.split("/")[0].strip() if stripped.startswith("S#") else stripped[:30])
        elif cur is None:
            start("")
        if stripped:
            m = SPEAKER_LINE.match(s)
            if m and m.group(1) not in LABELS:
                last_speaker = m.group(1)
                cur.lines.append(Line(i + 1, last_speaker, m.group(2).strip()))
            elif s.startswith("        ") and last_speaker and not stripped.startswith("-"):
                cur.lines.append(Line(i + 1, last_speaker, stripped))
            else:
                last_speaker = ""
                cur.lines.append(Line(i + 1, "", stripped))
        i += 1
    flush()
    return chunks


class LineFinder:
    """Maps a string or node ID back to its line in the raw JSON file."""

    def __init__(self, path: Path):
        self.lines = path.read_text().splitlines()
        self.pos = 0

    def find(self, needle: str) -> int:
        enc = json.dumps(needle, ensure_ascii=False)[1:-1]
        for start in (self.pos, 0):
            for n in range(start, len(self.lines)):
                if enc in self.lines[n]:
                    self.pos = n
                    return n + 1
        return self.pos + 1


def chapter_of(path: Path) -> int | None:
    m = re.search(r"(\d+)", path.stem)
    return int(m.group(1)) if m else None


def node_line(node: dict, names: dict[str, str], finder: LineFinder) -> Line | None:
    kind = node.get("Kind")
    if kind == "say" and "Lines" in node:
        return None
    text = node.get("Text") or node.get("Ask") or ""
    if kind == "scene":
        text = f"(장면: {node.get('Place', '')})"
    if not text:
        return None
    speaker = node.get("Speaker") or node.get("Actor") or ""
    if node.get("ID"):
        finder.find(node["ID"])
    return Line(finder.find(text), names.get(speaker, speaker), text)


def parse_story(path: Path, names: dict[str, str]) -> list[Chunk]:
    rel = str(path.relative_to(ROOT))
    data = json.loads(path.read_text())
    finder = LineFinder(path)
    chapter = chapter_of(path)
    chunks: list[Chunk] = []
    for group in ("Nodes", "Events"):
        cur: Chunk | None = None
        for node in data.get(group, []):
            if node.get("Kind") == "scene" or cur is None:
                if cur and cur.lines:
                    chunks.extend(split_long(cur))
                place = node.get("Place", "") if node.get("Kind") == "scene" else ""
                cur = Chunk("story", rel, chapter, f"{chapter}장 {'이벤트' if group == 'Events' else '대본'} · {place or node.get('ID', '')}")
            line = node_line(node, names, finder)
            if line:
                cur.lines.append(line)
        if cur and cur.lines:
            chunks.extend(split_long(cur))
    return chunks


def text_lines(obj, names: dict[str, str], finder: LineFinder) -> list[Line]:
    """Every Korean line under an explore person or a stage event, with its speaker when one is given."""
    out: list[Line] = []
    if isinstance(obj, dict):
        if isinstance(obj.get("Text"), str):
            sp = obj.get("Speaker") or obj.get("Actor") or ""
            out.append(Line(finder.find(obj["Text"]), names.get(sp, sp), obj["Text"]))
        for k, v in obj.items():
            if k in ("Text", "Map", "Sprite", "Faction", "Face", "If", "Time"):
                continue
            if isinstance(v, str) and HANGUL.search(v):
                label = {"Ask": "(출진 질문)", "Chatter": "(혼잣말)", "Name": "(이름)"}.get(k, "")
                out.append(Line(finder.find(v), "", f"{label} {v}".strip()))
            else:
                out += text_lines(v, names, finder)
    elif isinstance(obj, list):
        if len(obj) == 2 and all(isinstance(x, str) for x in obj) and HANGUL.search(obj[1]) and not HANGUL.search(obj[0][:1] or "a"):
            out.append(Line(finder.find(obj[1]), names.get(obj[0], obj[0]), obj[1]))
        elif len(obj) == 2 and all(isinstance(x, str) for x in obj) and HANGUL.search(obj[1]):
            out.append(Line(finder.find(obj[1]), obj[0], obj[1]))
        else:
            for v in obj:
                out += text_lines(v, names, finder)
    return out


def parse_explore(path: Path, names: dict[str, str]) -> list[Chunk]:
    rel = str(path.relative_to(ROOT))
    data = json.loads(path.read_text())
    finder = LineFinder(path)
    chapter = chapter_of(path)
    chunks = []
    for area_id, area in data.get("Areas", {}).items():
        area_name = area.get("Name", area_id)
        for who, person in area.get("People", {}).items():
            lines = text_lines(person, names, finder)
            if lines:
                title = f"{chapter}장 탐험 · {area_name} · {who.split('@')[0]}"
                chunks += split_long(Chunk("explore", rel, chapter, title, lines))
    return chunks


def parse_stage(path: Path, names: dict[str, str]) -> list[Chunk]:
    rel = str(path.relative_to(ROOT))
    data = json.loads(path.read_text())
    finder = LineFinder(path)
    chapter = chapter_of(path)
    name = data.get("Name", path.stem)
    chunks = []
    goals = text_lines({k: data.get(k) for k in ("Victory", "Defeat")}, names, finder)
    if goals:
        chunks.append(Chunk("stage", rel, chapter, f"{name} · 목표", goals))
    for ev in data.get("Events", []):
        lines = text_lines(ev.get("Do", []), names, finder)
        if lines:
            chunks += split_long(Chunk("stage", rel, chapter, f"{name} · {ev.get('ID', '')}", lines))
    return chunks


def collect() -> list[Chunk]:
    names = load_names()
    chunks = parse_fullstory(names)
    for p in sorted(SCENARIO.glob("story*.json")):
        chunks += parse_story(p, names)
    for p in sorted(SCENARIO.glob("explore*.json")):
        chunks += parse_explore(p, names)
    for p in sorted(SCENARIO.glob("stage*.json")):
        chunks += parse_stage(p, names)
    return chunks


def embed(texts: list[str]) -> list[list[float]]:
    req = urllib.request.Request(
        OLLAMA,
        data=json.dumps({"model": MODEL, "input": texts}).encode(),
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=600) as r:
            return json.loads(r.read())["embeddings"]
    except OSError as e:
        sys.exit(f"Ollama embedding failed ({e}). Is `ollama serve` running and `ollama pull {MODEL}` done?")


def pack(v: list[float]) -> bytes:
    return struct.pack(f"{len(v)}f", *v)


def connect() -> sqlite3.Connection:
    db = sqlite3.connect(DB_PATH)
    db.enable_load_extension(True)
    sqlite_vec.load(db)
    db.executescript(f"""
        CREATE TABLE IF NOT EXISTS embeds(id INTEGER PRIMARY KEY, hash TEXT UNIQUE);
        CREATE VIRTUAL TABLE IF NOT EXISTS vecs USING vec0(embedding float[{DIM}] distance_metric=cosine);
        CREATE TABLE IF NOT EXISTS chunks(
            id INTEGER PRIMARY KEY, embed INTEGER, source TEXT, path TEXT, chapter INTEGER,
            title TEXT, start INTEGER, end INTEGER, body TEXT);
        CREATE INDEX IF NOT EXISTS chunks_embed ON chunks(embed);
        CREATE TABLE IF NOT EXISTS lines(chunk INTEGER, lineno INTEGER, speaker TEXT, text TEXT);
        CREATE INDEX IF NOT EXISTS lines_chunk ON lines(chunk);
    """)
    return db


def cmd_index(args):
    """Embeddings are kept per chunk text hash and reused; chunks and lines are rewritten every run,
    so line numbers follow edits above them without embedding anything again."""
    chunks = collect()
    db = connect()
    have = dict(db.execute("SELECT hash, id FROM embeds"))
    want = {c.hash for c in chunks}
    gone = [eid for h, eid in have.items() if h not in want]
    for eid in gone:
        db.execute("DELETE FROM vecs WHERE rowid=?", (eid,))
        db.execute("DELETE FROM embeds WHERE id=?", (eid,))
    new = list({c.hash: c for c in chunks if c.hash not in have}.values())
    for i in range(0, len(new), 32):
        batch = new[i : i + 32]
        for c, v in zip(batch, embed([c.embed_text for c in batch])):
            eid = db.execute("INSERT INTO embeds(hash) VALUES (?)", (c.hash,)).lastrowid
            db.execute("INSERT INTO vecs(rowid, embedding) VALUES (?,?)", (eid, pack(v)))
            have[c.hash] = eid
        db.commit()
        print(f"\r  embedded {min(i + 32, len(new))}/{len(new)}", end="", file=sys.stderr, flush=True)
    if new:
        print(file=sys.stderr)
    db.execute("DELETE FROM lines")
    db.execute("DELETE FROM chunks")
    for c in chunks:
        cid = db.execute(
            "INSERT INTO chunks(embed, source, path, chapter, title, start, end, body) VALUES (?,?,?,?,?,?,?,?)",
            (have[c.hash], c.source, c.path, c.chapter, c.title, c.start, c.end, c.body),
        ).lastrowid
        db.executemany("INSERT INTO lines VALUES (?,?,?,?)", [(cid, l.lineno, l.speaker, l.text) for l in c.lines])
    db.commit()
    print(f"{len(chunks)} chunks: {len(new)} embedded, {len(gone)} dropped, {len(chunks) - len(new)} reused")


def filters(args) -> tuple[str, list]:
    sql, params = [], []
    if args.source:
        sql.append("c.source = ?")
        params.append(args.source)
    if args.chapter is not None:
        sql.append("c.chapter = ?")
        params.append(args.chapter)
    if args.speaker:
        sql.append("EXISTS (SELECT 1 FROM lines l WHERE l.chunk = c.id AND l.speaker = ?)")
        params.append(args.speaker)
    return (" AND ".join(sql) or "1"), params


def strip_particle(word: str) -> str:
    stem = PARTICLE.sub("", word)
    return stem if len(stem) >= 2 else word


def cmd_search(args):
    db = connect()
    where, params = filters(args)
    allowed = {r[0] for r in db.execute(f"SELECT id FROM chunks c WHERE {where}", params)}
    if not allowed:
        sys.exit("no chunks match the filters (run `index` first?)")
    k = min(db.execute("SELECT count(*) FROM embeds").fetchone()[0], max(200, args.n * 20))
    vec_rank, dist = [], {}
    for eid, d in db.execute(
        "SELECT rowid, distance FROM vecs WHERE embedding MATCH ? AND k = ? ORDER BY distance",
        (pack(embed([args.query])[0]), k),
    ):
        for (cid,) in db.execute("SELECT id FROM chunks WHERE embed=?", (eid,)):
            if cid in allowed:
                vec_rank.append(cid)
                dist[cid] = d

    total = db.execute("SELECT count(*) FROM chunks").fetchone()[0]
    kw: dict[int, float] = {}
    for t in {strip_particle(t) for t in args.query.split()}:
        if len(t) < 2:
            continue
        ids = [cid for (cid,) in db.execute("SELECT id FROM chunks WHERE body LIKE ? OR title LIKE ?", (f"%{t}%",) * 2)]
        if not ids or len(ids) > total * COMMON_TERM:
            continue
        idf = math.log(total / len(ids))
        for cid in ids:
            if cid in allowed:
                kw[cid] = kw.get(cid, 0) + idf
    kw_rank = sorted(kw, key=lambda c: -kw[c])

    score: dict[int, float] = {}
    for rank_list in (vec_rank, kw_rank):
        for r, cid in enumerate(rank_list):
            score[cid] = score.get(cid, 0) + 1 / (60 + r)
    top = sorted(score, key=lambda c: -score[c])[: args.n]

    for cid in top:
        src, path, title, start, end = db.execute(
            "SELECT source, path, title, start, end FROM chunks WHERE id=?", (cid,)
        ).fetchone()
        sim = f"{1 - dist[cid]:.2f}" if cid in dist else "  - "
        print(f"\n▶ {path}:{start}-{end}  [{src}] sim {sim} kw {kw.get(cid, 0):.1f}\n  {title}")
        rows = db.execute("SELECT lineno, speaker, text FROM lines WHERE chunk=? ORDER BY rowid", (cid,)).fetchall()
        shown = rows if args.full else rows[: args.lines]
        for lineno, sp, text in shown:
            print(f"  {lineno:5d}  {sp + ': ' if sp else ''}{text}")
        if len(rows) > len(shown):
            print(f"         … {len(rows) - len(shown)} more lines")


def cmd_grep(args):
    db = connect()
    sql = """SELECT c.path, l.lineno, l.speaker, l.text, c.title FROM lines l JOIN chunks c ON c.id = l.chunk
             WHERE l.text LIKE ?"""
    params: list = [f"%{args.term}%"]
    for col, val in (("l.speaker", args.speaker), ("c.source", args.source), ("c.chapter", args.chapter)):
        if val is not None:
            sql += f" AND {col} = ?"
            params.append(val)
    rows = db.execute(sql + " ORDER BY c.path, l.lineno", params).fetchall()
    for path, lineno, sp, text, title in rows:
        print(f"{path}:{lineno}  {sp + ': ' if sp else ''}{text}    ‹{title}›")
    print(f"-- {len(rows)} lines", file=sys.stderr)


def cmd_jargon(args):
    """Lines a light 삼국지 fan would stumble on: every use of a word marked 바꿈 in jargon.tsv.
    Words marked 유지 were judged fine by the user; a later sweep should not raise them again."""
    db = connect()
    sources = [args.source] if args.source else ["story", "explore", "stage"]
    marks = ",".join("?" * len(sources))
    total = 0
    for row in JARGON.read_text().splitlines()[1:]:
        if not row.strip():
            continue
        term, action, alt, note = (row.split("\t") + ["", "", ""])[:4]
        if action != "바꿈":
            continue
        rows = db.execute(
            f"""SELECT c.path, l.lineno, l.speaker, l.text FROM lines l JOIN chunks c ON c.id = l.chunk
                WHERE l.text LIKE ? AND c.source IN ({marks}) ORDER BY c.path, l.lineno""",
            [f"%{term}%", *sources],
        ).fetchall()
        if not rows:
            continue
        total += len(rows)
        print(f"\n■ {term} → {alt}" + (f"  ({note})" if note else ""))
        for path, lineno, sp, text in rows:
            print(f"  {path}:{lineno}  {sp + ': ' if sp else ''}{text}")
    print(f"-- {total} lines", file=sys.stderr)


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("index", help="(re)build the index; only changed chunks are embedded again")
    sub.add_parser("jargon", help="list lines using words from jargon.tsv").add_argument(
        "--source", choices=["fullstory", "story", "explore", "stage"])
    for name in ("search", "grep"):
        p = sub.add_parser(name)
        p.add_argument("query" if name == "search" else "term")
        p.add_argument("--source", choices=["fullstory", "story", "explore", "stage"])
        p.add_argument("--chapter", type=int)
        p.add_argument("--speaker", help="Korean name as shown, e.g. 관우")
        if name == "search":
            p.add_argument("-n", type=int, default=8)
            p.add_argument("--lines", type=int, default=6, help="lines shown per result")
            p.add_argument("--full", action="store_true", help="show every line of each result")
    args = ap.parse_args()
    {"index": cmd_index, "search": cmd_search, "grep": cmd_grep, "jargon": cmd_jargon}[args.cmd](args)


if __name__ == "__main__":
    main()
