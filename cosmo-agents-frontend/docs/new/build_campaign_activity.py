#!/usr/bin/env python3
"""Build the Campaign Outreach activity diagram from what the code does.

Flow after activation, as implemented:
  campaign worker  -> one send task per contact and step
  ai worker        -> knowledge context (RAG) + LLM rewrite of the template
  email worker     -> fill merge tags and sender name, strip leftover
                      placeholders, send through the Gmail API, then save the
                      email, log the interaction and move COLD -> NO_REPLY
                      with next_step = WAIT (only after the send succeeded)

Every edge is drawn as explicit orthogonal segments, so the checks below test
exactly what draw.io draws. Writes activity-campaign-outreach.drawio.
"""

from __future__ import annotations

import html
import itertools
import pathlib
import sys

LANES = ["BD (Sales Rep)", "COSMO System", "LLM Service (OpenAI)", "Gmail Service", "Prospect"]
LW, TOP, HEAD = 250, 20, 30
ROW = 64
BW, BH = 180, 44


def lx(i: int) -> float:
    return 20 + i * LW


def cx(lane: int) -> float:
    return lx(lane) + LW / 2


def ry(row: float) -> float:
    return TOP + HEAD + 30 + row * ROW


BD, SYS, LLM, GM, PR = range(5)

# id: (kind, lane, row, label)
NODES = {
    "init": ("initial", BD, 0, ""),
    "a1": ("action", BD, 1, "Select target contacts\nfrom contact list"),
    "a2": ("action", BD, 2, "Choose playbook\nstrategy"),
    "a3": ("action", BD, 3, "Configure campaign\nname and settings"),
    "a4": ("action", SYS, 4, "Build prompt from\nplaybook + contacts"),
    "a5": ("action", LLM, 4, "Generate multi-step\nemail sequence"),
    "a6": ("action", SYS, 5, "Create email templates\nfrom LLM output"),
    "a7": ("action", BD, 6, "Review generated\nemail templates"),
    "d1": ("decision", BD, 7, ""),
    "a8": ("action", BD, 8, "Edit template\nmanually"),
    "d2": ("decision", BD, 9, ""),
    "a8b": ("action", LLM, 9, "Enhance draft with\nAI writing assistant"),
    "m1": ("decision", BD, 10, ""),
    "a9": ("action", BD, 11, "Activate campaign"),
    "a10": ("action", SYS, 12, "Schedule a send task\nper contact and step"),
    "a11": ("action", SYS, 13, "Retrieve knowledge\ncontext (RAG)"),
    "a12": ("action", LLM, 13, "Rewrite template\nfor this contact"),
    "a13": ("action", SYS, 14, "Fill merge tags and sender\nname, drop leftover placeholders"),
    "a14": ("action", SYS, 15, "Send email request\nto Gmail API"),
    "fork": ("bar", SYS, 16, ""),
    "a15": ("action", GM, 17, "Deliver email\nvia Gmail"),
    "a16": ("action", PR, 17, "Receive\ncampaign email"),
    "a17": ("action", SYS, 17, "Save email record and\nlog the interaction"),
    "a18": ("action", SYS, 18, "Update contact state\nCOLD to NO_REPLY,\nnext_step = WAIT"),
    "join": ("bar", SYS, 19.2, ""),
    "final": ("final", SYS, 20, ""),
}

BAR_SPAN = {"fork": (SYS, GM), "join": (SYS, PR)}


def box(k: str) -> tuple[float, float, float, float]:
    kind, lane, row, _ = NODES[k]
    if kind == "bar":
        a, b = BAR_SPAN[k]
        x0, x1 = cx(a) - 70, cx(b) + 70
        return x0, ry(row) + BH / 2 - 4, x1 - x0, 8
    if kind in ("initial", "final"):
        return cx(lane) - 15, ry(row) + BH / 2 - 15, 30, 30
    if kind == "decision":
        return cx(lane) - 20, ry(row) + BH / 2 - 20, 40, 40
    h = BH + 14 if k == "a18" else BH
    return cx(lane) - BW / 2, ry(row), BW, h


def side(k: str, where: str, x: float | None = None) -> tuple[float, float]:
    x0, y0, w, h = box(k)
    return {
        "top": (x if x is not None else x0 + w / 2, y0),
        "bottom": (x if x is not None else x0 + w / 2, y0 + h),
        "left": (x0, y0 + h / 2),
        "right": (x0 + w, y0 + h / 2),
    }[where]


def mid_y(k: str) -> float:
    x0, y0, w, h = box(k)
    return y0 + h / 2


