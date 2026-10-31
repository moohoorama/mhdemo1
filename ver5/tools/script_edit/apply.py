#!/usr/bin/env python3
"""Carry a person's script edits (plan/script/chNN.edited.json, saved by cmd/_script) into the game and the story doc.

  python3 tools/script_edit/apply.py 10 --dry   # only report
  python3 tools/script_edit/apply.py 10         # apply what is safe, report the rest

Safe: a changed line whose old text occurs exactly once in its source file (storyNN.json for scene and event lines,
the explore file for explore talk) is replaced there, and in docs/fullstory.md when it occurs there exactly once.
Everything else is listed for the agent to carry by hand: changed directions (Stage → the scene beats in
assets/scenes/chNN.json), new and deleted lines, changed speakers, lines not found once, and lines fullstory words
differently. Files are edited by exact string replacement, never rewritten, so other sessions' changes survive.
Run build.py <ch> --absorb afterwards.
"""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
HERO = ROOT / 'assets/scenario/hero'
FULL = ROOT / 'docs/fullstory.md'


def main():
    args = sys.argv[1:]
    if not args:
        sys.exit(__doc__)
    ch, dry = int(args[0]), '--dry' in args
    base = json.loads((ROOT / f'plan/script/ch{ch:02d}.json').read_text())['Entries']
    edited_path = ROOT / f'plan/script/ch{ch:02d}.edited.json'
    if not edited_path.exists():
        sys.exit(f'{edited_path.relative_to(ROOT)} 없음: 저장된 수정이 없다')
    edited = json.loads(edited_path.read_text())['Entries']
    by_key = {e['Key']: e for e in base}
    texts = {}

    def source(e):
        if e.get('Section') == '탐험':
            return HERO / (e['Key'].split(':')[0] + '.json')
        return HERO / f'story{ch:02d}.json'

    def read(p):
        if p not in texts:
            texts[p] = p.read_text()
        return texts[p]

    done, manual = [], []
    for i, e in enumerate(edited):
        k = e['Key']
        b = by_key.get(k)
        where = f"{k} ({e.get('Speaker') or e.get('Who') or e.get('Kind')})"
        if b is None:
            prev = edited[i - 1]['Key'] if i else '(맨 앞)'
            manual.append(f"새 줄 [{prev} 뒤] {e.get('Speaker', '')}: {e.get('Text', '')}" + (f"\n    지문: {e['Stage']}" if e.get('Stage') else ''))
            continue
        if e.get('Deleted'):
            manual.append(f"삭제 {where}: {b.get('Text', '')}")
            continue
        old, new = b.get('Text') or '', e.get('Text') or ''
        if old != new:
            p = source(e)
            js_old, js_new = json.dumps(old, ensure_ascii=False), json.dumps(new, ensure_ascii=False)
            n = read(p).count(js_old)
            if n == 1:
                texts[p] = texts[p].replace(js_old, js_new)
                done.append(f"{where} → {p.relative_to(ROOT)}\n    - {old}\n    + {new}")
                f = read(FULL).count(old) if old else 0
                if f == 1:
                    texts[FULL] = texts[FULL].replace(old, new)
                    done.append(f"{where} → docs/fullstory.md")
                elif e.get('Section') != '탐험':
                    manual.append(f"fullstory에서 {'못 찾음' if f == 0 else f'{f}곳'} {where}: 같은 뜻의 줄을 찾아 고칠 것\n    + {new}")
            else:
                manual.append(f"대사 {where}: {p.relative_to(ROOT)}에서 {n}곳 (1곳이어야 함)\n    - {old}\n    + {new}")
        if (e.get('Speaker') or '') != (b.get('Speaker') or ''):
            manual.append(f"화자 {where}: {b.get('Speaker')} → {e.get('Speaker')}")
        if (e.get('Stage') or '') != (b.get('Stage') or ''):
            manual.append(f"지문 {where}\n    - {b.get('Stage') or '(없음)'}\n    + {e.get('Stage') or '(없음)'}")
    keys = {e['Key'] for e in edited}
    for b in base:
        if b['Key'] not in keys:
            manual.append(f"빠짐 {b['Key']}: {b.get('Text', '')}")

    if not dry:
        for p, t in texts.items():
            if p.read_text() != t:
                p.write_text(t)
    print(('바꿀 것' if dry else '바꾼 것') + f' {len(done)}')
    for d in done:
        print('  ' + d)
    print(f'직접 처리할 것 {len(manual)}')
    for m in manual:
        print('  ' + m)


if __name__ == '__main__':
    main()
