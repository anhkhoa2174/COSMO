#!/usr/bin/env python3
"""Build the COSMO sitemap from the app's real routes and sidebar.

Layout: entry pages stacked at the top, the dashboard below them, and one
horizontal bus under the dashboard that drops into each sidebar section. Each
section is a box listing its pages; a sub-page hangs off its parent through a
short elbow in the section's left gutter. Nothing fans across the page, so no
line crosses a page box.

Writes sitemap.drawio and checks every edge segment against every page box.
"""

from __future__ import annotations

import html
import pathlib
import sys

COL_W, GAP, LEFT = 214, 22, 40
PAGE_W, PAGE_H, SUB_W, SUB_H = 184, 44, 164, 40
PAD, HEAD, STEP = 15, 30, 54          # container padding, header height, row pitch

DASH_Y, BUS_Y, SEC_Y = 300, 400, 430

COLORS = {
    "root": ("#dae8fc", "#6c8ebf"),
    "auth": ("#e1d5e7", "#9673a6"),
    "hub": ("#d5e8d4", "#82b366"),
    "main": ("#fff2cc", "#d6b656"),
    "sub": ("#fffdf5", "#d6b656"),
    "admin": ("#f8cecc", "#b85450"),
    "admin_sub": ("#fff7f6", "#b85450"),
    "public": ("#f5f5f5", "#666666"),
}

# Sidebar sections in sidebar order. Each item: (label, route, is_sub_page).
SECTIONS = [
    ("DAILY WORK", "main", [("Daily Actions", "/daily-actions", False),
                            ("Smart Insights", "/smart-insights", False)]),
    ("INBOX", "main", [("AI Inboxes", "/ai-inboxes", False),
                       ("Emails", "/emails", False)]),
    ("PIPELINE", "main", [("Outreach", "/outreach", False),
                          ("Meetings", "/meetings", False),
                          ("Tasks", "/tasks", False)]),
    ("PROSPECTS", "main", [("All Prospects", "/all-prospects", False),
                           ("CSV Import", "/all-prospects/csv-import", True),
                           ("HubSpot Import", "/hubspot-integration", True),
                           ("Audiences", "/audiences", False),
                           ("Profile Fields", "/profile-fields", False),
                           ("Lead Forms", "/lead-forms", False)]),
    ("CAMPAIGNS", "main", [("Campaigns", "/campaigns", False),
                           ("Campaign Detail", "/campaigns/[id]", True),
                           ("Playbook Strategy", "/campaigns/playbook", True),
                           ("Templates", "/templates", False)]),
    ("KNOWLEDGE", "main", [("Knowledge Base", "/libraries", False),
                           ("Files", "/files", False)]),
    ("SETTINGS", "admin", [("Email Agents", "/agents", False),
                           ("Agent Detail", "/agents/[id]", True),
                           ("Company Information", "/company-information", False),
                           ("Team Members", "/team-members", False),
                           ("Sales Reps", "/sales-reps", False),
                           ("Outreach Timing", "/outreach-timing", False),
                           ("API Keys (user menu, any user)", "/settings/api-keys", False)]),
]

TOTAL_W = len(SECTIONS) * COL_W + (len(SECTIONS) - 1) * GAP
CENTER = LEFT + TOTAL_W / 2


def col_x(i: int) -> float:
    return LEFT + i * (COL_W + GAP)


cells: list[str] = ['<mxCell id="0"/>', '<mxCell id="1" parent="0"/>']
boxes: dict[str, tuple[float, float, float, float]] = {}   # page boxes, for checks
containers: dict[str, tuple[float, float, float, float]] = {}
segments: list[tuple[str, tuple[float, float], tuple[float, float]]] = []


def esc(s: str) -> str:
    return html.escape(s, quote=True)


def label(title: str, route: str, bold: bool = True) -> str:
    t = f"<b>{esc(title)}</b>" if bold else esc(title)
    return f"{t}<br>{esc(route)}" if route else t