L_BD_LEFT = lx(BD) + 18          # gutter for the [satisfied] bypass
ABOVE = 20                       # loop-back runs this far above a row

# (src, src side, tgt, tgt side, waypoints, label)
EDGES = [
    ("init", "bottom", "a1", "top", [], ""),
    ("a1", "bottom", "a2", "top", [], ""),
    ("a2", "bottom", "a3", "top", [], ""),
    ("a3", "bottom", "a4", "left", [(cx(BD), mid_y("a4"))], ""),
    ("a4", "right", "a5", "left", [], ""),
    ("a5", "bottom", "a6", "right", [(cx(LLM), mid_y("a6"))], ""),
    ("a6", "left", "a7", "top", [(cx(BD), mid_y("a6"))], ""),
    ("a7", "bottom", "d1", "top", [], ""),
    ("d1", "bottom", "a8", "top", [], "[needs edit]"),
    ("d1", "left", "m1", "left", [(L_BD_LEFT, mid_y("d1")), (L_BD_LEFT, mid_y("m1"))], "[satisfied]"),
    ("a8", "bottom", "d2", "top", [], ""),
    ("d2", "right", "a8b", "left", [], "[use AI]"),
    ("d2", "bottom", "m1", "top", [], "[done]"),
    ("a8b", "top", "a7", "right", [(cx(LLM), mid_y("a7"))], ""),
    ("m1", "bottom", "a9", "top", [], ""),
    ("a9", "bottom", "a10", "left", [(cx(BD), mid_y("a10"))], ""),
    ("a10", "bottom", "a11", "top", [], ""),
    ("a11", "right", "a12", "left", [], ""),
    ("a12", "bottom", "a13", "right", [(cx(LLM), mid_y("a13"))], ""),
    ("a13", "bottom", "a14", "top", [], ""),
    ("a14", "bottom", "fork", "top", [], ""),
    ("fork", "bottom", "a17", "top", [], ""),
    ("fork", "bottom@gm", "a15", "top", [], ""),
    ("a15", "right", "a16", "left", [], ""),
    ("a17", "bottom", "a18", "top", [], ""),
    ("a18", "bottom", "join", "top", [], ""),
    ("a16", "bottom", "join", "top@pr", [], ""),
    ("join", "bottom", "final", "top", [], ""),
]


def endpoint(k: str, spec: str) -> tuple[float, float]:
    if "@" in spec:
        where, lane = spec.split("@")
        return side(k, where, cx({"gm": GM, "pr": PR}[lane]))
    return side(k, spec)


def polyline(e):
    s, ss, t, ts, pts, _ = e
    a, b = endpoint(s, ss), endpoint(t, ts)
    # A bar is joined straight above or below the node it connects to.
    if NODES[s][0] == "bar" and "@" not in ss:
        a = (b[0], a[1])
    if NODES[t][0] == "bar" and "@" not in ts:
        b = (a[0], b[1])
    return [a, *pts, b]


def esc(s: str) -> str:
    return html.escape(s, quote=True).replace("\n", "&#xa;")


