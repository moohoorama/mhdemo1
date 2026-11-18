#!/usr/bin/env python3
"""Render the hero scenario as readable script documents.

Reads the same JSON the game loads (scenario.json, storyNN, scenes/chNN, exploreNN,
stageNN, maps/mapNN, data/*) and writes one markdown per chapter plus a combined
file. Nothing is edited; rerun after any data change.

Usage: build.py [--out output] [--scenario assets/scenario/hero]
"""
import argparse
import json
import os
import sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))

POLICY = {"hold": "제자리", "react": "반응", "assault": "돌격", "march": "행군", "flee": "퇴각", "run": "도주"}
RANK = {"rabble": "오합지졸", "recruit": "신병", "troop": "병졸", "officer": "장수", "commander": "대장", "boss": "거물"}
STAT = {"attack": "공격", "defense": "방어", "mind": "정신", "morale": "사기", "agility": "순발",
        "speed": "이동", "range": "사거리", "confusion": "혼란", "seal": "봉책", "root": "속박", "disarm": "봉격"}
ACT = {"Move": "이동", "Enter": "입장", "Exit": "퇴장", "Face": "방향", "Look": "바라봄", "Play": "동작",
       "Emote": "말풍", "Pose": "자세", "Say": "말", "Wait": "대기", "After": "+초", "Effect": "효과",
       "Place": "배치", "Shake": "흔들림", "Strike": "타격", "React": "반응", "Follow": "따름"}


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


class Data:
    def __init__(self, scen):
        d = os.path.join(ROOT, "assets", "data")
        self.chars = load(os.path.join(d, "characters.json"))["Characters"]
        self.chars.update(load(os.path.join(d, "characters_story.json"))["Characters"])
        self.classes = load(os.path.join(d, "classes.json")).get("Classes", {})
        self.items = load(os.path.join(d, "items.json"))["Items"]
        self.equip = load(os.path.join(d, "equipment.json"))["Equipment"]
        game = load(os.path.join(d, "game.json"))
        self.grounds = game["Grounds"]
        self.structs = game["Structures"]
        self.scen = scen
        self.sc = load(os.path.join(scen, "scenario.json"))

    def name(self, cid):
        c = self.chars.get(cid)
        return c["Name"] if c else cid

    def names(self, joined):
        return "·".join(self.name(x) for x in joined.split("·"))

    def item(self, iid):
        return (self.items.get(iid) or self.equip.get(iid) or {}).get("Name", iid)

    def cls(self, cid, override=""):
        c = override or self.chars.get(cid, {}).get("Class", "")
        return self.classes.get(c, {}).get("Name", c)


def cond(ifs):
    return " & ".join(ifs) if ifs else ""


def xy(p):
    return "(%d,%d)" % (p[0], p[1])


def act_text(a):
    who = a.get("Actor", "")
    parts = []
    for k, v in a.items():
        if k == "Actor":
            continue
        label = ACT.get(k, k)
        if isinstance(v, list) and len(v) == 2 and all(isinstance(i, (int, float)) for i in v):
            v = xy(v)
        elif isinstance(v, (list, dict)):
            v = json.dumps(v, ensure_ascii=False)
        parts.append(label if v in (True, "") else "%s %s" % (label, v))
    return "%s: %s" % (who, ", ".join(parts)) if who else ", ".join(parts)


def groups_text(groups):
    out = []
    for g in groups:
        out.append(" + ".join(act_text(a) for a in g))
    return " → ".join(out)


def cast_text(cast):
    out = []
    for n, c in cast.items():
        s = "%s%s" % (n, xy(c["At"]) if c.get("At") else "")
        flags = []
        if c.get("Stay"):
            flags.append("탐험 자리 그대로")
        if c.get("Pose"):
            flags.append(c["Pose"])
        if c.get("Off"):
            flags.append("등장 전")
        if c.get("If"):
            flags.append("조건 " + cond(c["If"]))
        if flags:
            s += "[%s]" % ",".join(flags)
        out.append(s)
    return ", ".join(out)


