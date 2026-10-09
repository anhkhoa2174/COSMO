"""Untangle a use case diagram by re-laying the group frames, not the notation.

This is the wider search that reslot_usecase.py could not reach. That tool held
every group frame exactly where the author drew it and only permuted use cases
between the slots already on the page, which left three crossings that are
geometrically unavoidable at that arrangement: Prospect associates with two use
cases that sit in different frames on the same side as BD's eleven, and two fans
leaving the same margin cross whenever one's targets straddle the other's.

So here the frames move too. What still may not change is anything that carries
meaning:

  * every ellipse, actor, frame and note keeps its style, its label and its size
  * every association keeps its arrow, its dash pattern and its «include» /
    «extend» stereotype
  * a use case never leaves the frame the author drew it in
  * frames keep their captions, their dashed border and their grid discipline —
    the same column pitch and row pitch the original used

What may change is where a frame sits, how many columns of use cases it is laid
out in, and therefore its width and height. A dashed grouping frame's position
and size say nothing about the system; they are a reading aid, and reflowing one
to fit its own contents is the same kind of edit as moving an actor.

Frames are packed into columns rather than placed freely. Free placement admits
layouts that are technically crossing-free and visually incoherent; columns keep
the result looking like the diagram it started as.
"""

from __future__ import annotations

import itertools
import math
import random
import re
import sys
import time

from reslot_usecase import Diagram, crosses, hits_box, rewrite

# Frame interior, matching the pitch the author already used: use cases 150 wide
# sat at x=330 and x=540 (a 60 px gutter), 60 tall at y=70 and y=155 (25 px).
PAD_X, PAD_TOP, PAD_BOTTOM = 30, 30, 20
GAP_X, GAP_Y = 60, 25
FRAME_GAP, COL_GAP = 30, 60
# How far outside the boundary an actor stands.
ACTOR_GAP = 70
SYS_MARGIN = 20

# Weighting these against each other is the whole design of the search, and a
# first attempt got it wrong in a way worth recording. With W_LENGTH at 0.001 a
# 27,000-pixel layout cost 27 points against 1,000 for a single crossing, so the
# optimiser bought crossing-freedom by flinging an actor 600 px past the edge of
# the diagram and dragging eleven associations across the whole canvas. It was
# crossing-free and unreadable — the failure this module's docstring warns about,
# produced by this module.
#
# Length now carries real weight: a sprawling layout costs several crossings, so
# the search cannot buy tidiness with distance. And an edge drawn through an
# unrelated ellipse is weighted above a crossing, because to a reader it is
# worse — two lines meeting are legible, a line vanishing into a shape is not.
W_CROSS, W_THROUGH, W_LENGTH = 1000.0, 1200.0, 0.20
# An actor drawn inside the system boundary is not an ugly layout, it is a false
# statement: the boundary is what separates the system from the things outside
# it that use it. Weighted as a hard constraint so no number of tidy edges can
# buy one.
W_STRAY = 100000.0

RESTARTS = 60
STEPS = 8000


