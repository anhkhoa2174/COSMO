"""Remove crossing edges from a use case .drawio by re-slotting, not redrawing.

The constraint this tool is built around: the notation must survive untouched.
Every ellipse keeps its style and size, every actor keeps its shape, every
association keeps its arrow, its dash pattern and its «include»/«extend» label,
and every dashed group frame keeps its caption. Nothing is added and nothing is
deleted.

What it is allowed to change is only which grid slot a use case occupies and
where an actor stands. The slots are the ones the author already drew: the tool
reads the existing positions, treats them as the layout's grammar, and searches
for a permutation of the nodes over those same slots that draws no crossings.
So the result is recognisably the same diagram — same frames, same columns,
same spacing — with the wires untangled.

Two consequences worth stating plainly:

  * A use case never leaves the group frame it was drawn in. Moving "Classify
    Reply Intent" out of the Automatic AI Pipeline frame would be a claim about
    the system, not a layout tweak.
  * Actors move freely in the margins outside the system boundary, because an
    actor's position carries no meaning in UML — only its associations do, and
    those are what we are trying to make readable.

Writes nothing unless it finds a strictly better layout, and reports what it
found either way.
"""

from __future__ import annotations

import itertools
import math
import random
import re
import sys
import time
import xml.etree.ElementTree as ET

EPS = 1e-9

# Search effort. Restarts trade wall-clock for the chance of escaping a local
# minimum; the step budget is per restart.
RESTARTS = 60
STEPS = 6000

# A crossing is the defect we are here to remove; a line drawn through an
# unrelated box is nearly as unreadable, so it is weighted in the same order of
# magnitude. Edge length only breaks ties between otherwise-clean layouts.
W_CROSS = 1000.0
W_THROUGH = 400.0
W_LENGTH = 0.001


# --------------------------------------------------------------------------
# geometry
# --------------------------------------------------------------------------

def orient(ax, ay, bx, by, cx, cy):
    return (bx - ax) * (cy - ay) - (by - ay) * (cx - ax)


def on_seg(ax, ay, bx, by, px, py):
    return (min(ax, bx) - EPS <= px <= max(ax, bx) + EPS
            and min(ay, by) - EPS <= py <= max(ay, by) + EPS)


def crosses(s1, s2):
    """Proper crossing or collinear overlap; shared endpoints do not count."""
    ax, ay, bx, by = s1
    cx, cy, dx, dy = s2
    if {(ax, ay), (bx, by)} & {(cx, cy), (dx, dy)}:
        return False
    d1 = orient(cx, cy, dx, dy, ax, ay)
    d2 = orient(cx, cy, dx, dy, bx, by)
    d3 = orient(ax, ay, bx, by, cx, cy)
    d4 = orient(ax, ay, bx, by, dx, dy)
    if abs(d1) < EPS and abs(d2) < EPS:
        return (on_seg(ax, ay, bx, by, cx, cy) or on_seg(ax, ay, bx, by, dx, dy)
                or on_seg(cx, cy, dx, dy, ax, ay))
    return ((d1 > 0) != (d2 > 0)) and ((d3 > 0) != (d4 > 0))


def hits_box(seg, box, pad=6.0):
    x, y, w, h = box
    x1, y1, x2, y2 = x - pad, y - pad, x + w + pad, y + h + pad
    ax, ay, bx, by = seg
    if x1 < ax < x2 and y1 < ay < y2:
        return True
    if x1 < bx < x2 and y1 < by < y2:
        return True
    sides = [(x1, y1, x2, y1), (x2, y1, x2, y2), (x2, y2, x1, y2), (x1, y2, x1, y1)]
    return any(crosses(seg, s) for s in sides)


# --------------------------------------------------------------------------
# model
# --------------------------------------------------------------------------

