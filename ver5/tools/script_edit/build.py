#!/usr/bin/env python3
"""Build a chapter's editable script (plan/script/chNN.json) from the game data, for the script editor (cmd/_script).

  python3 tools/script_edit/build.py 10            # (re)build plan/script/ch10.json
  python3 tools/script_edit/build.py 10 --absorb   # after apply.py: keep the edited directions, drop chNN.edited.json

Lines, speakers and explore talk always come fresh from storyNN.json and the explore files the story opens. What an
agent writes into the script by hand is kept across rebuilds, by entry key: each entry's Stage (the directions in
prose, naming the Render places as (A), (B) …) and Note, and the Renders block (which scenes to draw, their places
and shots; cmd/_scene_render draws them). Hint is a mechanical summary of the scene beats or of where an explore
person stands, for the agent writing Stage; the editor does not show it.
"""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
HERO = ROOT / 'assets/scenario/hero'
OUT = ROOT / 'plan/script'

FIXED = {'branch', 'set', 'back', 'grant', 'join', 'explore', 'battle'}


def load_names():
    names = {}
    for f in ('characters.json', 'characters_story.json'):
        for k, v in json.loads((ROOT / 'assets/data' / f).read_text()).get('Characters', {}).items():
            names[k] = v.get('Name', k)
    return names


def items():
    names = {}
    for f, key in (('items.json', 'Items'), ('equipment.json', 'Equipment')):
        for k, v in json.loads((ROOT / 'assets/data' / f).read_text()).get(key, {}).items():
            names[k] = v.get('Name', k)
    return names


def cell(p):
    return f"({p[0]:g},{p[1]:g})"


def step_hint(st):
    who = st.get('Actor', '')
    parts = []
    if 'Enter' in st:
        parts.append(f"{cell(st['Enter'])}에 등장")
    for v in st.get('Via', []):
        parts.append(f"{cell(v)} 거쳐")
    if 'Move' in st:
        parts.append(f"{cell(st['Move'])}로 걷기" + (f"(속도 {st['Speed']:g})" if st.get('Speed') else ''))
    if st.get('Exit'):
        parts.append('퇴장')
    for k, label in (('Face', '방향'), ('Look', '잠깐 봄'), ('Play', '동작'), ('Pose', '자세'), ('Emote', '말풍선'),
                     ('Say', '대사 말풍선'), ('Intro', '첫 등장 소개'), ('Sprite', '모습')):
        if st.get(k):
            parts.append(f"{label} {st[k]}")
    for k, label in (('Cam', '카메라→'), ('Time', '시간'), ('Effect', '효과'), ('Chatter', '혼잣말')):
        if st.get(k):
            parts.append(f"{label} {st[k]}")
    if 'CamAt' in st:
        parts.append(f"카메라 {cell(st['CamAt'])}")
    if st.get('Zoom'):
        parts.append(f"줌 {st['Zoom']:g}")
    if st.get('Wait'):
        parts.append(f"{st['Wait']:g}초 대기")
    if st.get('Shake'):
        parts.append('화면 흔들림')
    if 'Prop' in st:
        parts.append(f"소품 {st['Prop']}")
    if 'Patrol' in st:
        parts.append(f"{cell(st['Patrol'])} 오가며 순찰")
    if st.get('After'):
        parts.append(f"+{st['After']:g}초")
    return (who + ': ' if who else '') + ' '.join(parts)


def beat_hint(lines):
    out = []
    for i, groups in enumerate(lines):
        for g, group in enumerate(groups):
            out.append((f"[줄{i + 1}] " if len(lines) > 1 else '') + f"묶음{g + 1}: " + ' / '.join(step_hint(s) for s in group))
    return '\n'.join(out)


def cast_hint(scene):
    out = [f"맵 {scene['Map']}" + (f", {scene['Time']}" if scene.get('Time') else '')]
    for key, a in scene.get('Cast', {}).items():
        extra = [a.get('Face', '')] + [a[k] for k in ('Pose', 'Sprite') if a.get(k)] + (['처음엔 무대 밖'] if a.get('Off') else [])
        out.append(f"{key} {cell(a['At']) if 'At' in a else '(탐험에서 선 자리)'} {' '.join(x for x in extra if x)}")
    for st in scene.get('Ambient', []):
        out.append('주변: ' + step_hint(st))
    return '\n'.join(out)


def fixed_note(n, names, item_names):
    k = n['Kind']
    if k == 'branch':
        cases = [f"{' & '.join(c['If'])} → {c['Next']}" for c in n['Cases']]
        return ' / '.join(cases) + (f" / 아니면 → {n['Else']}" if n.get('Else') else '')
    if k == 'set':
        return '깃발 ' + ', '.join(n['Flags'])
    if k == 'grant':
        what = item_names.get(n.get('Item'), n.get('Item', ''))
        return f"{what} ×{n.get('Count', 1)}" + (f" → {names.get(n['Character'], n['Character'])}" if n.get('Character') else '')
    if k == 'join':
        return f"{names.get(n['Character'], n['Character'])} 합류"
    if k == 'back':
        return '탐험으로 돌아간다'
    if k == 'explore':
        return f"탐험 {n['Explore']}" + (f" (시작 이벤트 {n['Opening']})" if n.get('Opening') else '')
    if k == 'battle':
        return n.get('Text', '')
    return ''


