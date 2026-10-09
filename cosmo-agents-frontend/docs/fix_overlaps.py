#!/usr/bin/env python3
"""Remove crossing edges from a use case .drawio without changing its layout.

The diagrams already have the shape they should have: a system boundary, the
actors outside it, and the use cases on a grid inside. Only the assignment of
use cases to grid slots is wrong, which is what makes the connectors cross and
cut through boxes.

So nothing moves. The grid slots, the actors and the boundary stay exactly
where the author put them; the script only decides which use case sits in
which slot, searching for an assignment with no crossings, and rewrites the
file with the geometry swapped.

Usage:  python3 fix_overlaps.py file.drawio [...]
"""

from __future__ import annotations

import itertools
import math
import random
import re
import sys

CELL_RE = re.compile(
    r'(<mxCell id="(?P<id>[^"]+)"(?P<attrs>[^>]*?)vertex="1"[^>]*>\s*'
    r'<mxGeometry )(?P<geo>[^/]*?)(/>)',
    re.S,
)


def attr(blob: str, name: str, default: str = "") -> str:
    m = re.search(rf'\b{name}="([^"]*)"', blob)
    return m.group(1) if m else default


def load(path: str):
    src = open(path, encoding="utf-8").read()

    boxes: dict[str, dict] = {}
    for m in CELL_RE.finditer(src):
        cid = m.group("id")
        geo = m.group("geo")
        boxes[cid] = {
            "x": float(attr(geo, "x", "0")), "y": float(attr(geo, "y", "0")),
            "w": float(attr(geo, "width", "0")), "h": float(attr(geo, "height", "0")),
            "style": attr(m.group("attrs"), "style"),
            "label": re.sub(r"<[^>]+>", " ", attr(m.group("attrs"), "value")).strip()[:30],
        }

    edges = []
    for m in re.finditer(r'<mxCell id="([^"]+)"[^>]*?edge="1"[^>]*?>', src):
        blob = src[m.start():m.start() + 700]
        s, t = attr(blob, "source"), attr(blob, "target")
        if s and t:
            edges.append((m.group(1), s, t))

    return src, boxes, edges


def outside_boundary(cid: str, b: dict, boxes: dict) -> bool:
    """Anything drawn beside the system boundary rather than inside it.

    Actors are drawn as stick figures, but the «system» participants are
    rounded boxes parked in the same margin. Both sit outside the boundary, so
    both may slide up and down their own side without changing what the
    diagram says.
    """
    sys_box = boxes.get("sys")
    if sys_box is None or cid == "sys":
        return False
    if "umlActor" in b["style"]:
        return True
    left, right = sys_box["x"], sys_box["x"] + sys_box["w"]
    return b["x"] + b["w"] <= left + 1 or b["x"] >= right - 1


def is_usecase(cid: str, b: dict) -> bool:
    """Only the ellipses inside the boundary get shuffled."""
    return (
        "umlActor" not in b["style"]
        and cid != "sys"
        and "ellipse" in b["style"]
        and b["w"] > 5
    )


# ---------------------------------------------------------------- geometry


def seg(p, q):
    return p[0], p[1], q[0], q[1]


def orient(ax, ay, bx, by, cx, cy):
    return (bx - ax) * (cy - ay) - (by - ay) * (cx - ax)


def cross(s1, s2) -> bool:
    ax, ay, bx, by = s1
    cx, cy, dx, dy = s2
    if {(ax, ay), (bx, by)} & {(cx, cy), (dx, dy)}:
        return False
    d1 = orient(cx, cy, dx, dy, ax, ay)
    d2 = orient(cx, cy, dx, dy, bx, by)
    d3 = orient(ax, ay, bx, by, cx, cy)
    d4 = orient(ax, ay, bx, by, dx, dy)
    return ((d1 > 0) != (d2 > 0)) and ((d3 > 0) != (d4 > 0))


def border_point(b: dict, towards: tuple[float, float]) -> tuple[float, float]:
    """Where a line from the box centre towards `towards` leaves the ellipse.

    Anchoring on the border rather than the centre keeps the segment out of
    its own box, so a near-vertical connector cannot clip the neighbour
    directly above or below on its way out.
    """
    cx, cy = b["x"] + b["w"] / 2, b["y"] + b["h"] / 2
    dx, dy = towards[0] - cx, towards[1] - cy
    if dx == 0 and dy == 0:
        return cx, cy
    rx, ry = b["w"] / 2, b["h"] / 2
    if "ellipse" in b["style"]:
        k = math.hypot(dx / rx, dy / ry)
        return cx + dx / k, cy + dy / k
    scale = min(rx / abs(dx) if dx else math.inf, ry / abs(dy) if dy else math.inf)
    return cx + dx * scale, cy + dy * scale


