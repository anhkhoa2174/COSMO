#!/usr/bin/env python3
"""Untangle a large use case .drawio that fix_overlaps.py cannot finish.

Two things make the big diagram different from the small ones.

First, the metric. fix_overlaps.solve() minimises crossings and box
cut-throughs *before* the connectors are bent, then route_around() bends them,
then final_check() decides whether the file is acceptable. On a small diagram
those agree closely enough. Here nearly every actor-to-use-case connector
starts out cutting through the boxes between them, so the pre-bend count is
dominated by hits that bending removes anyway, and the seed that looks best
before bending is not the one that ends up cleanest after it. This script
scores each seed the way the acceptance test does: bend first, then count.

Second, the cost. route_around() tries all ordered pairs of the 104 candidate
bend points for every bad edge — 10,816 routes — and re-evaluates the whole
diagram for each, which is roughly 65 million operations per edge. It never
returned. The router here changes both halves of that:

  * Delta evaluation. Moving one edge changes only that edge's box hits and
    its crossings with the others, so a trial costs one edge against the rest
    instead of every edge against every other.
  * A pruned two-bend set. A connector that has to cross the grid wants to
    leave its row, run along one free lane, and come back — so the useful
    two-bend routes are the ~20 that pair a lane near the source with the
    matching lane near the target, not all 10,816 combinations.

Usage:
    python3 fix_big.py file.drawio [--seconds 3600] [--accept 0] [--out PATH]

Writes only when crossings + cut-throughs is at most --accept (default 0).
The source file is untouched unless --out points at it.
"""

from __future__ import annotations

import argparse
import sys
import time

import fix_overlaps as F


# --------------------------------------------------------------- geometry

def live_boxes(boxes, placement):
    out = {}
    for cid, b in boxes.items():
        nb = dict(b)
        if cid in placement:
            nb["x"], nb["y"] = placement[cid]
        out[cid] = nb
    return out


def chain(live, s, t, pts):
    """The polyline for one edge, as a list of segments."""
    a, b = live[s], live[t]
    ca = (a["x"] + a["w"] / 2, a["y"] + a["h"] / 2)
    cb = (b["x"] + b["w"] / 2, b["y"] + b["h"] / 2)
    nodes = [F.border_point(a, pts[0] if pts else cb)] + list(pts) + \
            [F.border_point(b, pts[-1] if pts else ca)]
    return [F.seg(p, q) for p, q in zip(nodes, nodes[1:])]


def box_hits(live, s, t, segs):
    """How many boxes this edge cuts through, ignoring its own endpoints.

    The system boundary is excluded: an actor drawn outside it has to cross it
    to reach anything, so counting that would make every diagram unfixable.
    """
    n = 0
    for g in segs:
        for cid, b in live.items():
            if cid in (s, t) or cid == "sys" or b["w"] < 5:
                continue
            if F.hits(g, b):
                n += 1
    return n


def pair_crossings(segs_a, segs_b):
    return sum(1 for g in segs_a for h in segs_b if F.cross(g, h))


# ------------------------------------------------------------------ lanes

def lanes(boxes):
    """The empty corridors between the rows and columns of use cases."""
    ucs = [b for c, b in boxes.items() if F.is_usecase(c, b)]
    if not ucs:
        return [], []
    xs = sorted({b["x"] for b in ucs})
    ys = sorted({b["y"] for b in ucs})
    w = max(b["w"] for b in ucs)
    h = max(b["h"] for b in ucs)
    lx = [xs[0] - 45] + [(a + w + b) / 2 for a, b in zip(xs, xs[1:])] + [xs[-1] + w + 45]
    ly = [ys[0] - 40] + [(a + h + b) / 2 for a, b in zip(ys, ys[1:])] + [ys[-1] + h + 40]
    return lx, ly


def routes_for(live, s, t, lx, ly):
    """Candidate bend routes for one edge, cheapest shape first.

    One bend keeps the connector simple and is tried first. The two-bend set is
    deliberately small: pair a lane beside the source with the same lane beside
    the target, which is the shape that takes an edge out of the grid, along a
    free corridor, and back in. Arbitrary pairs of lane points mostly produce
    zig-zags that are no better and cost a thousand times more to search.
    """
    a, b = live[s], live[t]
    ax, ay = a["x"] + a["w"] / 2, a["y"] + a["h"] / 2
    bx, by = b["x"] + b["w"] / 2, b["y"] + b["h"] / 2

    # Every route below is axis-aligned. Diagonals were the whole problem: a
    # straight hop from a box to a corridor cuts across whatever sits between
    # them, so a "detour" that reaches the corridor diagonally clears nothing.
    # Leaving a box along its own centre line and turning only inside a
    # corridor is what actually gets an edge across a dense grid.

    # L-shapes: one turn, the cheapest thing that can work.
    out = [[(ax, by)], [(bx, ay)]]

    # Down a column gap: out sideways at the source's height, along the gap,
    # in sideways at the target's height.
    out += [[(x, ay), (x, by)] for x in lx]

    # Along a row gap: out vertically at the source's column, across the gap,
    # back in vertically at the target's column.
    out += [[(ax, y), (bx, y)] for y in ly]

    # Staircases. Two turns are not enough on a grid this dense: the run that
    # finally drops into the target column lands inside it and clips whatever
    # shares that column. Three turns let the edge sit in a row gap, move to a
    # column gap, descend it, and only then enter the target sideways.
    out += [[(ax, y), (x, y), (x, by)] for y in ly for x in lx]
    out += [[(x, ay), (x, y), (bx, y)] for x in lx for y in ly]

    # Last resort, still axis-aligned: any single corridor intersection.
    out += [[(x, y)] for x in lx for y in ly]
    return out