def build() -> str:
    cells = ['<mxCell id="0"/>', '<mxCell id="1" parent="0"/>']
    height = ry(20) + 80 - TOP
    for i, name in enumerate(LANES):
        cells.append(
            f'<mxCell id="lane{i}" value="{esc(name)}" style="swimlane;startSize={HEAD};html=1;'
            f'fillColor=#FFFFFF;strokeColor=#000000;strokeWidth=1.5;fontStyle=1;fontSize=12;" '
            f'vertex="1" parent="1"><mxGeometry x="{lx(i)}" y="{TOP}" width="{LW}" height="{height}" '
            f'as="geometry"/></mxCell>')
    for k, (kind, lane, row, label) in NODES.items():
        x, y, w, h = box(k)
        style = {
            "action": "rounded=1;whiteSpace=wrap;html=1;arcSize=20;fillColor=#FFFFFF;strokeColor=#000000;strokeWidth=1.5;fontSize=11;",
            "decision": "rhombus;whiteSpace=wrap;html=1;fillColor=#FFFFFF;strokeColor=#000000;strokeWidth=1.5;",
            "bar": "rounded=0;html=1;fillColor=#000000;strokeColor=#000000;",
            "initial": "ellipse;html=1;fillColor=#000000;strokeColor=#000000;",
            "final": "ellipse;html=1;shape=endState;fillColor=#000000;strokeColor=#000000;strokeWidth=1.5;",
        }[kind]
        cells.append(
            f'<mxCell id="{k}" value="{esc(label)}" style="{style}" vertex="1" parent="1">'
            f'<mxGeometry x="{x}" y="{y}" width="{w}" height="{h}" as="geometry"/></mxCell>')
    for i, e in enumerate(EDGES):
        s, _, t, _, _, label = e
        pts = polyline(e)
        (sx, sy), (tx, ty) = pts[0], pts[-1]
        xml_pts = "".join(f'<mxPoint x="{x}" y="{y}"/>' for x, y in pts[1:-1])
        cells.append(
            f'<mxCell id="e{i}" value="{esc(label)}" style="edgeStyle=none;html=1;endArrow=open;'
            f'endSize=8;strokeColor=#000000;strokeWidth=1;fontSize=10;fontStyle=2;'
            f'labelBackgroundColor=#FFFFFF;" edge="1" parent="1" source="{s}" target="{t}">'
            f'<mxGeometry relative="1" as="geometry"><mxPoint x="{sx}" y="{sy}" as="sourcePoint"/>'
            f'<mxPoint x="{tx}" y="{ty}" as="targetPoint"/>'
            + (f'<Array as="points">{xml_pts}</Array>' if xml_pts else "")
            + '</mxGeometry></mxCell>')
    # Pin every end exactly where the checks assume.
    out = []
    for c in cells:
        out.append(c)
    xml = "".join(out)
    for i, e in enumerate(EDGES):
        s, _, t, _, _, _ = e
        (sx, sy), (tx, ty) = polyline(e)[0], polyline(e)[-1]
        bs, bt = box(s), box(t)
        anchors = (f"exitX={(sx - bs[0]) / bs[2]:.4f};exitY={(sy - bs[1]) / bs[3]:.4f};exitDx=0;exitDy=0;"
                   f"entryX={(tx - bt[0]) / bt[2]:.4f};entryY={(ty - bt[1]) / bt[3]:.4f};entryDx=0;entryDy=0;")
        xml = xml.replace(f'<mxCell id="e{i}" value=', f'<mxCell id="e{i}" ANCH="{anchors}" value=', 1)
    import re
    xml = re.sub(r'ANCH="([^"]*)" (value="[^"]*") style="', r'\2 style="\1', xml)
    return ('<mxfile host="app.diagrams.net"><diagram id="campaign" name="Campaign Outreach Activity">'
            '<mxGraphModel grid="0" page="0" math="0" shadow="0"><root>' + xml +
            "</root></mxGraphModel></diagram></mxfile>")


def cross(p1, p2, p3, p4) -> bool:
    def o(a, b, c):
        return (b[0] - a[0]) * (c[1] - a[1]) - (b[1] - a[1]) * (c[0] - a[0])
    d1, d2, d3, d4 = o(p3, p4, p1), o(p3, p4, p2), o(p1, p2, p3), o(p1, p2, p4)
    return (d1 > 0) != (d2 > 0) and (d3 > 0) != (d4 > 0)


def through(p, q, b, pad=3) -> bool:
    x, y, w, h = b
    for i in range(1, 100):
        px = p[0] + (q[0] - p[0]) * i / 100
        py = p[1] + (q[1] - p[1]) * i / 100
        if x + pad < px < x + w - pad and y + pad < py < y + h - pad:
            return True
    return False


def check() -> list[str]:
    segs = []
    for e in EDGES:
        pts = polyline(e)
        for p, q in zip(pts, pts[1:]):
            segs.append(((e[0], e[2]), p, q))
    problems = []
    for (e1, a1, b1), (e2, a2, b2) in itertools.combinations(segs, 2):
        if e1 == e2 or set(e1) & set(e2):
            continue
        if cross(a1, b1, a2, b2):
            problems.append(f"crossing {e1} x {e2}")
    for e, p, q in segs:
        for k in NODES:
            if k not in e and through(p, q, box(k)):
                problems.append(f"{e} runs through {k}")
    for k1, k2 in itertools.combinations(NODES, 2):
        a, b = box(k1), box(k2)
        if a[0] < b[0] + b[2] and b[0] < a[0] + a[2] and a[1] < b[1] + b[3] and b[1] < a[1] + a[3]:
            problems.append(f"overlap {k1} x {k2}")
    return problems


if __name__ == "__main__":
    out = pathlib.Path(__file__).with_name("activity-campaign-outreach.drawio")
    out.write_text(build(), encoding="utf-8")
    problems = check()
    print("\n".join(problems))
    print(f"{out.name}: {len(NODES)} nodes, {len(EDGES)} edges, {len(problems)} problems")
    sys.exit(1 if problems else 0)