class Diagram:
    def __init__(self, path):
        self.path = path
        self.text = open(path, encoding="utf-8").read()
        root = ET.fromstring(self.text)

        self.size = {}     # id -> (w, h)
        self.pos0 = {}     # id -> (x, y) as drawn
        self.style = {}
        self.label = {}
        self.edges = []    # (src, tgt)

        for cell in root.iter("mxCell"):
            cid = cell.get("id")
            geo = cell.find("mxGeometry")
            st = cell.get("style", "") or ""
            if cell.get("vertex") == "1" and geo is not None:
                self.size[cid] = (float(geo.get("width", 0)), float(geo.get("height", 0)))
                self.pos0[cid] = (float(geo.get("x", 0)), float(geo.get("y", 0)))
                self.style[cid] = st
                self.label[cid] = re.sub(r"<[^>]+>", "", cell.get("value", "") or "").strip()
            elif cell.get("edge") == "1" and cell.get("source") and cell.get("target"):
                self.edges.append((cell.get("source"), cell.get("target")))

        self._classify()

    def _classify(self):
        """Split the vertices by the role their style already declares."""
        self.frames = []      # group frames + system boundary: never hit-tested
        self.usecases = []
        self.actors = []
        self.fixed = []       # notes and anything else: kept where it is

        for cid, st in self.style.items():
            w, h = self.size[cid]
            if "ellipse" in st:
                self.usecases.append(cid)
            elif "umlActor" in st:
                self.actors.append(cid)
            elif "shape=note" in st:
                self.fixed.append(cid)
            elif "dashed=1" in st or w * h > 120000:
                self.frames.append(cid)
            elif "&laquo;system&raquo;" in st or "laquo;system" in (self.label[cid] or ""):
                self.actors.append(cid)
            else:
                # A plain bordered rectangle carrying a «system» stereotype is an
                # actor in every diagram in this set; anything else stays put.
                (self.actors if w <= 200 and h <= 120 else self.fixed).append(cid)

        # The outermost frame is the system boundary; the rest are groups.
        self.frames.sort(key=lambda c: -self.size[c][0] * self.size[c][1])
        self.boundary = self.frames[0] if self.frames else None
        self.groups = self.frames[1:]

        # Which group frame did the author draw each use case inside?
        self.group_of = {}
        for uc in self.usecases:
            x, y = self.pos0[uc]
            w, h = self.size[uc]
            cx, cy = x + w / 2, y + h / 2
            # By centre, not by containment: several frames were drawn a few
            # pixels too short for the last ellipse in them, and reading that
            # as "belongs to no group" would let the tool move it anywhere.
            for g in self.groups:
                gx, gy = self.pos0[g]
                gw, gh = self.size[g]
                if gx <= cx <= gx + gw and gy <= cy <= gy + gh:
                    self.group_of[uc] = g
                    break
            else:
                self.group_of[uc] = None

        # Slots = the positions already on the page, pooled per group so a use
        # case can only land where a use case of its own group already sat.
        self.slots = {}
        for uc in self.usecases:
            self.slots.setdefault(self.group_of[uc], []).append(self.pos0[uc])

        # Where an actor may stand: the margins outside the system boundary.
        self.actor_spots = self._margin_spots()

    def _margin_spots(self, step=20):
        if self.boundary is None:
            return []
        bx, by = self.pos0[self.boundary]
        bw, bh = self.size[self.boundary]
        spots = []
        for y in range(40, int(by + bh) + 40, step):
            for x in list(range(40, int(bx) - 60, step)) + \
                     list(range(int(bx + bw) + 40, int(bx + bw) + 300, step)):
                spots.append((float(x), float(y)))
        return spots


# --------------------------------------------------------------------------
# scoring
# --------------------------------------------------------------------------

def score(dg, pos):
    def centre(cid):
        x, y = pos[cid]
        w, h = dg.size[cid]
        return x + w / 2, y + h / 2

    segs = []
    for s, t in dg.edges:
        if s not in pos or t not in pos:
            continue
        ax, ay = centre(s)
        bx, by = centre(t)
        segs.append(((s, t), (ax, ay, bx, by)))

    n_cross = 0
    for (e1, s1), (e2, s2) in itertools.combinations(segs, 2):
        if crosses(s1, s2):
            n_cross += 1

    solid = [c for c in dg.usecases + dg.actors + dg.fixed]
    n_through = 0
    for (s, t), seg in segs:
        for cid in solid:
            if cid in (s, t):
                continue
            x, y = pos[cid]
            w, h = dg.size[cid]
            if hits_box(seg, (x, y, w, h)):
                n_through += 1

    # Nodes must not sit on top of each other, and actors must stay clear of
    # every frame so an association is never drawn out of a box it is not in.
    n_overlap = 0
    movable = dg.usecases + dg.actors
    for a, b in itertools.combinations(movable, 2):
        ax, ay = pos[a]
        aw, ah = dg.size[a]
        bx, by = pos[b]
        bw, bh = dg.size[b]
        if ax < bx + bw and bx < ax + aw and ay < by + bh and by < ay + ah:
            n_overlap += 1

    length = sum(math.dist((s[0], s[1]), (s[2], s[3])) for _, s in segs)
    return (W_CROSS * (n_cross + n_overlap) + W_THROUGH * n_through
            + W_LENGTH * length), n_cross, n_through, n_overlap