class Chapter:
    def __init__(self, data, cid, fname, num, refs):
        self.d = data
        self.cid = cid
        self.num = num
        self.refs = refs
        self.nodes = load(os.path.join(data.scen, fname))["Nodes"]
        scenes = os.path.join(ROOT, "assets", "scenes", "ch%s.json" % cid)
        self.scenes = load(scenes)["Scenes"] if os.path.exists(scenes) else {}
        self.s = self.x = 0
        self.out = []

    def w(self, line=""):
        self.out.append(line)

    # ---- story flow -------------------------------------------------
    def render(self):
        first = next((n for n in self.nodes if n["Kind"] == "scene"), None)
        title = first["Place"] if first else ""
        self.w("# 제 %s 장 (%s) — %s" % (self.num, self.cid, title))
        self.w()
        scene = None
        beat_line = {}
        for n in self.nodes:
            k, nid = n["Kind"], n["ID"]
            if nid in self.refs and k != "scene":
                self.w()
                self.w("> ▶ 호출: %s" % self.refs[nid])
            if k == "scene":
                self.s += 1
                scene = self.scenes.get(nid)
                self.scene_head(n, scene)
                beat_line = {}
            elif k == "say":
                self.say(n, scene, beat_line)
            elif k == "narrate":
                tag = "장 끝 카드" if n.get("Card") else "나레이션"
                self.w("- *%s* %s `%s`" % (tag, n["Text"].replace("\n", " / "), nid))
            elif k == "explore":
                self.explore(n)
            elif k == "battle":
                self.battle(n)
            else:
                self.flow(n)
        return "\n".join(self.out) + "\n"

    def scene_head(self, n, sc):
        self.w()
        self.w("## S%s-#%d 장면: %s  `%s`" % (self.num, self.s, n["Place"], n["ID"]))
        if not sc:
            self.w("(장면 연출 데이터 없음)")
            return
        meta = ["맵 %s" % sc["Map"]]
        if sc.get("Time"):
            meta.append("시간 %s" % sc["Time"])
        self.w("*" + ", ".join(meta) + "*")
        self.w()
        self.w("- 등장: " + cast_text(sc.get("Cast", {})))
        for a in sc.get("Ambient", []):
            self.w("- 주변: " + act_text(a))
        self.w()

    def say(self, n, sc, beat_line):
        nid = n["ID"]
        beats = (sc or {}).get("Beats", {}).get(nid, [])
        for i, g in enumerate(beats):
            if g:
                tag = "" if i == 0 else "(줄 %d 앞) " % (i + 1)
                self.w("  - 〔연출〕 %s%s" % (tag, groups_text(g)))
        self.w("- **%s**: %s `%s`" % (self.d.names(n["Speaker"]), n["Text"], nid))

    def flow(self, n):
        k, nid = n["Kind"], n["ID"]
        d = self.d
        if k == "set":
            txt = "플래그 설정 " + ", ".join("%s=%s" % kv for kv in n["Flags"].items())
        elif k == "branch":
            cs = ["%s → `%s`" % (cond(c["If"]), c["Next"]) for c in n["Cases"]]
            txt = "분기: " + "; ".join(cs) + ("; 그 외 → `%s`" % n["Else"] if n.get("Else") else "; 그 외 → 다음")
        elif k == "join":
            txt = "합류: %s" % d.name(n["Character"])
        elif k == "leave":
            txt = "이탈: %s" % d.name(n["Character"])
        elif k == "grant":
            who = " → %s 착용" % d.name(n["Character"]) if n.get("Character") else " → 창고/인벤토리"
            txt = "획득: %s ×%d%s" % (d.item(n["Item"]), n.get("Count", 1), who)
        elif k == "choice":
            txt = "선택지 “%s”: %s" % (n.get("Text", ""), ", ".join("%s → `%s`" % kv for kv in n["Choices"].items()))
        elif k == "end":
            txt = "이야기 끝" + (": " + n["Text"] if n.get("Text") else "")
        else:
            txt = "%s %s" % (k, json.dumps(n, ensure_ascii=False))
        nxt = " (다음 → `%s`)" % n["Next"] if n.get("Next") else ""
        self.w("- ▷ %s%s `%s`" % (txt, nxt, nid))

    # ---- exploration ------------------------------------------------
    def explore(self, n):
        self.x += 1
        ex = load(os.path.join(self.d.scen, n["Explore"] + ".json"))
        self.w()
        self.w("## X%s-#%d 탐험  `%s` (%s)" % (self.num, self.x, n["ID"], n["Explore"]))
        st = ex["Start"]
        start_area = ex["Areas"][st["Area"]]
        self.w("*조작 인물 %s, 시작 %s %s*" % (self.d.name(ex["Hero"]), start_area["Name"], xy(st["At"])))
        if n.get("Opening"):
            self.w("- 시작 이벤트: `%s`" % n["Opening"])
        self.w()
        self.w("**동선**")
        for aid, a in ex["Areas"].items():
            for e in a.get("Exits", []):
                dest = ex["Areas"][e["To"]]["Name"] if e.get("To") else "이벤트 `%s`" % e.get("Node")
                extra = ""
                if e.get("If"):
                    extra += " (조건: %s)" % cond(e.get("If"))
                if e.get("Shut"):
                    extra += " (막힘: “%s”)" % e["Shut"][0][1]
                self.w("- %s의 「%s」 %s → %s%s" % (a["Name"], e["Name"], xy(e["At"]), dest, extra))
        for aid, a in ex["Areas"].items():
            self.w()
            extras = []
            if a.get("Time") or a.get("Times"):
                extras.append("시간 %s" % (a.get("Time") or a.get("Times")))
            self.w("**%s** `%s` 맵 %s%s" % (a["Name"], aid, a["Map"], " · " + ", ".join(extras) if extras else ""))
            for sg in a.get("Signs", []):
                self.w("- 〔표지〕 %s %s: %s" % (sg["Name"], xy(sg["At"]), " / ".join(t[1] for t in sg.get("Talk", []))))
            for pn, p in a.get("People", {}).items():
                self.person(pn, p)

    def person(self, pn, p):
        label = pn.split("@")[0]
        bits = [xy(p["At"])]
        if p.get("If"):
            bits.append("조건 " + cond(p["If"]))
        if p.get("Pose"):
            bits.append(p["Pose"])
        if p.get("Move"):
            bits.append("이동 " + json.dumps(p["Move"], ensure_ascii=False))
        head = "- %s [%s]" % (label, ", ".join(bits))
        extras = []
        if p.get("Shop"):
            extras.append("🛒 상점 `%s`" % p["Shop"])
        if p.get("Sortie"):
            extras.append("⚔ 출진: “%s”" % p["Sortie"].get("Ask", ""))
        for ev in p.get("Event", []):
            extras.append("★ 이벤트 `%s`%s" % (ev["Node"], " (조건 %s)" % cond(ev["If"]) if ev.get("If") else ""))
        if p.get("Chatter"):
            extras.append("혼잣말 " + p["Chatter"])
        self.w(head + (" — " + "; ".join(extras) if extras else ""))
        for t in p.get("Talk", []):
            who = self.d.names(t[0]) if t[0] in self.d.chars or "·" in t[0] else (t[0] or label)
            self.w("  - %s: %s" % (who, t[1]))

    # ---- battle -----------------------------------------------------
    def battle(self, n):
        st = load(os.path.join(self.d.scen, "stage%s.json" % stage_file(self.d.scen, n["Stage"])))
        self.w()
        self.w("## B%s 전투: %s  `%s`" % (st["ID"][1:], st["Name"], n["ID"]))
        self.w("*%s*" % n.get("Text", ""))
        self.w()
        self.out.extend(Stage(self.d, st).render())
        if n.get("Next"):
            self.w("- 승리 뒤 → `%s`" % n["Next"])


