#!/usr/bin/env python3
"""Build the Overall Use Case diagram with a layout that has no crossings.

The layout follows three rules, which together keep every line clear:

* Every use case BD starts sits in one column next to BD, so BD's lines fan out
  to a single column and cannot cross each other. Admin sits below BD and its
  use cases sit below BD's, so the two fans stay apart.
* Everything the LLM touches sits in the right-most column, next to the LLM, so
  its fan has nothing in the way either.
* Each relationship between use cases runs inside its own band of rows.

Writes overall-usecase.drawio, then checks every edge against every other edge
and against every use case box. Exits non-zero on any problem.
"""

from __future__ import annotations

import html
import itertools
import math
import pathlib
import sys

ROW = 82          # vertical pitch between use cases
TOP = 60          # y of row 0
W, H = 170, 58    # use case ellipse

COL1, COL2 = 330, 590           # BD-facing column, and the column it extends to
R1, R2 = 1010, 1250             # automatic pipeline: inner, and next to the LLM


def y(row: float) -> float:
    return TOP + row * ROW


# id: (label, x, row)
USE_CASES = {
    # Contact & Intelligence
    "manage_contacts": ("Manage Contacts", COL1, 0),
    "import_contacts": ("Import Contacts", COL1, 1.5),
    "validate_insight": ("Validate AI\nInsight", COL1, 3),
    "enrich_contact": ("Enrich Contact\nwith AI", COL1, 4.2),
    "import_apollo": ("Import via\nApollo MCP", COL2, 0),
    "import_leadform": ("Capture via\nLead Form", COL2 + 200, 1),
    "import_linkedin": ("Import via\nLinkedIn Extension", COL2, 2),
    "import_csv": ("Import via\nCSV Upload", COL2, 3),
    # Campaign & Outreach
    "create_campaign": ("Create Campaign", COL1, 6.3),
    "choose_playbook": ("Choose Playbook\nStrategy", COL2, 7.3),
    # AI Inbox & Reply
    "view_conversations": ("View\nConversations", COL1, 8.8),
    "review_reply": ("Review AI\nSuggested Reply", COL1, 9.8),
    "manual_reply": ("Reply Manually", COL1, 10.8),
    "edit_reply": ("Edit & Send\nAI Reply", COL2, 9.3),
    "approve_reply": ("Approve & Send\nAI Reply", COL2, 10.3),
    "correct_intent": ("Correct Reply\nIntent", COL1, 11.8),
    "rewrite_draft": ("Rewrite Stale\nAI Draft", R2 - 90, 11.8),
    # Ask COSMO assistant
    "query_inapp": ("Query In-App\nAssistant", COL1, 13.3),
    "generate_answer": ("Generate Answer", R2, 13.3),
    "query_public": ("Query Public\nAssistant", R2, 14.5),
    # Monitoring & Administration
    "upload_knowledge": ("Upload Knowledge\nDocuments", COL1, 16),
    "daily_actions": ("View Daily Actions\nBriefing", COL1, 17.15),
    "notifications": ("Receive Real-time\nNotifications (SSE)", COL1, 18.35),
    "assign_conversation": ("Assign Conversation\nto Member", COL1, 19.8),
    "manage_lead_forms": ("Manage\nLead Forms", COL1, 20.8),
    # Automatic AI pipeline
    "send_campaign": ("Send Campaign\nEmails", R1, 0),
    "transition_state": ("Transition Pipeline\nState", R2, 0),
    "classify_intent": ("Classify Reply\nIntent", R2, 1.2),
    "generate_draft": ("Generate AI\nDraft Reply", R2, 2.4),
    "rag": ("Retrieve Knowledge\n(RAG)", R1, 2.4),
    "calc_scores": ("Calculate Segment\nScores", R1, 4),
    "embed": ("Generate Vector\nEmbedding", R2, 5),
    "insights": ("Generate AI\nInsights", R2, 6.1),
    "email_sequence": ("Generate Email\nSequence (AI)", R2, 7.3),
}

