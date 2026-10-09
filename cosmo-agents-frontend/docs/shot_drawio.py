#!/usr/bin/env python3
"""Screenshot a .drawio file the way draw.io actually renders it.

preview_drawio.py draws the same straight centre-to-centre lines the layout
tools reason about, which makes it a restatement of those tools rather than a
check on them. That is exactly how a layout was passed as crossing-free while
draw.io drew something else: no edge in these files declares an `edgeStyle`, so
draw.io falls back to its own default, which is orthogonal routing. Straight
lines and right-angled routes cross in different places.

So this renders with draw.io's own viewer library in headless Chrome. Whatever
comes out is what the author sees when they open the file, because it is
produced by the same code.

Usage:  python3 shot_drawio.py file.drawio [...]   ->  file.png
"""

from __future__ import annotations

import base64
import html
import json
import pathlib
import subprocess
import sys
import tempfile

VIEWER = "https://viewer.diagrams.net/js/viewer.min.js"

PAGE = """<!DOCTYPE html><html><head><meta charset="utf-8">
<style>html,body{{margin:0;padding:0;background:#fff}}</style></head>
<body><div class="mxgraph" style="background:#fff" data-mxgraph="{cfg}"></div>
<script src="{viewer}"></script></body></html>"""


def shot(path: str, width: int, height: int) -> str:
    xml = pathlib.Path(path).read_text(encoding="utf-8")
    cfg = html.escape(json.dumps({
        "highlight": "#0000ff", "nav": False, "resize": True,
        "toolbar": "", "edit": None, "xml": xml,
    }), quote=True)
    out = str(pathlib.Path(path).with_suffix(".png"))

    with tempfile.TemporaryDirectory() as tmp:
        page = pathlib.Path(tmp) / "p.html"
        page.write_text(PAGE.format(cfg=cfg, viewer=VIEWER), encoding="utf-8")
        subprocess.run([
            "google-chrome", "--headless=new", "--disable-gpu", "--no-sandbox",
            "--hide-scrollbars", "--force-device-scale-factor=2",
            f"--window-size={width},{height}",
            # The viewer fetches and lays out before painting; without a budget
            # the screenshot lands on a blank page.
            "--virtual-time-budget=25000",
            f"--screenshot={out}", str(page),
        ], check=True, capture_output=True, timeout=180)
    return out


if __name__ == "__main__":
    w, h = 1900, 1500
    for p in sys.argv[1:]:
        print(shot(p, w, h), flush=True)