def stage_file(scen, sid):
    for fn in sorted(os.listdir(scen)):
        if fn.startswith("stage") and fn.endswith(".json"):
            if load(os.path.join(scen, fn))["ID"] == sid:
                return fn[5:-5]
    raise SystemExit("stage %s not found" % sid)


class Stage:
    def __init__(self, data, st):
        self.d = data
        self.st = st
        mp = load(os.path.join(data.scen, "maps", st["Map"]))
        self.map = mp
        p = mp["Play"]
        self.ox, self.oy, self.w_, self.h_ = p["X"], p["Y"], p["W"], p["H"]

    def rel(self, x, y):
        return x - self.ox, y - self.oy

    def inside(self, x, y):
        rx, ry = self.rel(x, y)
        return 0 <= rx < self.w_ and 0 <= ry < self.h_

    def sp(self, s):
        """spawn -> text"""
        d = self.d
        rx, ry = self.rel(s["X"], s["Y"])
        c = d.chars.get(s["Character"], {})
        nm = d.name(s["Character"])
        cl = d.cls(s["Character"], s.get("Class", ""))
        bits = [nm, cl, RANK.get(s.get("Rank", ""), "")]
        pol = POLICY.get(s.get("Policy", ""), "")
        if pol:
            pol_txt = pol
            if s.get("Goal"):
                gx, gy = self.rel(s["Goal"]["X"], s["Goal"]["Y"])
                pol_txt += "→(%d,%d)" % (gx, gy)
            bits.append(pol_txt)
        if s.get("HPPct"):
            bits.append("병력 %d%%" % s["HPPct"])
        if s.get("Group"):
            bits.append("무리 " + s["Group"])
        out = " ".join(b for b in bits if b) + " (%d,%d)" % (rx, ry)
        if not self.inside(s["X"], s["Y"]):
            out += " ⚠맵 밖"
        return out

    def render(self):
        st, d = self.st, self.d
        w = []
        w.append("- 맵 `%s` 전투 영역 %d×%d%s%s" % (st["Map"], self.w_, self.h_, " · 배로 건넘" if st.get("Boats") else "",
                                                " · 밤" if st.get("Night") else ""))
        w.append("- 출전 최대 %d명%s%s" % (st["Max"], " · 필수 " + ", ".join(d.name(c) for c in st.get("Required", [])) if st.get("Required") else "",
                                      " · 제한 %d턴" % st["TurnLimit"] if st.get("TurnLimit") else " · 턴 제한 없음"))
        for g in st.get("Groups", []):
            w.append("- 출전 무리 「%s」 구역 %d, 필수 %s" % (g["Name"], g["Area"], ", ".join(d.name(c) for c in g.get("Required", [])) or "없음"))
        w.append("- 승리: " + " / ".join("%s `%s`" % (v["Text"], cond(v.get("If"))) for v in st["Victory"]))
        for v in st.get("Defeat", []):
            w.append("- 패배: %s `%s`" % (v.get("Text", ""), cond(v.get("If"))))
        for s in st.get("Sub", []):
            rew = ", ".join("%s×%d" % (d.item(k), n) for k, n in s.get("Reward", {}).items())
            w.append("- 부목표: %s `%s`%s" % (s["Text"], cond(s.get("If")), " → " + rew if rew else ""))
        if st.get("Reward"):
            w.append("- 보상: " + ", ".join("%s×%d" % (d.item(k), n) for k, n in st["Reward"].items()))
        if st.get("Armies"):
            w.append("- 진영: " + ", ".join("%s%s" % (a["Name"], "(아군)" if a.get("Ally") else "") for a in st["Armies"]))
        w.append("")
        w += self.ascii_map()
        w.append("")
        w.append("**구역**")
        for a in st.get("Areas", []):
            r = a["Rect"]
            rx, ry = self.rel(r["X"], r["Y"])
            w.append("- %s: %s (%d,%d) %d×%d" % (a["ID"], a.get("Name", ""), rx, ry, r["W"], r["H"]) +
                     (" — " + a["Description"] if a.get("Description") else ""))
        w.append("")
        w.append("**처음 배치**")
        w.append("- 아군 자리: " + " ".join(xy(self.rel(a["X"], a["Y"])) for a in st["Allies"]))
        if st.get("Recommend"):
            w.append("- 권장 자리: " + ", ".join("%s%s" % (d.name(k), xy(self.rel(v["X"], v["Y"]))) for k, v in st["Recommend"].items()))
        for s in st.get("Party", []):
            w.append("- 고정 아군 " + self.sp(s))
        w += self.enemy_lines(st["Enemies"])
        if st.get("Duels"):
            w.append("")
            w.append("**일기토**")
            for du in st["Duels"]:
                w.append("- `%s` %s vs %s (%s): %s" % (du["ID"], d.name(du["Ally"]), d.name(du["Enemy"]), du.get("Outcome", ""), du.get("Text", "")))
                for b in du.get("Beats", []):
                    if b.get("Speech"):
                        w.append("  - %s: %s" % ("아군" if b["Side"] == 0 else "적" if b["Side"] == 1 else "지원", b["Speech"]))
        w.append("")
        w.append("**전투 중 사건**")
        for e in st.get("Events", []):
            head = "- `%s` 조건 `%s`" % (e["ID"], cond(e.get("If")))
            if e.get("Every"):
                head += " (매 %s턴)" % e["Every"]
            w.append(head)
            for do in e["Do"]:
                w += ["  - " + t for t in self.do_text(do)]
        return w

    def enemy_lines(self, enemies):
        d = self.d
        named, tmpl = [], {}
        for s in enemies:
            c = d.chars.get(s["Character"], {})
            if c.get("Template") or s["Character"] not in d.chars:
                key = (d.name(s["Character"]), d.cls(s["Character"], s.get("Class", "")), s.get("Group", ""), POLICY.get(s.get("Policy", ""), ""))
                tmpl.setdefault(key, []).append(s)
            else:
                named.append(s)
        out = ["- 적 장수 %d명, 병졸 %d명" % (len(named), sum(len(v) for v in tmpl.values()))]
        for s in named:
            out.append("  - " + self.sp(s))
        for (nm, cl, grp, pol), lst in tmpl.items():
            pos = " ".join(xy(self.rel(s["X"], s["Y"])) for s in lst)
            warn = " ⚠맵 밖" if any(not self.inside(s["X"], s["Y"]) for s in lst) else ""
            out.append("  - %s(%s) ×%d [무리 %s, %s] %s%s" % (nm, cl, len(lst), grp or "-", pol or "-", pos, warn))
        return out

    def ascii_map(self):
        st = self.st
        cells = self.map["Cells"]
        dflt = self.map.get("Default", "g--")
        grid = []
        for y in range(self.h_):
            row = []
            for x in range(self.w_):
                c = cells.get("%d,%d" % (x + self.ox, y + self.oy), dflt)
                s = self.d.structs.get(c[1]) if len(c) > 1 and c[1] != "-" else None
                g = self.d.grounds.get(c[0])
                row.append(s["Glyph"] if s else (g["Glyph"] if g else "?"))
            grid.append(row)
        legend = []
        letters = iter("ABCDEFGHIJKLMNOPQRSTUVWXYZ")

        def put(s, ch):
            rx, ry = self.rel(s["X"], s["Y"])
            if 0 <= rx < self.w_ and 0 <= ry < self.h_:
                grid[ry][rx] = ch

        for a in st["Allies"]:
            put(a, "+")
        for s in st.get("Party", []):
            put(s, "p")
        for s in st["Enemies"]:
            c = self.d.chars.get(s["Character"], {})
            if s["Character"] in self.d.chars and not c.get("Template"):
                try:
                    ch = next(letters)
                except StopIteration:
                    ch = "E"
                legend.append("%s=%s" % (ch, self.d.name(s["Character"])))
                put(s, ch)
            else:
                put(s, "e")
        out = ["**맵** (전투 좌표, 왼쪽 위가 (0,0))", "```"]
        out.append("    " + "".join(str((x // 10) % 10) if x % 10 == 0 else " " for x in range(self.w_)))
        out.append("    " + "".join(str(x % 10) for x in range(self.w_)))
        for y, row in enumerate(grid):
            out.append("%3d " % y + "".join(row))
        out.append("```")
        out.append("범례: `.`초원 `d`황무지 `~`물 `s`여울 `l`평지 `u`구릉 `n`산 `h`절벽 `c`성벽 `m`성문 `i`성내 `f`나무 `v`마을 `k`주둔지 `o`성 `g`관문 `b`다리 `z`울타리 · `+`아군 자리 `p`고정 아군 `e`적 병졸 · " + " ".join(legend))
        return out

    def do_text(self, do):
        k, d = do["Kind"], self.d
        if k == "say":
            return ["%s: %s" % (d.names(l["Actor"]), l["Text"]) for l in do["Lines"]]
        if k in ("spawn", "join"):
            return ["%s: %s" % ("증원" if k == "spawn" else "합류", "; ".join(self.sp(s) for s in do["Spawns"]))]
        if k == "buff":
            return ["상태: %s %s %s%s" % (",".join(d.name(t) for t in do["Targets"]), STAT.get(do["Stat"], do["Stat"]),
                                         do.get("Value", ""), " (%d턴)" % do["Turns"] if do.get("Turns") else "")]
        if k == "ai":
            return ["행동 변경: %s → %s" % (",".join(do["Targets"]), POLICY.get(do["Policy"], do["Policy"]) +
                                         (" %s" % xy(self.rel(do["Goal"]["X"], do["Goal"]["Y"])) if do.get("Goal") else ""))]
        if k == "remove":
            return ["퇴각시킴: " + ",".join(d.name(t) for t in do["Targets"])]
        if k == "flag":
            return ["플래그: " + ", ".join("%s=%s" % kv for kv in do["Flags"].items())]
        if k == "terrain":
            return ["지형 변화: 구역 %s → 상태 %s" % (do.get("Area"), do.get("State"))]
        if k == "goals":
            return ["부목표 추가: " + "; ".join(s["Text"] for s in do.get("Sub", []))]
        if k == "duel":
            return ["일기토 `%s`" % do["Duel"]]
        if k == "cutscene":
            parts = []
            for step in do["Steps"]:
                for s in step:
                    kind = s["Kind"]
                    who = d.name(s.get("Unit", ""))
                    if kind == "say":
                        parts.append("%s: “%s”" % (who, s.get("Text", "")))
                    elif kind == "move":
                        parts.append("%s 이동 %s" % (who, xy(self.rel(s["To"]["X"], s["To"]["Y"])) if isinstance(s.get("To"), dict) else who))
                    else:
                        parts.append("%s %s" % (who, kind))
            return ["연출: " + " / ".join(parts)]
        return ["%s %s" % (k, json.dumps(do, ensure_ascii=False))]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default="output")
    ap.add_argument("--scenario", default=os.path.join("assets", "scenario", "hero"))
    a = ap.parse_args()
    scen = a.scenario if os.path.isabs(a.scenario) else os.path.join(ROOT, a.scenario)
    out = a.out if os.path.isabs(a.out) else os.path.join(os.getcwd(), a.out)
    data = Data(scen)
    os.makedirs(os.path.join(out, "chapters"), exist_ok=True)

    refs = {}
    for ch in data.sc["Chapters"]:
        for n in load(os.path.join(scen, ch["File"]))["Nodes"]:
            if n["Kind"] != "explore":
                continue
            ex = load(os.path.join(scen, n["Explore"] + ".json"))
            if n.get("Opening"):
                refs[n["Opening"]] = "탐험 %s 시작 이벤트" % n["Explore"]
            for aa in ex["Areas"].values():
                for p_name, p in aa.get("People", {}).items():
                    for ev in p.get("Event", []):
                        refs[ev["Node"]] = "%s의 %s에게 말을 걸면%s" % (aa["Name"], p_name.split("@")[0],
                                                                  " (조건 %s)" % cond(ev["If"]) if ev.get("If") else "")
                for e in aa.get("Exits", []):
                    if e.get("Node"):
                        refs[e["Node"]] = "%s의 「%s」로 나가면" % (aa["Name"], e["Name"])

    all_md, index = [], ["# 시나리오 대본 (자동 생성)", "",
                         "게임 데이터에서 뽑은 현재 구현 상태다. 직접 고치지 말고 `make scenario-doc`으로 다시 만든다.", "",
                         "표기: `S장-#n` 장면, `X장-#n` 탐험, `B번호` 전투. 백틱 안은 데이터의 노드·이벤트 ID.", ""]
    for ch in data.sc["Chapters"]:
        num = "".join(c for c in ch["ID"] if c.isdigit()).lstrip("0") or "0"
        c = Chapter(data, ch["ID"], ch["File"], num, refs)
        md = c.render()
        name = "ch%s.md" % ch["ID"]
        with open(os.path.join(out, "chapters", name), "w", encoding="utf-8") as f:
            f.write(md)
        all_md.append(md)
        title = md.splitlines()[0][2:]
        index.append("- [%s](chapters/%s) — 장면 %d · 탐험 %d" % (title, name, c.s, c.x))
    with open(os.path.join(out, "scenario.md"), "w", encoding="utf-8") as f:
        f.write("\n---\n\n".join(all_md))
    with open(os.path.join(out, "README.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(index) + "\n")
    print("wrote %d chapters to %s" % (len(all_md), out))


if __name__ == "__main__":
    sys.exit(main())