class Layout:
    """A candidate arrangement: frame → (column, rank, grid columns)."""

    def __init__(self, dg: Diagram, rng: random.Random):
        self.dg = dg
        self.groups = list(dg.groups)
        self.members = {g: [u for u in dg.usecases if dg.group_of[u] == g]
                        for g in self.groups}
        # Cell size per frame: the largest member, so every slot fits any member.
        self.cell = {}
        for g, ms in self.members.items():
            self.cell[g] = (max(dg.size[m][0] for m in ms),
                            max(dg.size[m][1] for m in ms))

        self.col = {g: rng.randrange(3) for g in self.groups}
        self.rank = {g: rng.random() for g in self.groups}
        self.ncols = {g: rng.choice(self._shapes(g)) for g in self.groups}
        self.order = {g: rng.sample(ms, len(ms)) for g, ms in self.members.items()}
        # An actor is placed by choosing which edge of the system boundary it
        # stands beside and how far along that edge, not by choosing a point on
        # the page. Free coordinates were the previous design and they produced
        # actors stranded hundreds of pixels off the diagram, trailing
        # associations across empty canvas: the length penalty was the only
        # thing pulling them back and it lost to the crossing penalty every
        # time. In a use case diagram actors hug the boundary. That is a
        # property of the notation, not something a search should have to
        # rediscover, so it is built into the representation instead.
        self.actor_side = {a: rng.randrange(4) for a in dg.actors + dg.fixed}
        self.actor_off = {a: rng.random() for a in dg.actors + dg.fixed}

    def _shapes(self, g):
        n = len(self.members[g])
        return [c for c in (1, 2, 3, 4) if c <= n] or [1]

    def frame_size(self, g):
        cw, ch = self.cell[g]
        cols = self.ncols[g]
        rows = math.ceil(len(self.members[g]) / cols)
        return (2 * PAD_X + cols * cw + (cols - 1) * GAP_X,
                PAD_TOP + rows * ch + (rows - 1) * GAP_Y + PAD_BOTTOM)

    def positions(self):
        """Resolve the arrangement into absolute coordinates."""
        dg = self.dg
        cols = {}
        for g in self.groups:
            cols.setdefault(self.col[g], []).append(g)
        for c in cols:
            cols[c].sort(key=lambda g: self.rank[g])

        sizes = {g: self.frame_size(g) for g in self.groups}
        sx, sy = dg.pos0[dg.boundary]

        pos = {}
        x = sx + SYS_MARGIN
        max_bottom = sy + SYS_MARGIN
        for c in sorted(cols):
            width = max(sizes[g][0] for g in cols[c])
            y = sy + SYS_MARGIN
            for g in cols[c]:
                fw, fh = sizes[g]
                pos[g] = (x, y)
                cw, ch = self.cell[g]
                ncol = self.ncols[g]
                for i, m in enumerate(self.order[g]):
                    mw, mh = dg.size[m]
                    r, k = divmod(i, ncol)
                    # Centre a member in its cell so mixed widths stay aligned.
                    pos[m] = (x + PAD_X + k * (cw + GAP_X) + (cw - mw) / 2,
                              y + PAD_TOP + r * (ch + GAP_Y) + (ch - mh) / 2)
                y += fh + FRAME_GAP
            max_bottom = max(max_bottom, y - FRAME_GAP)
            x += width + COL_GAP

        # The system boundary is redrawn around whatever the frames now need.
        bw = (x - COL_GAP) - sx + SYS_MARGIN
        bh = max_bottom - sy + SYS_MARGIN
        pos[dg.boundary] = (sx, sy)
        self.boundary_size = (bw, bh)

        for a in self.actor_side:
            aw, ah = dg.size[a]
            side, t = self.actor_side[a], self.actor_off[a]
            if side == 0:                                  # left
                pos[a] = (sx - ACTOR_GAP - aw, sy + t * max(0.0, bh - ah))
            elif side == 1:                                # right
                pos[a] = (sx + bw + ACTOR_GAP, sy + t * max(0.0, bh - ah))
            elif side == 2:                                # top
                pos[a] = (sx + t * max(0.0, bw - aw), sy - ACTOR_GAP - ah)
            else:                                          # bottom
                pos[a] = (sx + t * max(0.0, bw - aw), sy + bh + ACTOR_GAP)
        return pos

    def clone(self):
        c = Layout.__new__(Layout)
        c.dg, c.groups, c.members, c.cell = self.dg, self.groups, self.members, self.cell
        c.col, c.rank, c.ncols = dict(self.col), dict(self.rank), dict(self.ncols)
        c.order = {g: list(v) for g, v in self.order.items()}
        c.actor_side = dict(self.actor_side)
        c.actor_off = dict(self.actor_off)
        return c

    def mutate(self, rng):
        r = rng.random()
        if r < 0.35 and self.dg.actors:
            a = rng.choice(list(self.actor_side))
            if rng.random() < 0.3:
                self.actor_side[a] = rng.randrange(4)
            else:
                # A nudge along the current edge, so a good side is refined
                # rather than thrown away on every mutation.
                self.actor_off[a] = min(1.0, max(0.0,
                    self.actor_off[a] + rng.uniform(-0.25, 0.25)))
        elif r < 0.55:
            g = rng.choice([g for g in self.groups if len(self.members[g]) > 1])
            i, j = rng.sample(range(len(self.order[g])), 2)
            self.order[g][i], self.order[g][j] = self.order[g][j], self.order[g][i]
        elif r < 0.75:
            g = rng.choice(self.groups)
            self.col[g] = rng.randrange(3)
            self.rank[g] = rng.random()
        else:
            g = rng.choice(self.groups)
            self.ncols[g] = rng.choice(self._shapes(g))