# id: (label, x, y, kind)  kind: actor | system
ACTORS = {
    "bd": ("BD\n(Sales Rep)", -380, y(10.6) - 30, "actor"),
    "admin": ("Admin", -380, y(20.3) - 30, "actor"),
    "prospect": ("Prospect", COL2 + 200 + W / 2 - 20, -150, "actor"),
    "apollo": ("«system»\nApollo API\n(MCP)", COL2 + W / 2 - 60, -160, "system"),
    "gmail": ("«system»\nGmail\n(Pub/Sub)", 1560, y(1.2) + H / 2 - 35, "system"),
    "llm": ("«system»\nLLM Service\n(OpenAI)", 1640, y(5.2) + H / 2 - 35, "system"),
    "visitor": ("Visitor\n(unauthenticated)", 1600, y(14.5) - 1, "actor"),
}

ASSOC, INCLUDE, EXTEND, GENERAL = "assoc", "include", "extend", "general"
EDGES = [
    ("admin", "bd", GENERAL, ""),
    *[("bd", t, ASSOC, "") for t in (
        "manage_contacts", "import_contacts", "validate_insight", "enrich_contact",
        "create_campaign", "view_conversations", "review_reply", "manual_reply",
        "correct_intent", "query_inapp",
        "upload_knowledge", "daily_actions", "notifications")],
    ("admin", "assign_conversation", ASSOC, ""),
    ("admin", "manage_lead_forms", ASSOC, ""),
    ("prospect", "import_leadform", ASSOC, "submits"),
    ("prospect", "send_campaign", ASSOC, "receives"),
    ("apollo", "import_apollo", ASSOC, ""),
    ("gmail", "classify_intent", ASSOC, "pushes new email"),
    *[("llm", t, ASSOC, "") for t in (
        "classify_intent", "generate_draft", "embed", "insights", "email_sequence",
        "rewrite_draft", "generate_answer")],
    ("visitor", "query_public", ASSOC, ""),
    ("rewrite_draft", "correct_intent", EXTEND, "«extend»"),
    ("query_inapp", "generate_answer", INCLUDE, "«include»"),
    ("query_public", "generate_answer", INCLUDE, "«include»"),
    ("import_apollo", "import_contacts", EXTEND, "«extend»"),
    ("import_leadform", "import_contacts", EXTEND, "«extend»"),
    ("import_linkedin", "import_contacts", EXTEND, "«extend»"),
    ("import_csv", "import_contacts", EXTEND, "«extend»"),
    ("enrich_contact", "calc_scores", INCLUDE, "«include»"),
    ("enrich_contact", "embed", INCLUDE, "«include»"),
    ("enrich_contact", "insights", INCLUDE, "«include»"),
    ("create_campaign", "choose_playbook", INCLUDE, "«include»"),
    ("create_campaign", "email_sequence", INCLUDE, "«include»"),
    ("edit_reply", "review_reply", EXTEND, "«extend»"),
    ("approve_reply", "review_reply", EXTEND, "«extend»"),
    ("send_campaign", "transition_state", INCLUDE, "«include»"),
    ("classify_intent", "transition_state", INCLUDE, "«include»"),
    ("generate_draft", "classify_intent", INCLUDE, "«include»"),
    ("generate_draft", "rag", INCLUDE, "«include»"),
]

PAD = 22
GROUPS = [  # label, first row, last row, x from, x to
    ("Contact & Intelligence", 0, 4.2, COL1, COL2 + 200 + W),
    ("Campaign & Outreach", 6.3, 7.3, COL1, COL2 + W),
    ("AI Inbox & Reply (human-in-the-loop)", 8.8, 11.8, COL1, R2 + W),
    ("Ask COSMO Assistant (conversational)", 13.3, 14.5, COL1, R2 + W),
    ("Monitoring & Administration", 16, 20.8, COL1, COL1 + W),
    ("Automatic AI Pipeline (no human trigger)", 0, 7.3, R1, R2 + W),
]


def esc(s: str) -> str:
    return html.escape(s, quote=True).replace("\n", "&#xa;")


def boxes() -> dict[str, tuple[float, float, float, float, str]]:
    out = {k: (x, y(r), W, H, "uc") for k, (_, x, r) in USE_CASES.items()}
    for k, (_, x, yy, kind) in ACTORS.items():
        out[k] = (x, yy, 40, 60, kind) if kind == "actor" else (x, yy, 120, 70, kind)
    return out


