#!/usr/bin/env python3
"""Compose six pentatonic MIDI themes and build an offline listening page."""

from __future__ import annotations

import argparse
import base64
from dataclasses import dataclass, field
import io
import json
from pathlib import Path
import struct
import subprocess
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[2]
OUTPUT = ROOT / "assets/music/themes"
PPQ = 480
PITCHES = {"C": 0, "D": 2, "E": 4, "G": 7, "A": 9}


@dataclass
class Part:
    name: str
    program: int
    channel: int
    volume: int = 90
    pan: int = 64
    notes: list[tuple[float, float, int, int]] = field(default_factory=list)

    def add(self, beat: float, duration: float, pitch: int, velocity: int):
        assert beat >= 0 and duration > 0 and 0 < velocity < 128
        assert 0 <= pitch < 128
        if self.channel != 9:
            assert pitch % 12 in PITCHES.values(), (self.name, pitch)
        self.notes.append((beat, duration, pitch, velocity))


# A and B are individually written eight-bar phrases, in quarter-note beats.
THEMES = [
    dict(id="01-village", title="버들 아래 장터", han="村", theme="마을과 정겨움",
         mode="궁조식", tonic=48, bpm=100, color="#8faa84", lead=73,
         desc="익숙한 골목, 솥에서 오르는 김. 피리가 말을 걸면 뜯는 현이 웃으며 답합니다.",
         instruments="피리풍 선율 · 발현 · 낮은 현", form="주제 → 한낮의 활기 → 주제 변주 → 귀가",
         a=["G4:0.5 A4:0.5 C5:1 E5:1 D5:0.5 C5:0.5", "A4:1 G4:0.5 E4:0.5 G4:1 R:1",
            "E4:0.5 G4:0.5 A4:1 C5:1 A4:1", "G4:1 E4:1 D4:0.5 E4:0.5 C4:1",
            "G4:0.5 A4:0.5 C5:1 D5:0.5 E5:0.5 G5:1", "E5:1 D5:0.5 C5:0.5 A4:1 R:1",
            "G4:0.5 E4:0.5 D4:1 E4:0.5 G4:0.5 A4:1", "G4:1 E4:1 C4:1.5 R:0.5"],
         b=["E5:1 G5:0.5 A5:0.5 G5:1 E5:1", "D5:0.5 E5:0.5 G5:1 E5:1 R:1",
            "D5:1 E5:0.5 D5:0.5 C5:1 A4:1", "C5:1 A4:0.5 G4:0.5 E4:1 R:1",
            "A4:0.5 C5:0.5 D5:1 E5:1 G5:1", "E5:0.5 D5:0.5 C5:1 A4:0.5 G4:0.5 E4:1",
            "G4:1 A4:0.5 C5:0.5 D5:1 G4:1", "E4:1 D4:1 C4:1.5 R:0.5"]),
    dict(id="02-court", title="높은 처마의 조회", han="廷", theme="관청과 높은 사람들",
         mode="궁조식", tonic=48, bpm=88, color="#c7ae77", lead=68,
         desc="문이 열리고 발걸음이 멎습니다. 긴 관악과 낮은 종이 관청의 위엄을 세웁니다.",
         instruments="관악 · 낮은 종 · 현의 지속음", form="입청 → 문답 → 위엄의 재현 → 퇴청",
         a=["C4:2 G4:1 E4:1", "D4:1 E4:1 C4:1.5 R:0.5",
            "G4:1.5 A4:0.5 G4:1 E4:1", "D4:2 R:1 G3:1",
            "C4:1 E4:1 G4:2", "A4:1 G4:1 E4:1 D4:1",
            "E4:1.5 D4:0.5 G3:1 D4:1", "C4:3 R:1"],
         b=["E4:1 G4:1 C5:2", "A4:1 G4:1 E4:1 R:1",
            "G4:1 A4:1 C5:1 D5:1", "E5:2 D5:1 C5:1",
            "A4:1.5 G4:0.5 E4:1 G4:1", "D4:1 E4:1 G4:2",
            "E4:1 D4:1 G3:1 D4:1", "C4:3 R:1"]),
    dict(id="03-intrigue", title="발 너머의 밀담", han="謀", theme="음모를 꾸미는 사람들",
         mode="우조식", tonic=45, bpm=96, color="#b69ac8", lead=77,
         desc="발 너머에 누가 있는가. 낮은 발현 사이로 짧은 숨소리가 오가고, 끝내 답은 들리지 않습니다.",
         instruments="낮은 발현 · 숨 섞인 관악 · 드문 종", form="기척 → 은밀한 문답 → 의심 → 남은 그림자",
         a=["R:1 A3:0.5 C4:0.5 E4:1 R:1", "D4:0.5 C4:0.5 A3:1 R:2",
            "R:0.5 E4:0.5 G4:1 E4:0.5 D4:0.5 R:1", "C4:1 A3:1 R:2",
            "A3:0.5 C4:0.5 D4:1 E4:0.5 G4:0.5 R:1", "E4:1 C4:0.5 D4:0.5 A3:1 R:1",
            "R:1 G3:0.5 A3:0.5 C4:0.5 E4:0.5 D4:1", "C4:0.5 A3:1.5 R:2"],
         b=["R:1 E4:0.5 G4:0.5 A4:1 R:1", "G4:0.5 E4:0.5 D4:1 C4:0.5 E4:0.5 R:1",
            "D4:0.5 E4:0.5 G4:0.5 E4:0.5 R:1 C4:1", "A3:1 E4:0.5 D4:0.5 R:2",
            "C4:0.5 D4:0.5 E4:1 A4:0.5 G4:0.5 E4:1", "D4:1 C4:0.5 A3:0.5 R:1 E3:1",
            "R:0.5 A3:0.5 C4:0.5 D4:0.5 E4:1 G3:1", "A3:2 R:2"]),
    dict(id="04-battle", title="진군하는 깃발", han="陣", theme="전투",
         mode="상조식", tonic=38, bpm=132, color="#ca8273", lead=60,
         desc="북 두 번에 대열이 움직입니다. 짧고 굳센 군악 주제 위로 두 번째 깃발이 솟습니다.",
         instruments="군악풍 관악 · 반복하는 현 · 큰북", form="진군 → 맞부딪침 → 재정비 → 총공세",
         a=["D4:0.75 D4:0.25 A4:1 G4:0.5 E4:0.5 D4:1", "E4:0.5 G4:0.5 A4:1 D5:1 A4:1",
            "C5:0.75 A4:0.25 G4:1 E4:0.5 G4:0.5 A4:1", "G4:1 E4:1 D4:1 R:1",
            "D4:0.5 E4:0.5 G4:1 A4:0.5 C5:0.5 D5:1", "C5:0.5 A4:0.5 G4:1 A4:1 E4:1",
            "G4:0.5 E4:0.5 D4:1 A3:0.5 C4:0.5 E4:1", "D4:2 R:0.5 D4:0.5 E4:0.5 G4:0.5"],
         b=["A4:1 D5:1 E5:0.5 D5:0.5 C5:1", "A4:0.5 C5:0.5 D5:1 A4:1 G4:1",
            "E5:0.75 D5:0.25 C5:1 A4:0.5 G4:0.5 A4:1", "D5:1 A4:1 G4:0.5 E4:0.5 D4:1",
            "G4:0.5 A4:0.5 C5:1 D5:1 E5:1", "D5:0.5 C5:0.5 A4:1 G4:0.5 A4:0.5 E4:1",
            "D4:0.5 E4:0.5 G4:0.5 A4:0.5 G4:1 E4:1", "D4:2 A3:1 D4:1"]),
    dict(id="05-urgency", title="봉화가 오르는 밤", han="急", theme="다급한 상황",
         mode="우조식", tonic=45, bpm=160, color="#d9a46b", lead=71,
         desc="전령이 문턱을 넘기도 전에 다음 봉화가 켜집니다. 엇박과 짧은 음들이 쉴 틈을 좁힙니다.",
         instruments="짧은 관악 · 빠른 발현 · 몰아치는 북", form="급보 → 추격 → 길을 찾아서 → 마지막 질주",
         a=["A4:0.5 E5:0.5 A4:0.5 C5:0.5 D5:0.5 E5:0.5 D5:0.5 C5:0.5",
            "A4:0.75 A4:0.25 C5:0.5 D5:0.5 E5:1 R:0.5 E5:0.5",
            "G5:0.5 E5:0.5 D5:0.5 C5:0.5 E5:0.5 D5:0.5 C5:0.5 A4:0.5",
            "G4:0.5 A4:0.5 C5:0.5 E5:0.5 D5:1 A4:1",
            "A4:0.5 C5:0.5 E5:0.5 A5:0.5 G5:0.5 E5:0.5 D5:0.5 C5:0.5",
            "E5:0.75 D5:0.25 C5:0.5 A4:0.5 C5:0.5 D5:0.5 E5:1",
            "D5:0.5 C5:0.5 A4:0.5 G4:0.5 E4:0.5 G4:0.5 A4:0.5 C5:0.5",
            "A4:1 R:0.5 E4:0.5 G4:0.5 A4:0.5 C5:0.5 E5:0.5"],
         b=["A5:0.75 G5:0.25 E5:0.5 D5:0.5 E5:0.5 G5:0.5 E5:1",
            "D5:0.5 E5:0.5 G5:0.5 A5:0.5 G5:0.5 E5:0.5 D5:1",
            "C5:0.5 E5:0.5 A5:0.5 G5:0.5 E5:0.5 D5:0.5 C5:0.5 A4:0.5",
            "E5:1 D5:0.5 C5:0.5 A4:0.5 R:0.5 A4:1",
            "C5:0.5 D5:0.5 E5:0.5 G5:0.5 A5:1 G5:0.5 E5:0.5",
            "D5:0.5 C5:0.5 E5:0.5 D5:0.5 C5:0.5 A4:0.5 G4:0.5 E4:0.5",
            "G4:0.5 A4:0.5 C5:0.5 D5:0.5 E5:0.5 D5:0.5 C5:0.5 A4:0.5",
            "E4:0.5 G4:0.5 A4:1 R:0.5 E4:0.5 G4:0.5 A4:0.5"]),
    dict(id="06-victory", title="돌아오는 깃발", han="凱", theme="성취와 환희",
         mode="궁조식", tonic=48, bpm=116, color="#dbbf70", lead=56,
         desc="성문 밖으로 사람들이 나옵니다. 장터의 선율을 닮은 응답이 이제는 환호 속에 울립니다.",
         instruments="밝은 관악 · 발현 합주 · 종과 북", form="개선 → 환호 → 함께 부르는 주제 → 환희의 종지",
         a=["G4:0.75 C5:0.25 E5:1 G5:1 E5:1", "D5:0.5 E5:0.5 G5:1 C5:1 R:1",
            "A4:0.5 C5:0.5 E5:1 G5:0.5 A5:0.5 G5:1", "E5:1 D5:1 C5:1 R:1",
            "C5:0.5 D5:0.5 E5:1 G5:1 C6:1", "A5:0.5 G5:0.5 E5:1 D5:0.5 E5:0.5 G5:1",
            "E5:1 G5:0.5 E5:0.5 D5:1 G4:1", "C5:3 R:1"],
         b=["E5:1 G5:0.5 A5:0.5 C6:1 G5:1", "A5:1 G5:0.5 E5:0.5 D5:1 R:1",
            "C5:0.5 E5:0.5 G5:1 A5:1 C6:1", "G5:1 E5:1 D5:0.5 E5:0.5 C5:1",
            "G5:0.75 A5:0.25 C6:1 A5:0.5 G5:0.5 E5:1", "D5:0.5 E5:0.5 G5:1 E5:1 D5:1",
            "C5:0.5 D5:0.5 E5:1 G5:1 D5:1", "C5:3 R:1"]),
]


