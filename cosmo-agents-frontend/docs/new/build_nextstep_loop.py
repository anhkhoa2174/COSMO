#!/usr/bin/env python3
"""Build the next-step decision loop figure.

A column of five stages (observe, filter, select, approve, act) under the event
that starts a pass, a note beside each stage saying what it does, and one
return edge up the left side. A column keeps the text legible at page width;
a single row of five did not. Every edge is drawn as explicit
orthogonal segments so draw.io draws what is written here.
Writes nextstep-decision-loop.drawio.
"""

from __future__ import annotations

import html
import pathlib

BW, BH = 200, 62          # stage box
NW = 430                  # note box
ROW = 84
X0, Y0 = 90, 20
NX = X0 + BW + 34

STAGES = [
    ("1. Observe", "code",
     "Build the situation: stage, sends so far, time since last contact,\nreply intent and its details (return date, referred person, revisit\ndate, objection), meetings, lead score"),
    ("2. Filter", "code",
     "Apply eligibility rules: opt-out, bounce, wait window, meeting\nbooked, send cap, campaign owns the sending.\nOutput: the set of eligible actions"),
    ("3. Select", "LLM, bounded",
     "One eligible action: take it. Several: the LLM picks from the\neligible set only. Invalid answer or timeout: default cadence"),
    ("4. Approve", "human",
     "Internal actions run automatically. Outbound messages need\napproval. Sensitive replies go to a person to decide"),
    ("5. Act", "code",
     "Write the draft for the chosen action, or send nothing at all.\nStore the decision record"),
]

FILL = {"code": "#dae8fc", "LLM, bounded": "#e1d5e7", "human": "#d5e8d4"}
STROKE = {"code": "#6c8ebf", "LLM, bounded": "#9673a6", "human": "#82b366"}

cells: list[str] = []
_id = 1


def nid() -> str:
    global _id
    _id += 1
    return f"n{_id}"


def box(x, y, w, h, label, style) -> str:
    i = nid()
    cells.append(
        f'<mxCell id="{i}" value="{html.escape(label)}" style="{style}" vertex="1" parent="1">'
        f'<mxGeometry x="{x}" y="{y}" width="{w}" height="{h}" as="geometry"/></mxCell>')
    return i


def edge(points, label="", dashed=False, arrow=True) -> None:
    i = nid()
    (sx, sy), (tx, ty) = points[0], points[-1]
    mid = "".join(f'<mxPoint x="{x}" y="{y}"/>' for x, y in points[1:-1])
    style = ("html=1;rounded=0;strokeWidth=1.5;fontSize=13;"
             + ("endArrow=block;endFill=1;" if arrow else "endArrow=none;")
             + ("dashed=1;strokeColor=#999999;" if dashed else ""))
    cells.append(
        f'<mxCell id="{i}" value="{html.escape(label)}" style="{style}" edge="1" parent="1">'
        f'<mxGeometry relative="1" as="geometry"><mxPoint x="{sx}" y="{sy}" as="sourcePoint"/>'
        f'<mxPoint x="{tx}" y="{ty}" as="targetPoint"/>'
        + (f'<Array as="points">{mid}</Array>' if mid else "") + '</mxGeometry></mxCell>')


def y(row: int) -> float:
    return Y0 + row * ROW


# row 0: the event that starts a pass
box(X0, y(0), BW, BH, "<b>Event</b><br>sent · reply · timer · meeting · bounce",
    "rounded=1;whiteSpace=wrap;html=1;fillColor=#fff2cc;strokeColor=#d6b656;fontSize=13;")

for k, (title, who, note) in enumerate(STAGES, start=1):
    box(X0, y(k), BW, BH, f"<b>{title}</b><br><i>{who}</i>",
        f"rounded=1;whiteSpace=wrap;html=1;fillColor={FILL[who]};strokeColor={STROKE[who]};fontSize=14;")
    box(NX, y(k), NW, BH, note.replace("\n", "<br>"),
        "rounded=0;whiteSpace=wrap;html=1;fillColor=#f8f8f8;strokeColor=#b3b3b3;fontSize=12;"
        "align=left;spacingLeft=8;")
    edge([(X0 + BW, y(k) + BH / 2), (NX, y(k) + BH / 2)], dashed=True, arrow=False)
    edge([(X0 + BW / 2, y(k - 1) + BH), (X0 + BW / 2, y(k))])

# return edge: out of Act on the left, up the margin, into the event
left = X0 - 40
edge([(X0, y(5) + BH / 2), (left, y(5) + BH / 2), (left, y(0) + BH / 2), (X0, y(0) + BH / 2)])
box(left - 44, y(2) + 20, 40, 150, "the next event starts a new pass",
    "text;html=1;align=center;verticalAlign=middle;fontSize=12;fontStyle=2;horizontal=0;")

xml = ('<mxfile host="app.diagrams.net"><diagram name="next-step loop">'
       '<mxGraphModel dx="900" dy="700" grid="0" gridSize="10" guides="1" tooltips="1" connect="1" '
       'arrows="1" fold="1" page="0" pageScale="1" math="0" shadow="0"><root>'
       '<mxCell id="0"/><mxCell id="1" parent="0"/>' + "".join(cells) +
       '</root></mxGraphModel></diagram></mxfile>')
out = pathlib.Path(__file__).with_name("nextstep-decision-loop.drawio")
out.write_text(xml, encoding="utf-8")
print("wrote", out.name, len(cells), "cells")