def box(cid, value, x, y, w, h, kind, font=11, extra=""):
    fill, stroke = COLORS[kind]
    cells.append(
        f'<mxCell id="{cid}" value="{esc(value)}" style="rounded=1;whiteSpace=wrap;html=1;'
        f'fillColor={fill};strokeColor={stroke};fontSize={font};arcSize=12;{extra}" vertex="1" parent="1">'
        f'<mxGeometry x="{x}" y="{y}" width="{w}" height="{h}" as="geometry"/></mxCell>')
    boxes[cid] = (x, y, w, h)


def text(cid, value, x, y, w, h, style):
    cells.append(
        f'<mxCell id="{cid}" value="{esc(value)}" style="text;html=1;{style}" vertex="1" parent="1">'
        f'<mxGeometry x="{x}" y="{y}" width="{w}" height="{h}" as="geometry"/></mxCell>')


def edge(eid, src, tgt, pts, dashed=False, entry=None, exit_=None):
    """Polyline edge through absolute points; first/last are the anchors."""
    (sx, sy), (tx, ty) = pts[0], pts[-1]

    def anchor(cid, px, py):
        x, y, w, h = boxes.get(cid) or containers[cid]
        return (px - x) / w, (py - y) / h

    ex, ey = anchor(src, sx, sy)
    nx, ny = anchor(tgt, tx, ty)
    mids = "".join(f'<mxPoint x="{x}" y="{y}"/>' for x, y in pts[1:-1])
    style = (f"edgeStyle=none;html=1;endArrow=block;endSize=6;strokeColor=#555555;strokeWidth=1.2;"
             f"exitX={ex:.4f};exitY={ey:.4f};exitDx=0;exitDy=0;entryX={nx:.4f};entryY={ny:.4f};entryDx=0;entryDy=0;"
             + ("dashed=1;" if dashed else ""))
    cells.append(
        f'<mxCell id="{eid}" style="{style}" edge="1" parent="1" source="{src}" target="{tgt}">'
        f'<mxGeometry relative="1" as="geometry"><mxPoint x="{sx}" y="{sy}" as="sourcePoint"/>'
        f'<mxPoint x="{tx}" y="{ty}" as="targetPoint"/>'
        + (f'<Array as="points">{mids}</Array>' if mids else "") + '</mxGeometry></mxCell>')
    for p, q in zip(pts, pts[1:]):
        segments.append(((src, tgt), p, q))


# ---- title and legend -------------------------------------------------------
text("title", "COSMO — Sitemap", CENTER - 200, 10, 400, 34,
     "fontSize=22;fontStyle=1;align=center;verticalAlign=middle;")

lx, ly = LEFT, 60
cells.append(f'<mxCell id="legend" value="" style="rounded=1;whiteSpace=wrap;html=1;fillColor=none;'
             f'dashed=1;strokeColor=#999999;" vertex="1" parent="1">'
             f'<mxGeometry x="{lx}" y="{ly}" width="430" height="128" as="geometry"/></mxCell>')
text("legend_t", "<b>Legend</b>", lx + 12, ly + 6, 100, 20, "fontSize=12;align=left;")
legend = [("root", "Landing / Root"), ("auth", "Auth / Onboarding"), ("hub", "Dashboard (Hub)"),
          ("main", "Main feature page"), ("sub", "Sub-page (opened from its parent)"),
          ("admin", "Settings / Organisation"), ("public", "Public (no login)")]