def hits(s, b: dict, pad: float = 3.0) -> bool:
    x1, y1 = b["x"] - pad, b["y"] - pad
    x2, y2 = b["x"] + b["w"] + pad, b["y"] + b["h"] + pad
    ax, ay, bx, by = s
    if x1 < ax < x2 and y1 < ay < y2:
        return True
    if x1 < bx < x2 and y1 < by < y2:
        return True
    sides = [(x1, y1, x2, y1), (x2, y1, x2, y2), (x2, y2, x1, y2), (x1, y2, x1, y1)]
    return any(cross(s, side) for side in sides)


def score(boxes, edges, placement: dict[str, tuple[float, float]]) -> tuple[int, int, float]:
    """(crossings, boxes cut through, total edge length) — lower is better."""
    live = {}
    for cid, b in boxes.items():
        nb = dict(b)
        if cid in placement:
            nb["x"], nb["y"] = placement[cid]
        live[cid] = nb

    segs = []
    for eid, s, t in edges:
        if s not in live or t not in live:
            continue
        a, b = live[s], live[t]
        ca = (a["x"] + a["w"] / 2, a["y"] + a["h"] / 2)
        cb = (b["x"] + b["w"] / 2, b["y"] + b["h"] / 2)
        segs.append((s, t, seg(border_point(a, cb), border_point(b, ca))))

    xings = sum(
        1 for (s1, t1, g1), (s2, t2, g2) in itertools.combinations(segs, 2)
        if cross(g1, g2)
    )
    through = 0
    length = 0.0
    for s, t, g in segs:
        length += math.hypot(g[2] - g[0], g[3] - g[1])
        for cid, b in live.items():
            if cid in (s, t) or cid == "sys" or b["w"] < 5:
                continue
            if hits(g, b):
                through += 1
    return xings, through, length


# ---------------------------------------------------------------- search


def solve(boxes, edges, seed: int = 0):
    """Hill climbing over two moves that both preserve the diagram's shape.

    Use cases swap between the slots the author already drew, and actors slide
    up or down their own side. An actor's column is what places it inside or
    outside the boundary, so keeping x fixed keeps the structure intact while
    its height becomes another degree of freedom for untangling connectors.
    """
    rng = random.Random(seed)
    ucs = [c for c, b in boxes.items() if is_usecase(c, b)]
    actors = [c for c, b in boxes.items() if outside_boundary(c, b, boxes)]
    slots = [(boxes[c]["x"], boxes[c]["y"]) for c in ucs]

    ys = [b["y"] for c, b in boxes.items() if is_usecase(c, b)]
    lo, hi = (min(ys) - 40, max(ys) + 40) if ys else (0, 600)

    # Seed by pull: a use case wired to the right margin starts in a right-hand
    # slot, one wired to the left actor starts on the left. Random starts leave
    # the long margin connectors to be untangled by luck, and they are exactly
    # the ones that stay broken.
    margin_x = {c: boxes[c]["x"] for c in actors}
    mid_x = sum(x for _, x in [(c, boxes[c]["x"]) for c in ucs]) / max(len(ucs), 1)

    def pull(c: str) -> float:
        vals = []
        for _, s_, t_ in edges:
            other = t_ if s_ == c else (s_ if t_ == c else None)
            if other in margin_x:
                vals.append(margin_x[other])
        return sum(vals) / len(vals) if vals else mid_x

    order = sorted(ucs, key=pull)
    slots_sorted = sorted(slots, key=lambda p: (p[0], p[1]))
    place = {c: slots_sorted[i] for i, c in enumerate(order)}
    if seed:
        rng.shuffle(order)
        place = {c: slots[i] for i, c in enumerate(order)}
    for c in actors:
        place[c] = (boxes[c]["x"], boxes[c]["y"])
    best = score(boxes, edges, place)
    best_place = dict(place)
    cur = best

    stale = 0
    step = 0
    budget = 6000
    while stale < 600 and step < budget:
        if actors and rng.random() < 0.35:
            c = rng.choice(actors)
            old_pos = place[c]
            place[c] = (old_pos[0], round(rng.uniform(lo, hi) / 10) * 10)
            undo = lambda: place.__setitem__(c, old_pos)
        else:
            a, b = rng.sample(ucs, 2)
            place[a], place[b] = place[b], place[a]
            undo = lambda: place.__setitem__(a, place[b]) or place.__setitem__(b, place[a])
            saved = (place[a], place[b])
            undo = lambda s=saved, a=a, b=b: (place.__setitem__(a, s[1]),
                                              place.__setitem__(b, s[0]))

        cand = score(boxes, edges, place)
        # Simulated annealing rather than strict hill climbing: a swap that
        # looks worse is sometimes kept while the temperature is high, which
        # is the only way out of the arrangements where every single swap
        # makes things worse but the layout still has an overlap.
        cost_new = cand[0] * 100 + cand[1] * 100 + cand[2] * 0.001
        cost_cur = cur[0] * 100 + cur[1] * 100 + cur[2] * 0.001
        temp = max(0.01, 6.0 * (1 - step / budget))
        if cost_new <= cost_cur or rng.random() < math.exp((cost_cur - cost_new) / temp):
            cur = cand
            if cand < best:
                best, best_place = cand, dict(place)
                stale = 0
        else:
            undo()
        stale += 1
        step += 1
        if best[0] == 0 and best[1] == 0:
            break
    return best, best_place


