#!/usr/bin/env python3
"""Independent check that a .drawio use case diagram has no crossing edges
and no edge running through a box.

Deliberately shares no code with the layout tool: it re-reads the saved file
and rebuilds the geometry from what draw.io itself would use — the exitX and
entryX anchors written into each edge style, plus the waypoint array — so a
bug in the layout tool cannot hide behind its own verifier.

Usage:  python3 check_drawio.py file.drawio [...]
"""

from __future__ import annotations

import itertools
import re
import sys
import xml.etree.ElementTree as ET

EPS = 1e-9


def style_dict(style: str) -> dict[str, str]:
    out: dict[str, str] = {}
    for part in (style or "").split(";"):
        if "=" in part:
            k, v = part.split("=", 1)
            out[k.strip()] = v.strip()
    return out


def load(path: str):
    root = ET.parse(path).getroot()
    nodes: dict[str, dict] = {}
    edges: list[dict] = []

    for cell in root.iter("mxCell"):
        geo = cell.find("mxGeometry")
        cid = cell.get("id")
        if cell.get("vertex") == "1" and geo is not None:
            nodes[cid] = {
                "x": float(geo.get("x", 0)), "y": float(geo.get("y", 0)),
                "w": float(geo.get("width", 0)), "h": float(geo.get("height", 0)),
                "style": cell.get("style", ""),
                "label": re.sub(r"<[^>]+>", "", cell.get("value", "") or "")[:30],
            }
        elif cell.get("edge") == "1":
            pts = []
            if geo is not None:
                arr = geo.find("Array")
                if arr is not None:
                    pts = [(float(p.get("x")), float(p.get("y")))
                           for p in arr.findall("mxPoint")]
            edges.append({
                "id": cid, "src": cell.get("source"), "tgt": cell.get("target"),
                "pts": pts, "st": style_dict(cell.get("style", "")),
            })
    return nodes, edges


def anchor_from_style(n: dict, st: dict, which: str) -> tuple[float, float]:
    """Where draw.io attaches the edge, using the pinned anchor if present."""
    fx, fy = st.get(f"{which}X"), st.get(f"{which}Y")
    if fx is None or fy is None:
        return n["x"] + n["w"] / 2, n["y"] + n["h"] / 2
    return n["x"] + float(fx) * n["w"], n["y"] + float(fy) * n["h"]


def polyline(nodes, e) -> list[tuple[float, float]]:
    a, b = nodes[e["src"]], nodes[e["tgt"]]
    return [anchor_from_style(a, e["st"], "exit")] + e["pts"] + \
           [anchor_from_style(b, e["st"], "entry")]


def orient(ax, ay, bx, by, cx, cy) -> float:
    return (bx - ax) * (cy - ay) - (by - ay) * (cx - ax)


def on_seg(ax, ay, bx, by, px, py) -> bool:
    return (min(ax, bx) - EPS <= px <= max(ax, bx) + EPS
            and min(ay, by) - EPS <= py <= max(ay, by) + EPS)


def cross(s1, s2) -> bool:
    """True for a proper crossing or a collinear overlap.

    Collinear overlap counts: two edges drawn along the same line look like a
    single line to a reader, which is its own kind of wrong.
    """
    ax, ay, bx, by = s1
    cx, cy, dx, dy = s2
    if {(ax, ay), (bx, by)} & {(cx, cy), (dx, dy)}:
        return False

    d1 = orient(cx, cy, dx, dy, ax, ay)
    d2 = orient(cx, cy, dx, dy, bx, by)
    d3 = orient(ax, ay, bx, by, cx, cy)
    d4 = orient(ax, ay, bx, by, dx, dy)

    if abs(d1) < EPS and abs(d2) < EPS:  # collinear
        return (on_seg(ax, ay, bx, by, cx, cy) or on_seg(ax, ay, bx, by, dx, dy)
                or on_seg(cx, cy, dx, dy, ax, ay))
    return ((d1 > 0) != (d2 > 0)) and ((d3 > 0) != (d4 > 0))


def hits_box(s, n: dict, pad: float = 4.0) -> bool:
    x1, y1 = n["x"] - pad, n["y"] - pad
    x2, y2 = n["x"] + n["w"] + pad, n["y"] + n["h"] + pad
    ax, ay, bx, by = s
    if x1 < ax < x2 and y1 < ay < y2:
        return True
    if x1 < bx < x2 and y1 < by < y2:
        return True
    sides = [(x1, y1, x2, y1), (x2, y1, x2, y2), (x2, y2, x1, y2), (x1, y2, x1, y1)]
    return any(cross(s, side) for side in sides)


def check(path: str) -> bool:
    nodes, edges = load(path)
    drawn = []
    for e in edges:
        pts = polyline(nodes, e)
        for p, q in zip(pts, pts[1:]):
            drawn.append((e, (p[0], p[1], q[0], q[1])))

    problems: list[str] = []
    for (e1, s1), (e2, s2) in itertools.combinations(drawn, 2):
        if e1["id"] == e2["id"]:
            continue
        if cross(s1, s2):
            problems.append(
                f"cắt nhau: {nodes[e1['src']]['label']}->{nodes[e1['tgt']]['label']}"
                f"  ×  {nodes[e2['src']]['label']}->{nodes[e2['tgt']]['label']}")

    for e, s in drawn:
        for i, n in nodes.items():
            if i in (e["src"], e["tgt"]) or n["w"] < 5 or "text;" in n["style"]:
                continue
            if hits_box(s, n):
                problems.append(
                    f"xuyên ô: {nodes[e['src']]['label']}->{nodes[e['tgt']]['label']}"
                    f"  đè  {n['label']}")

    # Boxes must not sit on top of each other either.
    boxes = [(i, n) for i, n in nodes.items() if n["w"] >= 5 and "text;" not in n["style"]]
    for (i1, a), (i2, b) in itertools.combinations(boxes, 2):
        if (a["x"] < b["x"] + b["w"] and b["x"] < a["x"] + a["w"]
                and a["y"] < b["y"] + b["h"] and b["y"] < a["y"] + a["h"]):
            problems.append(f"chồng ô: {a['label']}  ×  {b['label']}")

    ok = not problems
    print(f"  {'OK ' if ok else 'XX '}{path:42s} "
          f"{len(nodes)} ô, {len(edges)} cạnh, {len(drawn)} đoạn")
    for p in problems[:12]:
        print(f"       {p}")
    return ok


if __name__ == "__main__":
    sys.exit(0 if all(check(p) for p in sys.argv[1:]) else 1)