def phrase(part: Part, bars: list[str], start: int, velocity: int):
    for index, bar in enumerate(bars):
        offset = 0.0
        for token in bar.split():
            name, length = token.split(":")
            duration = float(length)
            if name != "R":
                pitch = PITCHES[name[0]] + 12 * (int(name[1:]) + 1)
                # Small deterministic dynamics preserve the written pulse.
                accent = 5 if offset == 0 else (-4 if offset % 1 else 0)
                part.add((start + index) * 4 + offset, duration * 0.88,
                         pitch, min(118, velocity + accent))
            offset += duration
        assert offset == 4, (part.name, bar, offset)


def arrange(theme: dict) -> list[Part]:
    kind = theme["id"][3:]
    lead = Part("Melody", theme["lead"], 0, 96, 69)
    pluck = Part("Plucked strings", 107, 1, 72, 40)
    low = Part("Low strings", 42, 2, 67, 64)
    air = Part("Sustained ensemble", 48, 3, 47, 83)
    bell = Part("Bells", 14, 4, 55, 52)
    counter = Part("Answering flute", 75, 5, 62, 91)
    drum = Part("War drums", 0, 9, 82, 64)
    parts = [lead, pluck, low, air, bell, counter, drum]
    base_vel = {"village": 73, "court": 67, "intrigue": 58,
                "battle": 89, "urgency": 84, "victory": 86}[kind]
    phrase(lead, theme["a"], 0, base_vel)
    phrase(lead, theme["b"], 8, base_vel + 4)
    phrase(lead, theme["a"], 16, base_vel + 3)
    phrase(lead, theme["b"], 24, base_vel + 8)

    # Reinforce each mode with its own bass destinations and closing cadence.
    if theme["mode"] == "궁조식":
        chords = [(48, 55, 64), (48, 57, 64), (45, 52, 60), (43, 50, 57),
                  (48, 55, 64), (45, 52, 60), (43, 50, 57), (48, 55, 64)]
    elif theme["mode"] == "상조식":
        chords = [(38, 45, 52), (38, 45, 55), (43, 50, 57), (38, 45, 52),
                  (38, 45, 55), (36, 43, 50), (43, 50, 57), (38, 45, 52)]
    else:
        chords = [(45, 52, 60), (45, 55, 62), (40, 48, 55), (45, 52, 60),
                  (48, 55, 64), (43, 50, 57), (40, 48, 55), (45, 52, 60)]

    for bar in range(32):
        beat = bar * 4
        chord = chords[bar % 8]
        intense = bar >= 24 or 8 <= bar < 16
        if bar >= 30:
            chord = chords[0]
        if kind == "village":
            positions = (0, 0.75, 1.5, 2, 2.75, 3.5)
            indices = (0, 1, 2, 1, 2, 1)
            for pos, index in zip(positions, indices):
                pluck.add(beat + pos, 0.38, chord[index] + 12, 49 + index * 5)
            low.add(beat, 1.6, chord[0] - 12, 56)
            low.add(beat + 2, 1.5, chord[1] - 12, 44)
            if bar % 2 == 1:
                for pos, pitch in [(2.5, chord[2] + 12), (3, chord[1] + 12)]:
                    counter.add(beat + pos, 0.36, pitch, 46)
            if bar % 4 == 0:
                bell.add(beat, 0.8, 72, 32)
        elif kind == "court":
            low.add(beat, 3.6, chord[0] - 12, 54)
            for pitch in chord[1:]:
                air.add(beat + 0.08, 3.6, pitch, 42)
            for pos, pitch in [(1, chord[1] + 12), (3, chord[2] + 12)]:
                pluck.add(beat + pos, 0.65, pitch, 43)
            if bar % 4 == 0:
                bell.add(beat, 2, 48 if bar % 8 == 0 else 55, 52)
                drum.add(beat, 0.4, 41, 48)
            if bar >= 16 and bar % 2 == 0:
                counter.add(beat + 2, 1.6, chord[2], 42)
        elif kind == "intrigue":
            low.add(beat, 3.5, chord[0] - 12, 43)
            for pos, index in [(0, 0), (1.5, 1), (2.75, 2)]:
                pluck.add(beat + pos, 0.22, chord[index], 47 if pos == 0 else 36)
            if bar % 4 == 2:
                bell.add(beat + 2.5, 0.6, 76, 31)
            if intense:
                air.add(beat, 3.6, chord[1], 30)
                counter.add(beat + 3, 0.6, chord[2], 37)
            if bar % 2 == 1:
                drum.add(beat + 3.5, 0.15, 76, 32)
        elif kind in ("battle", "urgency"):
            step = 0.5 if kind == "battle" else 0.25
            count = int(4 / step)
            for i in range(count):
                pitch = chord[(0, 1, 2, 1)[i % 4]] + 12
                pluck.add(beat + i * step, step * 0.65, pitch,
                          (61 if i % 4 == 0 else 44) + (5 if intense else 0))
            for pos in (0, 1.5, 2, 3):
                low.add(beat + pos, 0.42, chord[0], 70 if pos == 0 else 57)
            for pitch in chord[1:]:
                air.add(beat, 3.5, pitch, 40 if intense else 31)
            hits = [(0, 36, 91), (1, 43, 69), (2, 36, 82), (3, 41, 76)]
            if kind == "urgency":
                hits += [(0.75, 76, 53), (1.5, 43, 60), (2.75, 76, 60), (3.5, 43, 67)]
            if bar % 4 == 3:
                hits += [(3.25, 45, 60), (3.5, 43, 68), (3.75, 41, 77)]
            hits_by_note = {}
            for pos, pitch, vel in hits:
                key = (pos, pitch)
                hits_by_note[key] = max(vel, hits_by_note.get(key, 0))
            for (pos, pitch), vel in hits_by_note.items():
                drum.add(beat + pos, 0.16, pitch, vel)
            if bar % 8 == 0:
                drum.add(beat, 0.6, 49, 58)
            if intense and bar % 2 == 1:
                counter.add(beat + 2, 0.7, chord[2] + 12, 47)
        else:
            for pos, index in zip((0, 0.5, 1, 1.5, 2, 2.5, 3, 3.5), (0, 1, 2, 1, 0, 2, 1, 2)):
                pluck.add(beat + pos, 0.4, chord[index] + 12, 57)
            low.add(beat, 1.5, chord[0] - 12, 66)
            low.add(beat + 2, 1.5, chord[1] - 12, 57)
            for pitch in chord[1:]:
                air.add(beat, 3.5, pitch, 49)
            for pos, pitch, vel in [(0, 36, 78), (1, 45, 54), (2, 36, 68), (3, 43, 60)]:
                drum.add(beat + pos, 0.2, pitch, vel)
            if bar % 4 == 0:
                bell.add(beat, 1.2, 72 if bar % 8 else 79, 59)
                drum.add(beat, 0.7, 49, 52)
            if bar >= 8 and bar % 2 == 1:
                counter.add(beat + 2, 0.7, chord[2] + 12, 56)
                counter.add(beat + 3, 0.6, chord[1] + 12, 51)

    # Clear the closing two bars and give every theme a real modal ending.
    for part in parts:
        part.notes = [note for note in part.notes if note[0] < 120]
    tonic = theme["tonic"]
    if kind == "urgency":
        phrase(lead, ["E5:0.5 D5:0.5 C5:0.5 A4:0.5 G4:0.5 E4:0.5 G4:0.5 A4:0.5",
                      "A4:3 R:1"], 30, 84)
    elif kind == "battle":
        phrase(lead, ["A4:0.5 G4:0.5 E4:1 D4:1 A3:1", "D4:3 R:1"], 30, 94)
    elif kind == "intrigue":
        phrase(lead, ["E4:0.5 D4:0.5 C4:1 R:1 E3:1", "A3:2 R:2"], 30, 54)
    elif kind == "court":
        phrase(lead, ["G4:1 E4:1 D4:1 G3:1", "C4:3 R:1"], 30, 67)
    elif kind == "village":
        phrase(lead, ["G4:0.5 A4:0.5 G4:1 E4:1 D4:1", "C4:2.5 R:1.5"], 30, 69)
    else:
        phrase(lead, ["E5:1 G5:1 D5:1 G4:1", "C5:3 R:1"], 30, 94)
    for beat in (120, 124):
        low.add(beat, 2.7, tonic - (12 if tonic >= 45 else 0), 56)
        for pitch in chords[0][1:]:
            air.add(beat, 2.5, pitch, 39)
        pluck.add(beat, 1.4, tonic + 12, 55)
    if kind in ("battle", "urgency", "victory", "court"):
        drum.add(124, 0.5, 36, 78)
    return parts