# ----------------------------------------------------------------- router

def route(boxes, edges, placement, budget_s: float):
    """Bend the edges that cut through boxes or cross each other.

    Returns (bends, crossings, cut_throughs).
    """
    live = live_boxes(boxes, placement)
    lx, ly = lanes(boxes)
    ends = {eid: (s, t) for eid, s, t in edges}

    bends: dict[str, list] = {}
    segs = {eid: chain(live, s, t, []) for eid, s, t in edges}
    hits_of = {eid: box_hits(live, *ends[eid], segs[eid]) for eid in segs}

    ids = list(segs)
    xross = {}
    for i, e1 in enumerate(ids):
        for e2 in ids[i + 1:]:
            n = pair_crossings(segs[e1], segs[e2])
            if n:
                xross[(e1, e2) if e1 < e2 else (e2, e1)] = n

    def cost_of(eid):
        """This edge's share: its box hits plus its crossings with the rest."""
        c = hits_of[eid]
        for (a, b), n in xross.items():
            if a == eid or b == eid:
                c += n
        return c

    deadline = time.time() + budget_s

    for _ in range(4):
        bad = [e for e in ids if cost_of(e) > 0]
        if not bad:
            break
        changed = False
        for eid in sorted(bad, key=cost_of, reverse=True):
            if time.time() > deadline:
                return bends, *tally(ids, hits_of, xross)
            s, t = ends[eid]
            base = cost_of(eid)
            best, best_cost = None, base

            for pts in routes_for(live, s, t, lx, ly):
                cand = chain(live, s, t, pts)
                c = box_hits(live, s, t, cand)
                if c >= best_cost:
                    continue  # already worse on boxes alone
                for oid in ids:
                    if oid == eid:
                        continue
                    c += pair_crossings(cand, segs[oid])
                    if c >= best_cost:
                        break
                if c < best_cost:
                    best, best_cost = (pts, cand), c
                    if c == 0:
                        break

            if best is not None:
                pts, cand = best
                bends[eid] = pts
                segs[eid] = cand
                hits_of[eid] = box_hits(live, s, t, cand)
                for oid in ids:
                    if oid == eid:
                        continue
                    key = (eid, oid) if eid < oid else (oid, eid)
                    xross.pop(key, None)
                    n = pair_crossings(cand, segs[oid])
                    if n:
                        xross[key] = n
                changed = True
        if not changed:
            break

    return bends, *tally(ids, hits_of, xross)


def tally(ids, hits_of, xross):
    return sum(xross.values()), sum(hits_of.values())


# ------------------------------------------------------------------ main

def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("path")
    ap.add_argument("--seconds", type=float, default=3600)
    ap.add_argument("--per-seed", type=float, default=90)
    ap.add_argument("--accept", type=int, default=0)
    ap.add_argument("--out", default=None)
    args = ap.parse_args()

    src, boxes, edges = F.load(args.path)
    before = F.score(boxes, edges, {})
    print(f"trước: cắt={before[0]} xuyên={before[1]}", flush=True)

    best = None
    t0 = time.time()
    seed = 0
    while time.time() - t0 < args.seconds:
        _, place = F.solve(boxes, edges, seed)
        bends, xings, through = route(boxes, edges, place, args.per_seed)
        # The acceptance test is fix_overlaps' own, not this router's bookkeeping.
        xings, through = F.final_check(boxes, edges, place, bends)
        total = xings + through

        if best is None or total < best[0]:
            best = (total, xings, through, dict(place), dict(bends), seed)
            print(f"  seed {seed:3d}: cắt={xings} xuyên={through} "
                  f"({len(bends)} đường bẻ góc)  <-- tốt nhất "
                  f"[{time.time() - t0:.0f}s]", flush=True)
        seed += 1
        if best[0] == 0:
            break

    if best is None:
        print("không chạy nổi một seed nào", file=sys.stderr)
        return 1

    total, xings, through, place, bends, seed = best
    print(f"\ntốt nhất: seed {seed}, cắt={xings} xuyên={through} "
          f"sau {seed + 1} seed / {time.time() - t0:.0f}s")

    if total > args.accept:
        print(f"KHÔNG ghi: {total} > ngưỡng {args.accept}. File gốc còn nguyên.")
        return 1

    out = args.out or args.path
    open(out, "w", encoding="utf-8").write(F.rewrite(src, place, bends))
    print(f"đã ghi {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