def gap_points(boxes) -> list[tuple[float, float]]:
    """Candidate bend points: the empty lanes between rows and columns.

    A connector from an actor pinned outside the boundary to a use case in the
    far column has to pass the middle column somehow. Bending it into a lane
    between boxes routes it around them without moving anything.
    """
    ucs = [b for c, b in boxes.items() if is_usecase(c, b)]
    if not ucs:
        return []

    xs = sorted({b["x"] for b in ucs})
    ys = sorted({b["y"] for b in ucs})
    w = max(b["w"] for b in ucs)
    h = max(b["h"] for b in ucs)

    lanes_x = [xs[0] - 45] + [(a + w + b) / 2 for a, b in zip(xs, xs[1:])] + [xs[-1] + w + 45]
    lanes_y = [ys[0] - 40] + [(a + h + b) / 2 for a, b in zip(ys, ys[1:])] + [ys[-1] + h + 40]
    pts = [(x, y) for x in lanes_x for y in lanes_y]

    # Lanes just outside the boundary as well. A connector from a «system» box
    # on the right to a use case in the far-left column has no clear path
    # straight across; taking it around the top or the bottom of the boundary
    # is the usual hand-drawn answer.
    sys_box = boxes.get("sys")
    if sys_box:
        top = sys_box["y"] - 25
        bottom = sys_box["y"] + sys_box["h"] + 25
        span = [sys_box["x"] - 25] + lanes_x + [sys_box["x"] + sys_box["w"] + 25]
        pts += [(x, y) for x in span for y in (top, bottom)]
    return pts


def route_around(boxes, edges, placement, bends: dict[str, list]) -> dict[str, list]:
    """Give a bend to every edge that still cuts through a box."""
    live = {}
    for cid, b in boxes.items():
        nb = dict(b)
        if cid in placement:
            nb["x"], nb["y"] = placement[cid]
        live[cid] = nb

    cands = gap_points(boxes)
    # Try the outer lanes first: the edges that survive slot swapping are the
    # long ones from the margin, and those are the routes that clear them.
    sys_box = boxes.get("sys")
    if sys_box:
        mid = sys_box["y"] + sys_box["h"] / 2
        cands.sort(key=lambda p: -abs(p[1] - mid))

    def segments(extra: dict[str, list]):
        out = []
        for eid, s, t in edges:
            if s not in live or t not in live:
                continue
            a, b = live[s], live[t]
            ca = (a["x"] + a["w"] / 2, a["y"] + a["h"] / 2)
            cb = (b["x"] + b["w"] / 2, b["y"] + b["h"] / 2)
            pts = extra.get(eid, [])
            chain = [border_point(a, pts[0] if pts else cb)] + pts + \
                    [border_point(b, pts[-1] if pts else ca)]
            for p, q in zip(chain, chain[1:]):
                out.append((eid, s, t, seg(p, q)))
        return out

    def problems(extra):
        segs = segments(extra)
        bad = set()
        for (e1, s1, t1, g1), (e2, s2, t2, g2) in itertools.combinations(segs, 2):
            if e1 == e2:
                continue
            if cross(g1, g2):
                bad.add(e1)
                bad.add(e2)
        for eid, s, t, g in segs:
            for cid, b in live.items():
                if cid in (s, t) or cid == "sys" or b["w"] < 5:
                    continue
                if hits(g, b):
                    bad.add(eid)
        return bad

    extra = dict(bends)
    for _ in range(6):
        bad = problems(extra)
        if not bad:
            break
        current = len(problems(extra))
        for eid in sorted(bad):
            trial = dict(extra)
            best_route, best_bad = None, current

            # One bend first — it keeps the connector simple.
            for pt in cands:
                trial[eid] = [pt]
                n = len(problems(trial))
                if n < best_bad:
                    best_route, best_bad = [pt], n
                if n == 0:
                    break

            # A single bend often trades a box overlap for a new crossing.
            # Two bends let the edge leave the grid, run along the outside and
            # come back in, which clears both.
            if best_bad:
                for p1, p2 in itertools.permutations(cands, 2):
                    trial[eid] = [p1, p2]
                    n = len(problems(trial))
                    if n < best_bad:
                        best_route, best_bad = [p1, p2], n
                    if n == 0:
                        break

            if best_route is not None and best_bad < current:
                extra[eid] = best_route
                current = best_bad
    return extra