def build() -> str:
    cells = ['<mxCell id="0"/>', '<mxCell id="1" parent="0"/>']
    sys_x, sys_y = COL1 - 50, y(0) - 60
    sys_w, sys_h = R2 + W + 40 - sys_x, y(20.8) + H + 30 - sys_y
    cells.append(
        f'<mxCell id="sys" value="COSMO AI-Native CRM" style="shape=rectangle;whiteSpace=wrap;html=1;'
        f'fillColor=none;strokeColor=#000000;strokeWidth=2;verticalAlign=top;align=left;spacingLeft=10;'
        f'spacingTop=6;fontStyle=1;fontSize=15;" vertex="1" parent="1">'
        f'<mxGeometry x="{sys_x}" y="{sys_y}" width="{sys_w}" height="{sys_h}" as="geometry"/></mxCell>')
    for i, (label, r0, r1, x0, x1) in enumerate(GROUPS):
        gx, gy = x0 - PAD, y(r0) - PAD - 14
        # Prospect's line enters the pipeline box at its top-left corner.
        align = "right;spacingRight=6" if x0 == R1 else "left;spacingLeft=6"
        cells.append(
            f'<mxCell id="grp{i}" value="{esc(label)}" style="shape=rectangle;whiteSpace=wrap;html=1;'
            f'fillColor=none;dashed=1;strokeColor=#999999;verticalAlign=top;align={align};'
            f'spacingTop=0;fontSize=11;fontStyle=2;fontColor=#666666;" vertex="1" parent="1">'
            f'<mxGeometry x="{gx}" y="{gy}" width="{x1 - x0 + 2 * PAD}" height="{y(r1) + H + PAD - gy}" '
            f'as="geometry"/></mxCell>')
    for k, (label, x, r) in USE_CASES.items():
        cells.append(
            f'<mxCell id="uc_{k}" value="{esc(label)}" style="ellipse;whiteSpace=wrap;html=1;'
            f'fillColor=#FFFFFF;strokeColor=#000000;strokeWidth=1.5;fontSize=12;" vertex="1" parent="1">'
            f'<mxGeometry x="{x}" y="{y(r)}" width="{W}" height="{H}" as="geometry"/></mxCell>')
    for k, (label, x, yy, kind) in ACTORS.items():
        if kind == "actor":
            style = ("shape=umlActor;verticalLabelPosition=bottom;verticalAlign=top;html=1;"
                     "fillColor=#000000;strokeColor=#000000;fontStyle=1;fontSize=12;")
            if k in ("bd", "admin"):
                # Label beside the figure: the Admin -> BD arrow runs below BD,
                # where a bottom label would sit under the line.
                style = style.replace("verticalLabelPosition=bottom;verticalAlign=top;",
                                      "labelPosition=left;verticalLabelPosition=middle;align=right;verticalAlign=middle;")
            w, h = 40, 60
        else:
            style = ("shape=rectangle;whiteSpace=wrap;html=1;fillColor=#FFFFFF;strokeColor=#000000;"
                     "strokeWidth=1.5;fontStyle=1;fontSize=12;")
            w, h = 120, 70
        cells.append(
            f'<mxCell id="a_{k}" value="{esc(label)}" style="{style}" vertex="1" parent="1">'
            f'<mxGeometry x="{x}" y="{yy}" width="{w}" height="{h}" as="geometry"/></mxCell>')

    def cid(k: str) -> str:
        return f"uc_{k}" if k in USE_CASES else f"a_{k}"

    base = "edgeStyle=none;html=1;strokeColor=#000000;strokeWidth=1;fontSize=10;labelBackgroundColor=#FFFFFF;"
    style_of = {
        ASSOC: base + "endArrow=none;fontStyle=2;",
        INCLUDE: base + "endArrow=open;endSize=10;dashed=1;dashPattern=8 4;",
        EXTEND: base + "endArrow=open;endSize=10;dashed=1;dashPattern=8 4;",
        GENERAL: base + "endArrow=block;endFill=0;endSize=14;",
    }
    for i, (s, t, kind, label) in enumerate(EDGES):
        cells.append(
            f'<mxCell id="e{i}" value="{esc(label)}" style="{style_of[kind]}" edge="1" parent="1" '
            f'source="{cid(s)}" target="{cid(t)}"><mxGeometry relative="1" as="geometry"/></mxCell>')

    note_y = y(16.8) + H + 60
    cells.append(
        f'<mxCell id="note" value="{esc("Pipeline steps run on their own (worker + Pub/Sub).&#xa;BD role = supervisor, not operator.".replace("&#xa;", chr(10)))}" '
        f'style="shape=note;whiteSpace=wrap;html=1;size=14;fillColor=#FFFFFF;strokeColor=#000000;'
        f'align=left;spacingLeft=8;fontSize=11;" vertex="1" parent="1">'
        f'<mxGeometry x="{COL2}" y="{y(17.5)}" width="{R2 + W - COL2}" height="50" as="geometry"/></mxCell>')
    del note_y
    return ('<mxfile host="app.diagrams.net"><diagram id="overall" name="Overall Use Case">'
            '<mxGraphModel dx="1600" dy="1200" grid="0" gridSize="10" guides="1" tooltips="1" '
            'connect="1" arrows="1" fold="1" page="0" pageScale="1" pageWidth="1700" pageHeight="1900" '
            'math="0" shadow="0"><root>' + "".join(cells) + "</root></mxGraphModel></diagram></mxfile>")


