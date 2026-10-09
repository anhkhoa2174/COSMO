#!/usr/bin/env python3
"""Render a .drawio use case diagram to SVG so the layout can be eyeballed
without opening draw.io.

It reads the file as written, waypoints included, so what the SVG shows is
what the .drawio actually contains — a check on the layout tool rather than a
restatement of it.

Usage:  python3 preview_drawio.py file.drawio [...]   ->  file.svg
"""

from __future__ import annotations

import html
import math
import re
import sys
import xml.etree.ElementTree as ET

PAD = 40


def plain(value: str) -> str:
    text = re.sub(r"<br\s*/?>", "\n", html.unescape(value or ""))
    text = re.sub(r"<[^>]+>", "", text)
    return html.unescape(text).replace("&laquo;", "«").replace("&raquo;", "»").strip()


def render(path: str) -> str:
    root = ET.parse(path).getroot()

    nodes: dict[str, dict] = {}
    edges: list[dict] = []

    for cell in root.iter("mxCell"):
        cid = cell.get("id")
        geo = cell.find("mxGeometry")
        if cell.get("vertex") == "1" and geo is not None:
            nodes[cid] = {
                "x": float(geo.get("x", 0)),
                "y": float(geo.get("y", 0)),
                "w": float(geo.get("width", 0)),
                "h": float(geo.get("height", 0)),
                "label": plain(cell.get("value", "")),
                "style": cell.get("style", ""),
            }
        elif cell.get("edge") == "1":
            pts = []
            if geo is not None:
                arr = geo.find("Array")
                if arr is not None:
                    pts = [(float(p.get("x")), float(p.get("y")))
                           for p in arr.findall("mxPoint")]
            edges.append({
                "src": cell.get("source"), "tgt": cell.get("target"),
                "pts": pts, "label": plain(cell.get("value", "")),
                "style": cell.get("style", ""),
            })

    W = int(max((n["x"] + n["w"] for n in nodes.values()), default=800) + PAD)
    H = int(max((n["y"] + n["h"] for n in nodes.values()), default=600) + PAD)

    out = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" '
        f'viewBox="0 0 {W} {H}" font-family="Helvetica,Arial,sans-serif">',
        f'<rect width="{W}" height="{H}" fill="#ffffff"/>',
    ]

    def anchor(n: dict, towards: tuple[float, float]) -> tuple[float, float]:
        cy = n["y"] + n["h"] / 2
        return (n["x"] + n["w"], cy) if towards[0] > n["x"] + n["w"] / 2 else (n["x"], cy)

    # The system boundary is drawn first and unfilled, otherwise it paints over
    # every connector inside it and the preview looks like the edges vanished.
    for cid, n in nodes.items():
        if n["w"] > 400 and n["h"] > 300 and "ellipse" not in n["style"]:
            out.append(f'<rect x="{n["x"]}" y="{n["y"]}" width="{n["w"]}" '
                       f'height="{n["h"]}" fill="none" stroke="#000" stroke-width="2"/>')
            if n["label"]:
                out.append(f'<text x="{n["x"] + 12}" y="{n["y"] + 22}" font-size="13" '
                           f'font-weight="bold" fill="#111">{html.escape(n["label"])}</text>')

    for e in edges:
        a, b = nodes.get(e["src"]), nodes.get(e["tgt"])
        if not a or not b:
            continue
        first = e["pts"][0] if e["pts"] else (b["x"] + b["w"] / 2, b["y"] + b["h"] / 2)
        last = e["pts"][-1] if e["pts"] else (a["x"] + a["w"] / 2, a["y"] + a["h"] / 2)
        pts = [anchor(a, first)] + e["pts"] + [anchor(b, last)]
        d = " ".join(f"{'M' if i == 0 else 'L'}{x:.1f},{y:.1f}"
                     for i, (x, y) in enumerate(pts))
        dashed = "dashed=1" in e["style"]
        out.append(
            f'<path d="{d}" fill="none" stroke="#000" stroke-width="1.3"'
            + (' stroke-dasharray="8,4"' if dashed else "") + "/>"
        )

        # An open arrowhead where the style asks for one. «include» and
        # «extend» are told apart by which end it sits on, so leaving it out
        # of the preview hid the very thing that distinguishes them.
        if "endArrow=open" in e["style"]:
            (px, py), (qx, qy) = pts[-2], pts[-1]
            ang = math.atan2(qy - py, qx - px)
            for side in (+1, -1):
                a = ang + side * 0.38
                out.append(
                    f'<path d="M{qx - 13 * math.cos(a):.1f},{qy - 13 * math.sin(a):.1f} '
                    f'L{qx:.1f},{qy:.1f}" fill="none" stroke="#000" stroke-width="1.3"/>'
                )

        if e["label"]:
            mid = len(pts) // 2
            mx = (pts[mid - 1][0] + pts[mid][0]) / 2
            my = (pts[mid - 1][1] + pts[mid][1]) / 2
            wid = len(e["label"]) * 5.4 + 6
            out.append(
                f'<rect x="{mx - wid / 2:.0f}" y="{my - 14:.0f}" width="{wid:.0f}" '
                f'height="14" fill="#ffffff"/>'
            )
            out.append(
                f'<text x="{mx:.0f}" y="{my - 3:.0f}" font-size="10" fill="#000" '
                f'text-anchor="middle">{html.escape(e["label"])}</text>'
            )

    for n in nodes.values():
        cx, cy = n["x"] + n["w"] / 2, n["y"] + n["h"] / 2
        if "umlActor" in n["style"]:
            r = 9
            out.append(f'<circle cx="{cx}" cy="{n["y"] + r}" r="{r}" fill="none" stroke="#333" stroke-width="1.6"/>')
            out.append(f'<path d="M{cx},{n["y"] + 2 * r} L{cx},{n["y"] + n["h"] - 18} '
                       f'M{cx - 14},{n["y"] + 30} L{cx + 14},{n["y"] + 30} '
                       f'M{cx},{n["y"] + n["h"] - 18} L{cx - 12},{n["y"] + n["h"]} '
                       f'M{cx},{n["y"] + n["h"] - 18} L{cx + 12},{n["y"] + n["h"]}" '
                       f'fill="none" stroke="#333" stroke-width="1.6"/>')
        elif "ellipse" in n["style"] or "rounded=1" in n["style"]:
            out.append(f'<ellipse cx="{cx}" cy="{cy}" rx="{n["w"] / 2}" ry="{n["h"] / 2}" '
                       f'fill="#ffffff" stroke="#000000" stroke-width="1.5"/>')
        elif "text;" in n["style"]:
            pass
        elif n["w"] > 400 and n["h"] > 300:
            continue  # boundary already drawn
        else:
            out.append(f'<rect x="{n["x"]}" y="{n["y"]}" width="{n["w"]}" height="{n["h"]}" '
                       f'rx="6" fill="#ffffff" stroke="#000000" stroke-width="1.3"/>')

        if n["w"] > 400 and n["h"] > 300 and "ellipse" not in n["style"]:
            continue
        lines = [l for l in n["label"].split("\n") if l.strip()]
        if not lines:
            continue
        wrapped: list[str] = []
        for line in lines:
            while len(line) > 24:
                cut = line.rfind(" ", 0, 24)
                cut = cut if cut > 0 else 24
                wrapped.append(line[:cut])
                line = line[cut:].strip()
            wrapped.append(line)

        below = "umlActor" in n["style"]
        size = 11 if not below else 10
        start = (n["y"] + n["h"] + 12) if below else cy - (len(wrapped) - 1) * 6.5
        for k, line in enumerate(wrapped):
            out.append(
                f'<text x="{cx:.0f}" y="{start + k * 13:.0f}" font-size="{size}" '
                f'text-anchor="middle" fill="#111">{html.escape(line)}</text>'
            )

    out.append("</svg>")
    return "\n".join(out)


if __name__ == "__main__":
    for p in sys.argv[1:]:
        svg = p.rsplit(".", 1)[0] + ".svg"
        open(svg, "w", encoding="utf-8").write(render(p))
        print(f"  {svg}")