def evaluate(dg, layout):
    pos = layout.positions()
    bw, bh = layout.boundary_size

    # Actors and the note belong in the margin, never on top of the frames.
    bx, by = pos[dg.boundary]
    strays = 0
    for a in list(dg.actors) + list(dg.fixed):
        ax, ay = pos[a]
        aw, ah = dg.size[a]
        if ax < bx + bw and bx < ax + aw and ay < by + bh and by < ay + ah:
            strays += 1

    centre = lambda c: (pos[c][0] + dg.size[c][0] / 2, pos[c][1] + dg.size[c][1] / 2)
    segs = [((s, t), (*centre(s), *centre(t))) for s, t in dg.edges]

    n_cross = sum(1 for (e1, s1), (e2, s2) in itertools.combinations(segs, 2)
                  if crosses(s1, s2))

    solid = dg.usecases + dg.actors + dg.fixed
    n_through = 0
    for (s, t), seg in segs:
        for cid in solid:
            if cid in (s, t):
                continue
            x, y = pos[cid]
            w, h = dg.size[cid]
            if hits_box(seg, (x, y, w, h)):
                n_through += 1

    overlap = 0
    for a, b in itertools.combinations(list(dg.actors) + list(dg.fixed), 2):
        ax, ay = pos[a]; aw, ah = dg.size[a]
        bx2, by2 = pos[b]; bw2, bh2 = dg.size[b]
        if ax < bx2 + bw2 and bx2 < ax + aw and ay < by2 + bh2 and by2 < ay + ah:
            overlap += 1

    length = sum(math.dist(s[:2], s[2:]) for _, s in segs)
    cost = (W_STRAY * strays + W_CROSS * (n_cross + overlap)
            + W_THROUGH * n_through + W_LENGTH * length)
    return cost, n_cross + overlap + strays * 100, n_through, pos, (bw, bh)


def search(dg, rng, steps, deadline):
    cur = Layout(dg, rng)
    cost, c, t, pos, bsize = evaluate(dg, cur)
    best = (c, t, cost, pos, bsize)
    cur_cost = cost

    for i in range(steps):
        if deadline and i % 100 == 0 and time.time() > deadline:
            break
        temp = max(0.01, 1.0 - i / steps) * 500.0
        cand = cur.clone()
        cand.mutate(rng)
        cost, c, t, pos, bsize = evaluate(dg, cand)
        if cost <= cur_cost or rng.random() < math.exp((cur_cost - cost) / temp):
            cur, cur_cost = cand, cost
            if (c, t, cost) < best[:3]:
                best = (c, t, cost, pos, bsize)
                if c == 0 and t == 0:
                    break
    return best


def process(path, budget=900.0):
    dg = Diagram(path)
    # Baseline, measured the same way so the comparison is like for like.
    base_pos = dict(dg.pos0)
    centre = lambda c: (base_pos[c][0] + dg.size[c][0] / 2,
                        base_pos[c][1] + dg.size[c][1] / 2)
    segs = [((s, t), (*centre(s), *centre(t))) for s, t in dg.edges]
    c0 = sum(1 for (e1, s1), (e2, s2) in itertools.combinations(segs, 2)
             if crosses(s1, s2))
    print(f"{path}: ban goc cat nhau={c0}", flush=True)

    deadline = time.time() + budget
    best = None
    for r in range(RESTARTS):
        if time.time() > deadline:
            print("  het thoi gian", flush=True)
            break
        res = search(dg, random.Random(7000 + r), STEPS, deadline)
        if best is None or res[:3] < best[:3]:
            best = res
            print(f"  lan {r}: cat nhau={res[0]}, xuyen o={res[1]}", flush=True)
            if res[0] == 0 and res[1] == 0:
                break

    c, t, _, pos, (bw, bh) = best
    if c >= c0:
        print("  khong tot hon — khong ghi gi", flush=True)
        return False

    text = rewrite(dg, pos)
    # The system boundary is the one box whose size follows the layout.
    text = re.sub(
        r'(<mxCell id="' + re.escape(dg.boundary) + r'"(?:(?!</mxCell>).)*?<mxGeometry '
        r'x="[-\d.]+" y="[-\d.]+" )width="[-\d.]+" height="[-\d.]+"',
        lambda m: f'{m.group(1)}width="{bw:g}" height="{bh:g}"', text, count=1, flags=re.S)
    for g in dg.groups:
        # Each frame is redrawn around the members that actually landed in it,
        # rather than kept at whatever size it happened to have.
        gx, gy = pos[g]
        members = [u for u in dg.usecases if dg.group_of[u] == g]
        xs = [pos[m][0] for m in members]
        ys = [pos[m][1] for m in members]
        ws = [pos[m][0] + dg.size[m][0] for m in members]
        hs = [pos[m][1] + dg.size[m][1] for m in members]
        fw = max(ws) - gx + PAD_X
        fh = max(hs) - gy + PAD_BOTTOM
        text = re.sub(
            r'(<mxCell id="' + re.escape(g) + r'"(?:(?!</mxCell>).)*?<mxGeometry '
            r'x="[-\d.]+" y="[-\d.]+" )width="[-\d.]+" height="[-\d.]+"',
            lambda m: f'{m.group(1)}width="{fw:g}" height="{fh:g}"', text, count=1, flags=re.S)

    open(path, "w", encoding="utf-8").write(text)
    print(f"  DA GHI: cat nhau={c}, xuyen o={t}", flush=True)
    return True


if __name__ == "__main__":
    for p in sys.argv[1:]:
        process(p)