# ---- checks: straight centre-to-centre lines, as edgeStyle=none draws them ----

def clip(b, toward):
    """Point where the line from b's centre toward `toward` leaves b's outline."""
    x, yy, w, h, kind = b
    cx, cy = x + w / 2, yy + h / 2
    dx, dy = toward[0] - cx, toward[1] - cy
    if kind == "uc":
        t = 1 / math.sqrt((dx / (w / 2)) ** 2 + (dy / (h / 2)) ** 2)
    else:
        t = min(abs((w / 2) / dx) if dx else math.inf, abs((h / 2) / dy) if dy else math.inf)
    return cx + dx * t, cy + dy * t


def segments():
    bx = boxes()
    out = []
    for s, t, _, _ in EDGES:
        a, b = bx[s], bx[t]
        ca = (a[0] + a[2] / 2, a[1] + a[3] / 2)
        cb = (b[0] + b[2] / 2, b[1] + b[3] / 2)
        out.append(((s, t), clip(a, cb), clip(b, ca)))
    return out


def cross(p1, p2, p3, p4) -> bool:
    def o(a, b, c):
        return (b[0] - a[0]) * (c[1] - a[1]) - (b[1] - a[1]) * (c[0] - a[0])
    d1, d2, d3, d4 = o(p3, p4, p1), o(p3, p4, p2), o(p1, p2, p3), o(p1, p2, p4)
    return (d1 > 0) != (d2 > 0) and (d3 > 0) != (d4 > 0)


def hits(p, q, b, margin=8.0) -> bool:
    """Does the segment pass within `margin` of box b (ellipse for use cases)?"""
    x, yy, w, h, kind = b
    cx, cy = x + w / 2, yy + h / 2
    for i in range(201):
        px = p[0] + (q[0] - p[0]) * i / 200
        py = p[1] + (q[1] - p[1]) * i / 200
        if kind == "uc":
            if ((px - cx) / (w / 2 + margin)) ** 2 + ((py - cy) / (h / 2 + margin)) ** 2 < 1:
                return True
        elif x - margin < px < x + w + margin and yy - margin < py < yy + h + margin:
            return True
    return False


def check() -> list[str]:
    segs = segments()
    bx = boxes()
    problems = []
    for (e1, a1, b1), (e2, a2, b2) in itertools.combinations(segs, 2):
        if set(e1) & set(e2):
            continue  # edges sharing a node meet there by design
        if cross(a1, b1, a2, b2):
            problems.append(f"crossing: {e1} x {e2}")
    for e, a, b in segs:
        for k, box in bx.items():
            if k not in e and hits(a, b, box):
                problems.append(f"through box: {e} over {k}")
    keys = list(bx)
    for k1, k2 in itertools.combinations(keys, 2):
        a, b = bx[k1], bx[k2]
        if a[0] < b[0] + b[2] and b[0] < a[0] + a[2] and a[1] < b[1] + b[3] and b[1] < a[1] + a[3]:
            problems.append(f"overlap: {k1} x {k2}")
    return problems


if __name__ == "__main__":
    out = pathlib.Path(__file__).with_name("overall-usecase.drawio")
    out.write_text(build(), encoding="utf-8")
    problems = check()
    for p in problems:
        print(p)
    print(f"{out.name}: {len(EDGES)} edges, {len(problems)} problems")
    sys.exit(1 if problems else 0)
