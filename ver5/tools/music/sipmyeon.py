"""Re-voices the user's 십면매복 MIDI (tools/music/sources/sipmyeon-maebok.mid, from musics.zip) for the battle theme and
renders it: the lead on the koto (a guzheng's stand-in), the plucked part as pizzicato strings, the bass on the cello,
the taiko doubled by timpani, each with its own reverb and pan. Loudness is matched to -15.8 LUFS, the old render's.

    python3 tools/music/sipmyeon.py SOUNDFONT.sf2 OUT.mp3

The soundfont used is GeneralUser GS (free for any use): https://github.com/mrbumpy409/GeneralUser-GS"""
import subprocess
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
import smf  # noqa: E402

SRC = Path(__file__).parent / 'sources/sipmyeon-maebok.mid'


def program(events, ch, prog):
    return [(dt, bytes([0xc0 | ch, prog]) if b[0] & 0xf0 == 0xc0 else b) for dt, b in events]


def controls(ch, reverb, chorus, pan, volume):
    return [(0, bytes([0xb0 | ch, c, v])) for c, v in ((91, reverb), (93, chorus), (10, pan), (7, volume))]


def copy_to(events, ch, prog, velocity):
    """The track's notes on another channel and program, softer by `velocity`; its other events become no-ops."""
    out = [(0, bytes([0xc0 | ch, prog]))]
    for dt, b in events:
        if b[0] < 0xf0 and b[0] & 0xf0 in (0x80, 0x90):
            v = b[2] if b[0] & 0xf0 == 0x80 or b[2] == 0 else max(1, min(127, int(b[2] * velocity)))
            out.append((dt, bytes([(b[0] & 0xf0) | ch, b[1], v])))
        else:
            out.append((dt, bytes([0xb0 | ch, 0x0b, 127])))  # expression at full: keeps the timing
    return out


def main():
    soundfont, out = sys.argv[1], sys.argv[2]
    fmt, div, tracks = smf.read(SRC)
    lead, koto, strings, bass, taiko = tracks[1:6]
    tracks[1] = controls(0, 50, 0, 64, 110) + program(lead, 0, 107)  # koto
    tracks[2] = controls(1, 40, 0, 40, 90) + program(koto, 1, 45)  # pizzicato strings
    tracks[3] = controls(2, 70, 20, 90, 85) + program(strings, 2, 49)  # slow strings
    tracks[4] = controls(3, 30, 0, 64, 100) + program(bass, 3, 42)  # cello
    tracks[5] = controls(4, 40, 0, 64, 110) + taiko
    tracks.append(controls(5, 50, 0, 64, 80) + copy_to(taiko, 5, 47, .8))  # timpani
    with tempfile.TemporaryDirectory() as tmp:
        mid, wav = Path(tmp) / 'b.mid', Path(tmp) / 'b.wav'
        smf.write(mid, fmt, div, tracks)
        subprocess.run(['fluidsynth', '-ni', '-q', '-g', '0.6', '-r', '44100', '-F', str(wav), soundfont, str(mid)], check=True)
        subprocess.run(['ffmpeg', '-loglevel', 'error', '-y', '-i', str(wav), '-af',
                        'silenceremove=stop_periods=-1:stop_duration=2:stop_threshold=-60dB,loudnorm=I=-15.8:TP=-1.5:LRA=11',
                        '-ar', '44100', '-b:a', '192k', out], check=True)


if __name__ == '__main__':
    main()