def rewrite(src: str, placement: dict[str, tuple[float, float]],
            bends: dict[str, list] | None = None) -> str:
    def sub(m):
        cid = m.group("id")
        if cid not in placement:
            return m.group(0)
        x, y = placement[cid]
        geo = re.sub(r'\bx="[^"]*"', f'x="{x:g}"', m.group("geo"))
        geo = re.sub(r'\by="[^"]*"', f'y="{y:g}"', geo)
        return m.group(1) + geo + m.group(5)

    out = CELL_RE.sub(sub, src)

    for eid, pts in (bends or {}).items():
        if not pts:
            continue
        arr = "".join(f'<mxPoint x="{x:g}" y="{y:g}" />' for x, y in pts)
        pat = re.compile(
            rf'(<mxCell id="{re.escape(eid)}"[^>]*?edge="1"[^>]*?>\s*)'
            rf'<mxGeometry([^/>]*?)(/>|>\s*</mxGeometry>)', re.S)
        out = pat.sub(
            lambda m: m.group(1) + "<mxGeometry" + m.group(2)
            + f'><Array as="points">{arr}</Array></mxGeometry>', out, count=1)
    return out


def process(path: str) -> bool:
    src, boxes, edges = load(path)

    best, place = None, None
    for seed in range(40):
        s, p = solve(boxes, edges, seed)
        if best is None or s < best:
            best, place = s, p
        if best[0] == 0 and best[1] == 0:
            break

    before = score(boxes, edges, {})
    bends = route_around(boxes, edges, place, {})
    xings, through = final_check(boxes, edges, place, bends)

    ok = xings == 0 and through == 0
    print(f"  {'OK ' if ok else 'XX '}{path:40s} "
          f"trước: cắt={before[0]} xuyên={before[1]}  ->  "
          f"sau: cắt={xings} xuyên={through}, {len(bends)} đường phải bẻ góc")

    if ok:
        open(path, "w", encoding="utf-8").write(rewrite(src, place, bends))
    return ok


def final_check(boxes, edges, placement, bends) -> tuple[int, int]:
    live = {}
    for cid, b in boxes.items():
        nb = dict(b)
        if cid in placement:
            nb["x"], nb["y"] = placement[cid]
        live[cid] = nb

    segs = []
    for eid, s, t in edges:
        if s not in live or t not in live:
            continue
        a, b = live[s], live[t]
        ca = (a["x"] + a["w"] / 2, a["y"] + a["h"] / 2)
        cb = (b["x"] + b["w"] / 2, b["y"] + b["h"] / 2)
        pts = bends.get(eid, [])
        chain = [border_point(a, pts[0] if pts else cb)] + pts + \
                [border_point(b, pts[-1] if pts else ca)]
        for p, q in zip(chain, chain[1:]):
            segs.append((eid, s, t, seg(p, q)))

    xings = sum(
        1 for (e1, _, _, g1), (e2, _, _, g2) in itertools.combinations(segs, 2)
        if e1 != e2 and cross(g1, g2)
    )
    through = 0
    for eid, s, t, g in segs:
        for cid, b in live.items():
            if cid in (s, t) or cid == "sys" or b["w"] < 5:
                continue
            if hits(g, b):
                through += 1
    return xings, through


if __name__ == "__main__":
    sys.exit(0 if all(process(p) for p in sys.argv[1:]) else 1)
