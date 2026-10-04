#!/usr/bin/env python3
"""Generate 영걸전/조조전-style General MIDI BGM: main theme, battle, event.

Pure standard library. Usage: python3 compose.py  (writes midi/*.mid)
"""
import random
import struct
from pathlib import Path

PPQ = 480
OUT = Path(__file__).parent / "midi"

NOTE_BASE = {"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}
CHORD_INTERVALS = {"": (0, 4, 7), "m": (0, 3, 7)}

# GM programs (0-indexed)
FLUTE, VIOLIN, STRINGS, SLOW_STRINGS, PIZZ = 73, 40, 48, 49, 45
CONTRABASS, TIMPANI, BRASS, TRUMPET, KOTO, TAIKO = 43, 47, 61, 56, 107, 116
DRUM_CH = 9
KICK, SNARE, CRASH, CHINA, TOM_LOW = 35, 38, 49, 52, 41


def pitch(name):
    """'D5', 'F#4', 'Bb3' -> MIDI number (C4 = 60)."""
    pc = NOTE_BASE[name[0]]
    rest = name[1:]
    while rest and rest[0] in "#b":
        pc += 1 if rest[0] == "#" else -1
        rest = rest[1:]
    return 12 * (int(rest) + 1) + pc


def chord(name, octave=3):
    """'Dm' -> root midi note in given octave and triad intervals."""
    root = name[0] + (name[1] if len(name) > 1 and name[1] in "#b" else "")
    quality = name[len(root):]
    return pitch(f"{root}{octave}"), CHORD_INTERVALS[quality]


class Track:
    def __init__(self, name, channel, program=None, volume=100, pan=64, reverb=70):
        self.name, self.channel = name, channel
        self.events = []  # (tick, order, bytes); order puts note-offs first
        if program is not None:
            self._ev(0, 0, bytes([0xC0 | channel, program]))
        for cc, val in ((7, volume), (10, pan), (91, reverb), (93, 20)):
            self._ev(0, 0, bytes([0xB0 | channel, cc, val]))

    def _ev(self, tick, order, data):
        self.events.append((int(round(tick)), order, data))

    def note(self, beat, dur, note, vel=90, legato=0.95):
        on = beat * PPQ
        off = on + max(dur * legato, 0.05) * PPQ
        vel = max(1, min(127, vel))
        self._ev(on, 1, bytes([0x90 | self.channel, note, vel]))
        self._ev(off, 0, bytes([0x80 | self.channel, note, 0]))

    def melody(self, beat, text, vel=92, transpose=0, legato=0.95, accent=8):
        """'D5:1.5 C5:0.5 R:1 ...' starting at beat; '|' is ignored. Returns end beat."""
        for tok in text.replace("|", " ").split():
            name, dur = tok.split(":")
            dur = float(dur)
            if name != "R":
                v = vel + (accent if beat % 1 == 0 else 0) + RNG.randint(-4, 4)
                self.note(beat, dur, pitch(name) + transpose, v, legato)
            beat += dur
        return beat

    def chunk(self):
        data = bytearray(meta(0x03, self.name.encode()))
        last = 0
        for tick, _, ev in sorted(self.events, key=lambda e: (e[0], e[1])):
            data += varlen(tick - last) + ev
            last = tick
        data += b"\x00\xff\x2f\x00"
        return b"MTrk" + struct.pack(">I", len(data)) + bytes(data)


def varlen(n):
    out = [n & 0x7F]
    n >>= 7
    while n:
        out.append((n & 0x7F) | 0x80)
        n >>= 7
    return bytes(reversed(out))


def meta(kind, payload):
    return b"\x00\xff" + bytes([kind]) + varlen(len(payload)) + payload


def write_midi(path, bpm, tracks, title):
    conductor = meta(0x03, title.encode())
    conductor += meta(0x51, struct.pack(">I", round(60_000_000 / bpm))[1:])
    conductor += meta(0x58, bytes([4, 2, 24, 8]))
    conductor += b"\x00\xff\x2f\x00"
    head = b"MThd" + struct.pack(">IHHH", 6, 1, len(tracks) + 1, PPQ)
    body = b"MTrk" + struct.pack(">I", len(conductor)) + conductor
    path.write_bytes(head + body + b"".join(t.chunk() for t in tracks))
    print(f"wrote {path} ({len(tracks)} tracks, {bpm} bpm)")


# ---------- accompaniment helpers (one bar = 4 beats) ----------

def pad(track, beat, chords, vel=60, octave=4):
    for i, c in enumerate(chords):
        root, iv = chord(c, octave - 1)
        for n in iv:
            track.note(beat + 4 * i, 4, root + n + 12 * (n < 5), vel + RNG.randint(-3, 3), 1.0)
    return beat + 4 * len(chords)


def bass_roots(track, beat, chords, vel=80, pattern=((0, 2, 0), (2, 2, 7))):
    for i, c in enumerate(chords):
        root, _ = chord(c, 2)
        for off, dur, iv in pattern:
            track.note(beat + 4 * i + off, dur, root + iv, vel + RNG.randint(-4, 4))
    return beat + 4 * len(chords)


def arpeggio(track, beat, chords, vel=70, steps=(0, 7, 12, 16, 19, 16, 12, 7), octave=3, step_len=0.5):
    for i, c in enumerate(chords):
        root, iv = chord(c, octave)
        third = iv[1]
        for j, s in enumerate(steps):
            s = third if s == 4 else third + 12 if s == 16 else s
            track.note(beat + 4 * i + j * step_len, step_len * 1.6, root + s,
                       vel + (8 if j % 4 == 0 else 0) + RNG.randint(-5, 5), 1.0)
    return beat + 4 * len(chords)


def hits(track, beat, bars, pattern, note=None):
    """pattern: [(beat_offset, velocity[, note])] repeated each bar."""
    for b in range(bars):
        for p in pattern:
            n = p[2] if len(p) > 2 else note
            track.note(beat + 4 * b + p[0], 0.5, n, p[1] + RNG.randint(-5, 5), 0.9)
    return beat + 4 * bars


def roll(track, beat, beats, note, start_vel=40, end_vel=110, step=0.125):
    n = int(beats / step)
    for i in range(n):
        track.note(beat + i * step, step, note, start_vel + (end_vel - start_vel) * i // n, 1.0)


# ---------- 1. 메인 테마 「천하」 (D minor pentatonic, 88 bpm) ----------

def main_theme():
    lead = Track("Dizi (Flute)", 0, FLUTE, 105, 70)
    strings = Track("Strings", 1, STRINGS, 92, 50)
    brass = Track("Brass", 2, BRASS, 98, 78)
    bass = Track("Contrabass", 3, CONTRABASS, 100, 54)
    timp = Track("Timpani", 4, TIMPANI, 100, 64)
    koto = Track("Guzheng (Koto)", 5, KOTO, 84, 40)
    taiko = Track("Taiko", 6, TAIKO, 96, 60)
    drums = Track("Cymbals", DRUM_CH, None, 85, 64)

    intro_ch = ["Dm", "Bb", "C", "Dm"]
    a_ch = ["Dm", "F", "Gm", "Dm", "Bb", "C", "Dm", "Am"]
    b_ch = ["Bb", "C", "Dm", "F", "Gm", "C", "Dm", "Dm"]
    a_mel = ("D5:1.5 C5:0.5 A4:1 G4:1 | A4:1.5 C5:0.5 F5:2 | G5:1 F5:0.5 D5:0.5 C5:1 D5:1 | A4:3 R:1 |"
             "D5:1.5 F5:0.5 G5:1 F5:1 | G5:1.5 F5:0.5 D5:1 C5:1 | D5:1 C5:0.5 A4:0.5 G4:1 F4:1 | A4:4")
    b_mel = ("D5:2 C5:1 Bb4:1 | C5:1.5 D5:0.5 E5:2 | F5:1.5 E5:0.5 D5:1 A4:1 | C5:3 A4:1 |"
             "Bb4:1.5 C5:0.5 D5:1 G4:1 | C5:1.5 Bb4:0.5 A4:1 G4:1 | A4:1 D5:1 F5:1 E5:1 | D5:4")

    # Intro: brass fanfare over timpani
    t = 0
    brass.melody(t, "D4:1 A4:0.5 A4:0.5 D5:2 | C5:1 Bb4:0.5 A4:0.5 F4:2 | G4:1 A4:0.5 C5:0.5 D5:1 C5:1 | D5:4", 100)
    pad(strings, t, intro_ch, 70)
    bass_roots(bass, t, intro_ch, 90, ((0, 4, 0),))
    hits(timp, t, 3, [(0, 105), (2.5, 80), (3, 90)], pitch("D2"))
    roll(timp, t + 12, 4, pitch("A1"))
    drums.note(t, 2, CHINA, 100)
    drums.note(t + 12, 4, CRASH, 90)
    t += 16

    for rep in range(2):
        # A: dizi melody with guzheng arpeggio
        lead.melody(t, a_mel, 92)
        if rep:
            strings.melody(t, a_mel, 66, transpose=-12, legato=1.0)
        else:
            pad(strings, t, a_ch, 56)
        arpeggio(koto, t, a_ch, 64)
        bass_roots(bass, t, a_ch, 78)
        hits(timp, t + 28, 1, [(0, 80), (2, 70), (3, 90)], pitch("A1"))
        drums.note(t, 2, CRASH, 70)
        t += 32

        # B: brass tutti, strings an octave above, taiko pulse
        brass.melody(t, b_mel, 100, transpose=-12)
        strings.melody(t, b_mel, 88, legato=1.0)
        lead.melody(t + 28, "A5:1 C6:1 D6:2", 80)
        bass_roots(bass, t, b_ch, 92, ((0, 1.5, 0), (1.5, 0.5, 0), (2, 2, 7)))
        hits(taiko, t, 8, [(0, 110), (1.5, 70), (2, 95), (3, 80), (3.5, 70)], pitch("C3"))
        for i, c in enumerate(b_ch):
            root, _ = chord(c, 2)
            timp.note(t + 4 * i, 1, root, 100)
        drums.note(t, 2, CHINA, 105)
        drums.note(t + 16, 2, CRASH, 90)
        t += 32

    # Ending
    brass.melody(t, "D4:1 A4:1 D5:6", 108, legato=1.0)
    pad(strings, t, ["Dm", "Dm"], 85)
    bass.note(t, 8, pitch("D2"), 100, 1.0)
    roll(timp, t, 6, pitch("D2"), 60, 115)
    drums.note(t, 4, CHINA, 115)
    write_midi(OUT / "01_main_theme.mid", 88,
               [lead, strings, brass, bass, timp, koto, taiko, drums], "Main Theme - Tenka")


# ---------- 2. 전투 「출진」 (A minor pentatonic, 152 bpm) ----------

def battle():
    lead = Track("Erhu (Violin)", 0, VIOLIN, 108, 70)
    dizi = Track("Dizi (Flute)", 1, FLUTE, 90, 84)
    ost = Track("Strings Ostinato", 2, STRINGS, 96, 44)
    brass = Track("Brass Riff", 3, BRASS, 100, 82)
    pizz = Track("Pizzicato Bass", 4, PIZZ, 104, 56)
    timp = Track("Timpani", 5, TIMPANI, 104, 64)
    taiko = Track("Taiko", 6, TAIKO, 110, 60)
    drums = Track("Drums", DRUM_CH, None, 95, 64)

    a_ch = ["Am", "Am", "F", "G", "Am", "Am", "F", "E"]
    b_ch = ["C", "G", "Am", "Em", "F", "G", "Am", "E"]
    a_mel = ("A4:0.5 A4:0.5 C5:0.5 D5:0.5 E5:1 E5:1 | G5:0.5 E5:0.5 D5:0.5 C5:0.5 D5:2 |"
             "C5:0.5 C5:0.5 D5:0.5 E5:0.5 G5:1 A5:1 | G5:1.5 E5:0.5 D5:2 |"
             "E5:0.5 E5:0.5 G5:0.5 A5:0.5 C6:1 A5:1 | G5:0.5 A5:0.5 G5:0.5 E5:0.5 D5:1 C5:1 |"
             "D5:1 E5:0.5 D5:0.5 C5:1 A4:1 | E5:2 B4:2")
    b_mel = ("E5:3 G5:1 | D5:3 B4:1 | C5:2 E5:1 A5:1 | G5:3 E5:1 |"
             "A5:2 G5:1 F5:1 | G5:2 D5:1 B4:1 | C5:1 D5:1 E5:1 A5:1 | E5:1 G#5:1 B5:2")
    taiko_pat = [(0, 115), (0.75, 70), (1.5, 90), (2, 110), (3, 95), (3.5, 85)]
    drum_pat = [(0, 100, KICK), (1, 80, SNARE), (2, 95, KICK), (2.5, 70, KICK), (3, 85, SNARE),
                (3.75, 60, SNARE)]

    def ostinato(t, chords, vel=78):
        for i, c in enumerate(chords):
            root, iv = chord(c, 3)
            for j, s in enumerate((0, 0, 12, 0, iv[2], 0, 12, iv[1] + 12)):
                ost.note(t + 4 * i + j * 0.5, 0.5, root + s, vel + (14 if j % 2 == 0 else 0), 0.7)

    def riff(t, chords, vel=95):
        for i, c in enumerate(chords):
            root, iv = chord(c, 3)
            for off, dur in ((0, 0.75), (1.5, 0.5), (2.5, 0.5), (3, 0.75)):
                for n in iv:
                    brass.note(t + 4 * i + off, dur, root + n + 12, vel + RNG.randint(-4, 4), 0.8)

    def pizz_bass(t, chords):
        for i, c in enumerate(chords):
            root, iv = chord(c, 2)
            for j, s in enumerate((0, 0, 12, 0, iv[2], 0, 12, iv[2])):
                pizz.note(t + 4 * i + j * 0.5, 0.5, root + s, 96 + (10 if j % 4 == 0 else 0))

    # Intro: ostinato + taiko build, brass call
    t = 0
    ostinato(t, ["Am"] * 4, 70)
    pizz_bass(t, ["Am"] * 4)
    hits(taiko, t, 4, taiko_pat, pitch("C3"))
    brass.melody(t + 8, "A4:0.75 A4:0.25 C5:0.5 D5:0.5 E5:2 | G5:0.75 E5:0.25 D5:0.5 C5:0.5 B4:2", 104,
                 transpose=-12)
    roll(drums, t + 14, 2, SNARE, 50, 110)
    t += 16

    for rep in range(2):
        drums.note(t, 2, CHINA, 115)
        lead.melody(t, a_mel, 96, legato=0.85)
        dizi.melody(t, a_mel, 82, transpose=12, legato=0.8)
        ostinato(t, a_ch)
        pizz_bass(t, a_ch)
        if rep:
            riff(t, a_ch, 90)
        hits(taiko, t, 8, taiko_pat, pitch("C3"))
        hits(drums, t, 8, drum_pat)
        for i, c in enumerate(a_ch):
            timp.note(t + 4 * i, 1, chord(c, 2)[0], 100)
        t += 32

        drums.note(t, 2, CRASH, 110)
        lead.melody(t, b_mel, 100, legato=1.0)
        brass.melody(t, b_mel, 94, transpose=-12, legato=1.0)
        riff(t, b_ch, 84)
        ostinato(t, b_ch, 84)
        pizz_bass(t, b_ch)
        hits(taiko, t, 8, taiko_pat, pitch("C3"))
        hits(drums, t, 8, drum_pat)
        roll(timp, t + 28, 4, pitch("E2"), 60, 120)
        t += 32

    write_midi(OUT / "02_battle.mid", 152,
               [lead, dizi, ost, brass, pizz, timp, taiko, drums], "Battle - Shutsujin")


# ---------- 3. 이벤트 「도원」 (G major pentatonic, 66 bpm) ----------

def event():
    flute = Track("Dizi (Flute)", 0, FLUTE, 100, 70)
    erhu = Track("Erhu (Violin)", 1, VIOLIN, 96, 58)
    koto = Track("Guzheng (Koto)", 2, KOTO, 92, 44)
    strings = Track("Slow Strings", 3, SLOW_STRINGS, 80, 64)
    bass = Track("Contrabass", 4, CONTRABASS, 84, 60)
    drums = Track("Gong", DRUM_CH, None, 60, 64)

    a_ch = ["G", "Em", "C", "D", "G", "Em", "Am", "D"]
    b_ch = ["Em", "C", "G", "D", "Em", "C", "D", "G"]
    a_mel = ("D5:1.5 E5:0.5 D5:1 B4:1 | A4:1 B4:1 G4:2 | E4:1 G4:1 A4:1 B4:1 | A4:3 R:1 |"
             "B4:1.5 D5:0.5 E5:1 G5:1 | E5:1.5 D5:0.5 B4:2 | A4:1 B4:0.5 A4:0.5 G4:1 E4:1 | A4:4")
    b_mel = ("G5:2 E5:1 D5:1 | E5:3 D5:0.5 B4:0.5 | D5:2 B4:1 A4:1 | A4:3 R:1 |"
             "B4:1 D5:1 E5:1 G5:1 | A5:2 G5:1 E5:1 | D5:1.5 E5:0.5 D5:1 A4:1 | G4:4")

    t = 0
    drums.note(t, 4, CHINA, 55)
    arpeggio(koto, t, ["G", "G"], 66)
    t += 8

    for rep in range(2):
        flute.melody(t, a_mel, 84, legato=1.0, accent=4)
        arpeggio(koto, t, a_ch, 60)
        pad(strings, t, a_ch, 52)
        bass_roots(bass, t, a_ch, 66, ((0, 4, 0),))
        t += 32

        erhu.melody(t, b_mel, 88, transpose=-12 if rep == 0 else 0, legato=1.0, accent=4)
        if rep:
            flute.melody(t, b_mel, 66, transpose=-12, legato=1.0, accent=0)
        arpeggio(koto, t, b_ch, 56, steps=(0, 7, 12, 16, 12, 7, 0, 7))
        pad(strings, t, b_ch, 58)
        bass_roots(bass, t, b_ch, 66, ((0, 4, 0),))
        t += 32

    koto.melody(t, "G3:0.5 D4:0.5 G4:0.5 B4:0.5 D5:0.5 G5:2.5", 70, legato=1.6)
    strings.note(t, 6, pitch("G3"), 50, 1.0)
    strings.note(t, 6, pitch("D4"), 50, 1.0)
    bass.note(t, 6, pitch("G2"), 60, 1.0)
    write_midi(OUT / "03_event.mid", 66, [flute, erhu, koto, strings, bass, drums], "Event - Touen")


if __name__ == "__main__":
    OUT.mkdir(exist_ok=True)
    for song in (main_theme, battle, event):
        RNG = random.Random(song.__name__)
        song()