for i, (kind, name) in enumerate(legend):
    cx = lx + 14 + (i // 4) * 210
    cy = ly + 32 + (i % 4) * 23
    fill, stroke = COLORS[kind]
    cells.append(f'<mxCell id="lg{i}" value="" style="rounded=1;html=1;fillColor={fill};strokeColor={stroke};" '
                 f'vertex="1" parent="1"><mxGeometry x="{cx}" y="{cy}" width="22" height="15" as="geometry"/></mxCell>')
    text(f"lgt{i}", name, cx + 30, cy - 3, 180, 20, "fontSize=11;align=left;verticalAlign=middle;")

# ---- entry flow -------------------------------------------------------------
EW, EH = 180, 48
box("landing", label("Landing Page", "/"), CENTER - EW / 2, 60, EW, EH, "root", 12)
box("login", label("Login", "/auth/login"), CENTER - EW / 2, 160, EW, EH, "auth", 12)
box("onboarding", label("Onboarding", "/onboarding (first login)"), CENTER - EW / 2 - 290, 160, EW + 20, EH, "auth", 12)
box("callbacks", "<i>OAuth callbacks</i><br>/auth/callback/google · gmail · hubspot · invite · member",
    CENTER + EW / 2 + 90, 160, 300, EH, "public", 10)
box("dashboard", label("Dashboard", "/dashboard"), CENTER - 100, DASH_Y - 40, 200, 56, "hub", 14)

edge("e_land_login", "landing", "login", [(CENTER, 60 + EH), (CENTER, 160)])
edge("e_login_cb", "login", "callbacks", [(CENTER + EW / 2, 184), (CENTER + EW / 2 + 90, 184)], dashed=True)
edge("e_login_onb", "login", "onboarding", [(CENTER - EW / 2, 184), (CENTER - EW / 2 - 90, 184)])
edge("e_login_dash", "login", "dashboard", [(CENTER, 160 + EH), (CENTER, DASH_Y - 40)])
# Onboarding finishes at the dashboard: down its centre, then right into it.
ob_cx = CENTER - EW / 2 - 290 + (EW + 20) / 2
edge("e_onb_dash", "onboarding", "dashboard", [(ob_cx, 160 + EH), (ob_cx, DASH_Y - 12), (CENTER - 100, DASH_Y - 12)])

# ---- public pages (top right) ----------------------------------------------
px = col_x(len(SECTIONS) - 1)
public = [("Public Lead Form", "/public/forms/[slug]"), ("Terms of Service", "/tos"),
          ("Privacy Policy", "/policy"), ("Data Processing", "/data-processing")]
ph = HEAD + PAD + len(public) * (SUB_H + 10)
cells.append(f'<mxCell id="sec_public" value="" style="rounded=1;whiteSpace=wrap;html=1;fillColor=none;'
             f'dashed=1;strokeColor=#888888;arcSize=4;" vertex="1" parent="1">'
             f'<mxGeometry x="{px}" y="40" width="{COL_W}" height="{ph}" as="geometry"/></mxCell>')
containers["sec_public"] = (px, 40, COL_W, ph)
text("sec_public_t", "<b>PUBLIC (no login)</b>", px, 44, COL_W, 22, "fontSize=12;align=center;fontColor=#555555;")
for i, (t, r) in enumerate(public):
    box(f"pub{i}", label(t, r), px + (COL_W - PAGE_W) / 2, 40 + HEAD + i * (SUB_H + 10), PAGE_W, SUB_H, "public", 10)

# ---- sections ---------------------------------------------------------------
bus_left = col_x(0) + COL_W / 2
bus_right = col_x(len(SECTIONS) - 1) + COL_W / 2

max_bottom = 0.0
for i, (title, kind, items) in enumerate(SECTIONS):
    x = col_x(i)
    rows = len(items)
    h = HEAD + PAD + rows * STEP
    sid = f"sec{i}"
    stroke = COLORS[kind][1]
    cells.append(f'<mxCell id="{sid}" value="" style="rounded=1;whiteSpace=wrap;html=1;fillColor=none;'
                 f'dashed=1;strokeColor={stroke};arcSize=3;" vertex="1" parent="1">'
                 f'<mxGeometry x="{x}" y="{SEC_Y}" width="{COL_W}" height="{h}" as="geometry"/></mxCell>')
    containers[sid] = (x, SEC_Y, COL_W, h)
    text(f"{sid}_t", f"<b>{title}</b>", x, SEC_Y + 4, COL_W, 22, "fontSize=12;align=center;fontColor=#555555;")
    max_bottom = max(max_bottom, SEC_Y + h)

    parent_id = None
    for j, (t, r, is_sub) in enumerate(items):
        y = SEC_Y + HEAD + j * STEP
        pid = f"p{i}_{j}"
        if is_sub:
            bx = x + (COL_W - PAGE_W) / 2 + 24
            box(pid, label(t, r, bold=False), bx, y + 2, SUB_W - 4, SUB_H, "admin_sub" if kind == "admin" else "sub", 10)
            # elbow from the parent's bottom, down the gutter, into the sub-page
            gx = x + (COL_W - PAGE_W) / 2 + 12
            pbx, pby, pbw, pbh = boxes[parent_id]
            edge(f"e_{pid}", parent_id, pid, [(gx, pby + pbh), (gx, y + 2 + SUB_H / 2), (bx, y + 2 + SUB_H / 2)])
        else:
            # API keys sit in the user menu and are open to every user.
            page_kind = "main" if r == "/settings/api-keys" else kind
            box(pid, label(t, r), x + (COL_W - PAGE_W) / 2, y, PAGE_W, PAGE_H, page_kind, 11)
            parent_id = pid

    # drop from the bus into the section header; the middle section sits
    # right under the dashboard and is fed straight down
    cx = x + COL_W / 2
    if abs(cx - CENTER) < 1:
        edge(f"e_bus_{sid}", "dashboard", sid, [(CENTER, DASH_Y + 16), (CENTER, SEC_Y)])
    else:
        edge(f"e_bus_{sid}", "dashboard", sid, [(CENTER, DASH_Y + 16), (CENTER, BUS_Y), (cx, BUS_Y), (cx, SEC_Y)])

# ---- role-gated note ----------------------------------------------------------
note_y = max_bottom - 118
note = ("<b>Role-gated UI (admin only)</b><br>"
        "• Team Members: delete other members<br>"
        "• All Prospects: “Added By” column<br>"
        "• AI Inboxes: Assigned to AI / Sent tabs<br>"
        "• Conversation header: assign &amp; delete<br>"
        "• Dashboard: Team view")
cells.append(f'<mxCell id="note" value="{esc(note)}" style="shape=note;whiteSpace=wrap;html=1;size=14;'
             f'fillColor=#fff2cc;strokeColor=#d6b656;align=left;spacingLeft=10;fontSize=11;verticalAlign=top;spacingTop=6;" '
             f'vertex="1" parent="1"><mxGeometry x="{col_x(0)}" y="{note_y}" width="{2 * COL_W + GAP}" height="118" as="geometry"/></mxCell>')
boxes["note"] = (col_x(0), note_y, 2 * COL_W + GAP, 118)


def through(p, q, b, pad=2.0) -> bool:
    x, y, w, h = b
    for k in range(1, 200):
        px_ = p[0] + (q[0] - p[0]) * k / 200
        py_ = p[1] + (q[1] - p[1]) * k / 200
        if x + pad < px_ < x + w - pad and y + pad < py_ < y + h - pad:
            return True
    return False


def overlap(a, b) -> bool:
    return a[0] < b[0] + b[2] and b[0] < a[0] + a[2] and a[1] < b[1] + b[3] and b[1] < a[1] + a[3]


def check() -> list[str]:
    problems = []
    for (s, t), p, q in segments:
        for k, b in boxes.items():
            if k not in (s, t) and through(p, q, b):
                problems.append(f"edge {s}->{t} runs through {k}")
        # Elbows between pages stay inside their own section; every other
        # line must keep out of sections it does not lead into.
        if s.startswith("p") and t.startswith("p"):
            continue
        for k, b in containers.items():
            if k != t and through(p, q, b, pad=-1):
                problems.append(f"edge {s}->{t} crosses section {k}")
    keys = list(boxes)
    for a in range(len(keys)):
        for b in range(a + 1, len(keys)):
            if overlap(boxes[keys[a]], boxes[keys[b]]):
                problems.append(f"overlap {keys[a]} x {keys[b]}")
    return problems


if __name__ == "__main__":
    out = pathlib.Path(__file__).with_name("sitemap.drawio")
    out.write_text('<mxfile host="app.diagrams.net"><diagram id="sitemap" name="Sitemap">'
                   '<mxGraphModel grid="0" page="0" math="0" shadow="0"><root>' + "".join(cells) +
                   "</root></mxGraphModel></diagram></mxfile>", encoding="utf-8")
    problems = check()
    print("\n".join(problems))
    print(f"{out.name}: {len(boxes)} boxes, {len(segments)} segments, {len(problems)} problems")
    sys.exit(1 if problems else 0)