def variable_length(value: int) -> bytes:
    result = [value & 127]
    while value >> 7:
        value >>= 7
        result.insert(0, (value & 127) | 128)
    return bytes(result)


def meta(kind: int, data: bytes) -> bytes:
    return bytes((255, kind)) + variable_length(len(data)) + data


def midi_track(events: list[tuple[int, int, bytes]], end: int) -> bytes:
    output = bytearray()
    previous = 0
    for tick, priority, data in sorted(events, key=lambda item: (item[0], item[1])):
        output.extend(variable_length(tick - previous))
        output.extend(data)
        previous = tick
    output.extend(variable_length(max(end, previous) - previous))
    output.extend(meta(47, b""))
    return b"MTrk" + struct.pack(">I", len(output)) + output


def write_midi(theme: dict, parts: list[Part]) -> bytes:
    end = 128 * PPQ
    tempo = round(60_000_000 / theme["bpm"])
    events = [(0, 0, meta(3, theme["title"].encode())),
              (0, 1, meta(81, tempo.to_bytes(3, "big"))),
              (0, 1, meta(88, bytes((4, 2, 24, 8))))]
    for bar, name in [(0, "A"), (8, "B"), (16, "A variation"), (24, "B climax"), (30, "Coda")]:
        events.append((bar * 4 * PPQ, 2, meta(6, name.encode())))
    tracks = [midi_track(events, end)]
    for part in parts:
        ch = part.channel
        events = [(0, 0, meta(3, part.name.encode())),
                  (0, 1, bytes((192 | ch, part.program)))]
        for cc, value in [(7, part.volume), (10, part.pan), (91, 38), (93, 8)]:
            events.append((0, 2, bytes((176 | ch, cc, value))))
        for beat, duration, pitch, velocity in part.notes:
            events.append((round(beat * PPQ), 4, bytes((144 | ch, pitch, velocity))))
            events.append((round((beat + duration) * PPQ), 3, bytes((128 | ch, pitch, 0))))
        tracks.append(midi_track(events, end))
    return b"MThd" + struct.pack(">IHHH", 6, 1, len(tracks), PPQ) + b"".join(tracks)


