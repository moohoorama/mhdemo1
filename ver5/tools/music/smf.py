"""Minimal standard MIDI file reader and writer: tracks as lists of (delta ticks, event bytes)."""
import struct
def vlq(data, i):
    v = 0
    while True:
        b = data[i]; i += 1; v = (v << 7) | (b & 0x7f)
        if b < 0x80: return v, i
def enc(v):
    out = [v & 0x7f]; v >>= 7
    while v: out.append(0x80 | (v & 0x7f)); v >>= 7
    return bytes(reversed(out))
def read(path):
    d = open(path, 'rb').read()
    fmt, ntr, div = struct.unpack('>HHH', d[8:14]); i = 14; tracks = []
    for _ in range(ntr):
        assert d[i:i+4] == b'MTrk'; n = struct.unpack('>I', d[i+4:i+8])[0]; j = i + 8; end = j + n; ev = []; run = None
        while j < end:
            dt, j = vlq(d, j); s = d[j]
            if s == 0xff:
                t = d[j+1]; ln, k = vlq(d, j+2); ev.append((dt, d[j:k+ln])); j = k + ln
            elif s in (0xf0, 0xf7):
                ln, k = vlq(d, j+1); ev.append((dt, d[j:k+ln])); j = k + ln
            else:
                if s & 0x80: run = s; j += 1
                n2 = 1 if (run & 0xf0) in (0xc0, 0xd0) else 2
                ev.append((dt, bytes([run]) + d[j:j+n2])); j += n2
        tracks.append(ev); i = end
    return fmt, div, tracks
def write(path, fmt, div, tracks):
    out = b'MThd' + struct.pack('>IHHH', 6, fmt, len(tracks), div)
    for ev in tracks:
        body = b''.join(enc(dt) + b for dt, b in ev)
        out += b'MTrk' + struct.pack('>I', len(body)) + body
    open(path, 'wb').write(out)