def build(ch):
    names, item_names = load_names(), items()
    story = json.loads((HERO / f'story{ch:02d}.json').read_text())
    scenes_path = ROOT / f'assets/scenes/ch{ch:02d}.json'
    scenes = json.loads(scenes_path.read_text())['Scenes'] if scenes_path.exists() else {}
    by_node = {n: (sid, s) for sid, s in scenes.items() for n in s['Nodes']}

    entries = []

    def node_entries(nodes, section):
        for n in nodes:
            k = n['Kind']
            e = {'Key': n['ID'], 'Kind': k, 'Section': section}
            sid, sc = by_node.get(n['ID'], (None, None))
            if k == 'scene':
                e['Text'] = n['Place']
                if sc is not None:
                    e['Scene'] = sid
                    e['Hint'] = cast_hint(sc)
            elif k in ('say', 'narrate'):
                e['Text'] = n['Text']
                if k == 'say':
                    e['Speaker'] = '·'.join(names.get(s, s) for s in n['Speaker'].split('·'))
                    e['SpeakerID'] = n['Speaker']
            if k in FIXED:
                e['Fixed'] = True
                e['Note'] = fixed_note(n, names, item_names)
            if sc is not None and sc.get('Beats', {}).get(n['ID']):
                hint = beat_hint(sc['Beats'][n['ID']])
                e['Hint'] = (e['Hint'] + '\n' + hint) if e.get('Hint') else hint
            e['Stage'] = ''
            entries.append(e)

    node_entries(story['Nodes'], '장면')
    node_entries(story.get('Events', []), '이벤트')

    explores = [n['Explore'] for n in story['Nodes'] + story.get('Events', []) if n['Kind'] == 'explore']
    for ex in dict.fromkeys(explores):
        data = json.loads((HERO / f'{ex}.json').read_text())
        for area_id, area in data['Areas'].items():
            entries.append({'Key': f'{ex}:{area_id}', 'Kind': 'area', 'Section': '탐험', 'Text': area['Name'], 'Fixed': True,
                            'Note': f"{ex} · 맵 {area['Map']}", 'Stage': ''})
            for pid, p in area.get('People', {}).items():
                who = pid.split('@')[0]
                hint = f"작은 칸 {cell(p['At'])} {p.get('Face', '')}" + (f" 순찰 {p['Move']}" if p.get('Move') else '')
                lines = []
                if p.get('Chatter'):
                    lines.append({'Kind': 'chatter', 'Text': p['Chatter'], 'Note': '혼잣말 말풍선, | 로 여러 개'})
                talk = p.get('Talk', [])
                for i, (sid, text) in enumerate(talk):
                    lines.append({'Kind': 'talk', 'Text': text, 'Speaker': names.get(sid, sid), 'SpeakerID': sid,
                                  'Note': f"말 걸 때 {i + 1}번째" if len(talk) > 1 else '말 걸 때'})
                if p.get('Sortie', {}).get('Ask'):
                    sid = who if who not in names.values() else next(k for k, v in names.items() if v == who)
                    lines.append({'Kind': 'talk', 'Text': p['Sortie']['Ask'], 'Speaker': who, 'SpeakerID': sid, 'Note': '출진 묻기'})
                for ev in p.get('Event', []):
                    lines.append({'Kind': 'event', 'Fixed': True, 'Note': f"말 걸면 이벤트 {ev['Node']}" + (f" ({' & '.join(ev['If'])})" if ev.get('If') else '')})
                if not lines:
                    lines.append({'Kind': 'person', 'Text': ''})
                for i, l in enumerate(lines):
                    e = {'Key': f'{ex}:{area_id}:{pid}:{l["Kind"]}{i}', 'Section': '탐험', 'Who': who, 'Stage': '', **l}
                    if i == 0:
                        e['Hint'] = hint
                    entries.append(e)
            for i, sign in enumerate(area.get('Signs', [])):
                for j, (sid, text) in enumerate(sign.get('Talk', [])):
                    entries.append({'Key': f'{ex}:{area_id}:sign{i}:{j}', 'Kind': 'sign', 'Section': '탐험', 'Who': sign['Name'],
                                    'Text': text, 'Stage': '', 'Hint': f"작은 칸 {cell(sign['At'])}"})
    return entries


def main():
    args = sys.argv[1:]
    if not args:
        sys.exit(__doc__)
    ch, absorb = int(args[0]), '--absorb' in args
    base_path, edited_path = OUT / f'ch{ch:02d}.json', OUT / f'ch{ch:02d}.edited.json'
    old = json.loads(base_path.read_text()) if base_path.exists() else {}
    keep = {e['Key']: e for e in old.get('Entries', [])}
    if absorb and edited_path.exists():
        for e in json.loads(edited_path.read_text())['Entries']:
            if e['Key'] in keep and not e.get('Deleted'):
                keep[e['Key']] = {**keep[e['Key']], 'Stage': e.get('Stage', ''), 'Note': e.get('Note', keep[e['Key']].get('Note'))}
    entries = build(ch)
    for e in entries:
        k = keep.get(e['Key'])
        if k:
            e['Stage'] = k.get('Stage', '')
            if k.get('Note') and not e.get('Fixed'):
                e['Note'] = k['Note']
            elif k.get('Note') and e.get('Fixed') and k.get('NoteByHand'):
                e['Note'], e['NoteByHand'] = k['Note'], True
    story = json.loads((HERO / f'story{ch:02d}.json').read_text())
    title = next((n['Place'] for n in story['Nodes'] if n['Kind'] == 'scene'), '')
    out = {'Chapter': ch, 'Title': old.get('Title', title), 'Renders': old.get('Renders', []), 'Entries': entries}
    OUT.mkdir(parents=True, exist_ok=True)
    base_path.write_text(json.dumps(out, ensure_ascii=False, indent=1) + '\n')
    print(f"{base_path.relative_to(ROOT)}: {len(entries)} entries, {sum(1 for e in entries if e['Stage'])} with directions")
    if absorb and edited_path.exists():
        edited_path.unlink()
        print(f"{edited_path.relative_to(ROOT)} absorbed and removed")


if __name__ == '__main__':
    main()