def run(command: list[str]):
    result = subprocess.run(command, text=True, capture_output=True, check=True)
    if "No preset found" in result.stderr or "error:" in result.stderr.lower():
        raise RuntimeError(result.stderr)
    return result


def render(theme: dict, soundfont: Path, scratch: Path):
    wav = scratch / f"{theme['id']}.wav"
    run(["fluidsynth", "-ni", "-g", "0.6", "-r", "44100", "-T", "wav",
         "-o", "synth.reverb.active=1", "-o", "synth.chorus.active=0",
         "-F", str(wav), str(soundfont), str(OUTPUT / f"{theme['id']}.mid")])
    seconds = 128 * 60 / theme["bpm"]
    # Keep a short natural release, then a quiet tail. Not a gapless loop master.
    length = seconds + 1.5
    filter_graph = (f"atrim=0:{length:.6f},apad=whole_dur={length:.6f},"
                    f"afade=t=out:st={length-1:.6f}:d=1,"
                    "loudnorm=I=-18:TP=-1.5:LRA=9")
    run(["ffmpeg", "-v", "error", "-y", "-i", str(wav), "-af", filter_graph,
         "-ar", "44100", "-ac", "2", "-codec:a", "libmp3lame", "-b:a", "128k",
         "-metadata", f"title={theme['title']}", "-metadata", "artist=삼국지 여섯 테마",
         str(OUTPUT / f"{theme['id']}.mp3")])
    theme["duration"] = length