# --------------------------------------------------------------------------
# search
# --------------------------------------------------------------------------

def initial(dg, rng):
    pos = dict(dg.pos0)
    for g, slots in dg.slots.items():
        members = [u for u in dg.usecases if dg.group_of[u] == g]
        order = members[:]
        rng.shuffle(order)
        for uc, slot in zip(order, slots):
            pos[uc] = slot
    for a in dg.actors:
        pos[a] = rng.choice(dg.actor_spots)
    return pos


def anneal(dg, rng, steps, deadline=None):
    """Returns (rank, positions) where rank is (crossings, through, length).

    Ranked lexicographically rather than by the blended cost: crossings are the
    defect we were asked to remove, and no number of shorter edges makes a
    crossing acceptable.
    """
    members_of = {g: [u for u in dg.usecases if dg.group_of[u] == g] for g in dg.slots}
    groups = [g for g, m in members_of.items() if len(m) > 1]

    pos = initial(dg, rng)
    cost, c, t, o = score(dg, pos)
    cur = cost
    best_rank = (c + o, t, cost)
    best_pos = dict(pos)

    for i in range(steps):
        if deadline is not None and i % 200 == 0 and time.time() > deadline:
            break
        temp = max(0.01, 1.0 - i / steps) * 400.0

        if dg.actors and rng.random() < 0.45:
            node = rng.choice(dg.actors)
            old = pos[node]
            pos[node] = rng.choice(dg.actor_spots)
            undo = [(node, old)]
        else:
            g = rng.choice(groups)
            a, b = rng.sample(members_of[g], 2)
            pos[a], pos[b] = pos[b], pos[a]
            undo = [(a, pos[b]), (b, pos[a])]

        cost, c, t, o = score(dg, pos)
        if cost <= cur or rng.random() < math.exp((cur - cost) / temp):
            cur = cost
            rank = (c + o, t, cost)
            if rank < best_rank:
                best_rank, best_pos = rank, dict(pos)
                if c == 0 and o == 0 and t == 0:
                    break
        else:
            for node, val in undo:
                pos[node] = val

    return best_rank, best_pos


# --------------------------------------------------------------------------
# writing back
# --------------------------------------------------------------------------

def rewrite(dg, pos):
    """Substitute x/y in place, leaving every other byte of the file alone."""
    text = dg.text
    for cid, (x, y) in pos.items():
        if cid not in dg.pos0 or (x, y) == dg.pos0[cid]:
            continue
        pattern = re.compile(
            r'(<mxCell id="' + re.escape(cid) + r'"(?:(?!</mxCell>).)*?'
            r'<mxGeometry )x="[-\d.]+" y="[-\d.]+"',
            re.S)
        new, n = pattern.subn(lambda m: f'{m.group(1)}x="{x:g}" y="{y:g}"', text, count=1)
        if n != 1:
            raise SystemExit(f"khong tim thay geometry cua {cid} — dung lai, khong ghi gi")
        text = new
    return text


def process(path, budget=900.0):
    dg = Diagram(path)
    cost0, c0, t0, o0 = score(dg, dg.pos0)
    print(f"{path}: ban goc — cat nhau={c0}, xuyen o={t0}, chong o={o0}", flush=True)
    print(f"  {len(dg.usecases)} use case trong {len(dg.groups)} khung, "
          f"{len(dg.actors)} actor, {len(dg.edges)} canh", flush=True)

    deadline = time.time() + budget
    best_rank, best_pos = (c0 + o0, t0, cost0), dict(dg.pos0)
    for r in range(RESTARTS):
        if time.time() > deadline:
            print("  het thoi gian", flush=True)
            break
        rank, p = anneal(dg, random.Random(1000 + r), STEPS, deadline)
        if rank < best_rank:
            best_rank, best_pos = rank, p
            print(f"  lan {r}: cat nhau={rank[0]}, xuyen o={rank[1]}", flush=True)
            if rank[0] == 0 and rank[1] == 0:
                break

    _, c, t, o = score(dg, best_pos)
    if (c + o, t) >= (c0 + o0, t0):
        print("  khong tim duoc bo cuc tot hon — khong ghi gi", flush=True)
        return False

    open(path, "w", encoding="utf-8").write(rewrite(dg, best_pos))
    print(f"  DA GHI: cat nhau={c}, xuyen o={t}, chong o={o}", flush=True)
    return True


if __name__ == "__main__":
    for p in sys.argv[1:]:
        process(p)