def build_page():
    bundle = io.BytesIO()
    with zipfile.ZipFile(bundle, "w", zipfile.ZIP_DEFLATED) as archive:
        for theme in THEMES:
            info = zipfile.ZipInfo(f"{theme['id']}.mid", date_time=(2026, 10, 25, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, (OUTPUT / info.filename).read_bytes())
    (OUTPUT / "six-themes-midi.zip").write_bytes(bundle.getvalue())
    data = []
    for theme in THEMES:
        track = {key: value for key, value in theme.items() if key not in ("a", "b", "lead", "tonic")}
        for ext in ("mp3", "mid"):
            track[ext] = base64.b64encode((OUTPUT / f"{theme['id']}.{ext}").read_bytes()).decode()
        data.append(track)
    template = Path(__file__).with_name("player.html").read_text()
    page = template.replace("__TRACK_DATA__", json.dumps(data, ensure_ascii=False))
    page = page.replace("__MIDI_ZIP__", base64.b64encode(bundle.getvalue()).decode())
    target = ROOT / "docs/music-themes.html"
    target.write_text(page)
    print(f"HTML: {target} ({target.stat().st_size / 1024 / 1024:.1f} MiB)")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--soundfont", type=Path, required=True)
    args = parser.parse_args()
    if not args.soundfont.is_file():
        parser.error(f"SoundFont not found: {args.soundfont}")
    OUTPUT.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="sanguo-music-") as scratch:
        for theme in THEMES:
            parts = arrange(theme)
            (OUTPUT / f"{theme['id']}.mid").write_bytes(write_midi(theme, parts))
            render(theme, args.soundfont, Path(scratch))
            print(f"{theme['id']}: {theme['duration']:.1f}s, "
                  f"{sum(len(p.notes) for p in parts)} notes", flush=True)
    build_page()


if __name__ == "__main__":
    main()
