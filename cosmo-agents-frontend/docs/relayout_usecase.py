#!/usr/bin/env python3
"""Re-lay out the use case .drawio diagrams so no edge crosses another edge
and no edge passes through a use case box.

The diagrams were hand-placed, so edges wander across the canvas and cut
through ovals. Every one of these graphs is sparse (edges <= nodes), which
means a layered left-to-right layout can reach zero crossings; the script
searches for such an ordering and refuses to write a file until it verifies
one geometrically.

Usage:  python3 relayout_usecase.py [file.drawio ...]
"""

from __future__ import annotations

import html
import itertools
import random
import re
import sys
from dataclasses import dataclass, field

# ---------------------------------------------------------------- geometry

# Tuned for a figure that sits on a report page: a wide, flat diagram gets
# scaled down to unreadable text once it is fitted to the text width.
COL_W = 230          # horizontal distance between layers
ROW_H = 120          # vertical distance between rows — actor labels sit below
                     # the glyph, so the rows need room for them
NODE_W, NODE_H = 150, 58
ACTOR_W, ACTOR_H = 40, 50
MARGIN_X, MARGIN_Y = 50, 70


@dataclass
class Node:
    id: str
    label: str
    raw_value: str
    style: str
    is_actor: bool
    w: float = NODE_W
    h: float = NODE_H
    layer: int = 0
    order: int = 0
    x: float = 0.0
    y: float = 0.0

    @property
    def cx(self) -> float:
        return self.x + self.w / 2

    @property
    def cy(self) -> float:
        return self.y + self.h / 2

    def box(self) -> tuple[float, float, float, float]:
        return self.x, self.y, self.x + self.w, self.y + self.h


@dataclass
class Edge:
    id: str
    src: str
    tgt: str
    raw_value: str
    style: str


@dataclass
class Diagram:
    path: str
    diagram_attrs: str
    nodes: dict[str, Node] = field(default_factory=dict)
    edges: list[Edge] = field(default_factory=list)
    title: str = ""
    title_style: str = ""


# ---------------------------------------------------------------- parsing

CELL_RE = re.compile(r'<mxCell\b([^>]*?)(/>|>)', re.S)


def _attr(blob: str, name: str) -> str:
    m = re.search(rf'\b{name}="((?:[^"\\]|\\.)*)"', blob)
    return m.group(1) if m else ""


def plain(value: str) -> str:
    text = html.unescape(value)
    text = re.sub(r"<br\s*/?>", " ", text)
    text = re.sub(r"<[^>]+>", "", text)
    return html.unescape(text).replace("\n", " ").strip()


def parse(path: str) -> Diagram:
    src = open(path, encoding="utf-8").read()

    dm = re.search(r"<diagram\b([^>]*)>", src)
    dia = Diagram(path=path, diagram_attrs=dm.group(1) if dm else "")

    for m in CELL_RE.finditer(src):
        blob = m.group(1)
        cid = _attr(blob, "id")
        if cid in ("0", "1", ""):
            continue
        value = _attr(blob, "value")
        style = _attr(blob, "style")

        if 'edge="1"' in blob:
            s, t = _attr(blob, "source"), _attr(blob, "target")
            if s and t:
                dia.edges.append(Edge(cid, s, t, value, style))
            continue

        if 'vertex="1"' not in blob:
            continue

        # The system-boundary rectangle is re-derived from the final layout,
        # and free-floating legends carry no edges, so both are dropped.
        label = plain(value)
        if "swimlane" in style or ("Module" in label and "rounded=0" in style):
            dia.title = label
            dia.title_style = style
            continue
        if "Legend" in label or label.startswith("«") and not _attr(blob, "style"):
            continue

        is_actor = "shape=umlActor" in style or "actor" in cid.lower()
        node = Node(cid, label, value, style, is_actor)
        if is_actor:
            node.w, node.h = ACTOR_W, ACTOR_H
        dia.nodes[cid] = node

    dia.edges = [e for e in dia.edges if e.src in dia.nodes and e.tgt in dia.nodes]

    # Drop anything with no connections. The system-boundary rectangle and the
    # legend are decoration; left in the graph they get a layout slot and other
    # edges then run straight through them.
    linked = {i for e in dia.edges for i in (e.src, e.tgt)}
    dropped: dict[str, str] = {}
    for i in [i for i in dia.nodes if i not in linked]:
        dropped[i] = dia.nodes.pop(i).label

    # The boundary rectangle is always id "sys"; the group labels inside it
    # would otherwise win the title by matching on a keyword.
    if not dia.title:
        dia.title = dropped.get("sys", "")

    return dia


# ---------------------------------------------------------------- layering


def assign_layers(dia: Diagram) -> None:
    """Longest-path layering over the edge direction, with actors pinned.

    Human actors start the flow on the left; system actors (LLM, Gmail,
    Apollo) are pushed one column past everything they touch so their edges
    approach from the right instead of reaching back across the picture.
    """
    ids = list(dia.nodes)
    out: dict[str, list[str]] = {i: [] for i in ids}
    inc: dict[str, list[str]] = {i: [] for i in ids}
    for e in dia.edges:
        out[e.src].append(e.tgt)
        inc[e.tgt].append(e.src)

    system = {
        i for i, n in dia.nodes.items()
        if n.is_actor and ("«" in n.label or "system" in n.label.lower())
    }

    layer = {i: 0 for i in ids}
    for _ in range(len(ids)):
        changed = False
        for e in dia.edges:
            if e.src in system or e.tgt in system:
                continue
            if layer[e.tgt] < layer[e.src] + 1:
                layer[e.tgt] = layer[e.src] + 1
                changed = True
        if not changed:
            break

    for i in system:
        neigh = [layer[j] for j in out[i] + inc[i] if j not in system]
        layer[i] = (max(neigh) + 1) if neigh else max(layer.values(), default=0) + 1

    for i, l in layer.items():
        dia.nodes[i].layer = l


def layer_bfs_from_actors(dia: Diagram) -> None:
    """Alternative layering: distance from the nearest human actor.

    Longest-path layering follows edge direction, but UML «extend» arrows
    point from the refinement back to the base use case. That drags a
    refinement to column 0 and leaves it reaching across the whole diagram to
    a base sitting two columns away — the shape that forces most of the
    crossings. Hop distance from the actor keeps related boxes adjacent.
    """
    adj: dict[str, list[str]] = {i: [] for i in dia.nodes}
    for e in dia.edges:
        adj[e.src].append(e.tgt)
        adj[e.tgt].append(e.src)

    human = [
        i for i, n in dia.nodes.items()
        if n.is_actor and "«" not in n.label and "system" not in n.label.lower()
    ]
    roots = human or [min(dia.nodes, key=lambda i: len(adj[i]))]

    dist = {i: None for i in dia.nodes}
    frontier = list(roots)
    for r in roots:
        dist[r] = 0
    while frontier:
        nxt = []
        for i in frontier:
            for j in adj[i]:
                if dist[j] is None:
                    dist[j] = dist[i] + 1
                    nxt.append(j)
        frontier = nxt

    far = max((v for v in dist.values() if v is not None), default=0)
    for i, v in dist.items():
        dia.nodes[i].layer = v if v is not None else far + 1

    # Two nodes the same number of hops from an actor can still be joined to
    # each other. Such an edge is drawn as a vertical line inside a column and
    # runs over every box between its ends, so push one endpoint outwards
    # until every edge spans at least one column.
    for _ in range(len(dia.nodes)):
        clash = [
            e for e in dia.edges
            if dia.nodes[e.src].layer == dia.nodes[e.tgt].layer
        ]
        if not clash:
            break
        for e in clash:
            a, b = dia.nodes[e.src], dia.nodes[e.tgt]
            # Move whichever end has fewer other connections, to disturb less.
            deg = lambda i: sum(1 for x in dia.edges if i in (x.src, x.tgt))
            (b if deg(e.tgt) <= deg(e.src) else a).layer += 1


# ---------------------------------------------------------------- ordering


def add_dummies(dia: Diagram) -> dict[str, list[str]]:
    """Split every edge that spans more than one layer into unit-length hops.

    Without this a long edge is drawn as one straight line across the columns
    it skips, and nothing stops it from slicing through the boxes parked
    there. Each hop gets an invisible waypoint node that takes part in the
    ordering, so the crossing minimiser can steer the edge around them.

    Returns edge id -> ordered list of dummy node ids, used when emitting the
    polyline.
    """
    chains: dict[str, list[str]] = {}
    extra: list[Edge] = []
    keep: list[Edge] = []

    for e in dia.edges:
        a, b = dia.nodes[e.src], dia.nodes[e.tgt]
        span = abs(a.layer - b.layer)
        if span <= 1:
            keep.append(e)
            continue

        lo, hi = (a, b) if a.layer < b.layer else (b, a)
        ids: list[str] = []
        prev = lo.id
        for l in range(lo.layer + 1, hi.layer):
            did = f"__d_{e.id}_{l}"
            d = Node(did, "", "", "shape=point", is_actor=False, w=1, h=1)
            d.layer = l
            dia.nodes[did] = d
            ids.append(did)
            extra.append(Edge(f"{e.id}__s{l}", prev, did, "", ""))
            prev = did
        extra.append(Edge(f"{e.id}__e", prev, hi.id, "", ""))
        chains[e.id] = ids if a.layer < b.layer else list(reversed(ids))
        keep.append(e)

    dia.edges = keep
    dia.segment_edges = [e for e in keep if e.id not in chains] + extra  # type: ignore[attr-defined]
    return chains


def layers_of(dia: Diagram) -> dict[int, list[str]]:
    out: dict[int, list[str]] = {}
    for i, n in dia.nodes.items():
        out.setdefault(n.layer, []).append(i)
    return out


def count_crossings(dia: Diagram, order: dict[str, int]) -> int:
    """Pairs of edges whose endpoints interleave between adjacent layers."""
    total = 0
    by_span: dict[tuple[int, int], list[tuple[int, int]]] = {}
    for e in getattr(dia, "segment_edges", dia.edges):
        a, b = dia.nodes[e.src], dia.nodes[e.tgt]
        lo, hi = (a, b) if a.layer <= b.layer else (b, a)
        by_span.setdefault((lo.layer, hi.layer), []).append((order[lo.id], order[hi.id]))
    for pairs in by_span.values():
        for (a1, b1), (a2, b2) in itertools.combinations(pairs, 2):
            if (a1 - a2) * (b1 - b2) < 0:
                total += 1
    return total


def order_layers(dia: Diagram, seed: int = 0) -> dict[str, int]:
    """Barycentre sweeps with random restarts, keeping the best ordering."""
    rng = random.Random(seed)
    ls = layers_of(dia)
    adj: dict[str, list[str]] = {i: [] for i in dia.nodes}
    for e in getattr(dia, "segment_edges", dia.edges):
        adj[e.src].append(e.tgt)
        adj[e.tgt].append(e.src)

    best, best_score = None, 10**9
    for attempt in range(60):
        order: dict[str, int] = {}
        for l in sorted(ls):
            ids = ls[l][:]
            if attempt:
                rng.shuffle(ids)
            for k, i in enumerate(ids):
                order[i] = k

        for sweep in range(40):
            for l in sorted(ls, reverse=bool(sweep % 2)):
                ids = ls[l]
                def bary(i: str) -> float:
                    ns = [order[j] for j in adj[i] if dia.nodes[j].layer != l]
                    return sum(ns) / len(ns) if ns else order[i]
                for k, i in enumerate(sorted(ids, key=bary)):
                    order[i] = k

        # Transpose: swapping neighbours often clears the crossings that
        # barycentre sweeps settle into and cannot escape.
        improved = True
        while improved:
            improved = False
            for l in sorted(ls):
                ids = sorted(ls[l], key=lambda i: order[i])
                for k in range(len(ids) - 1):
                    a, b = ids[k], ids[k + 1]
                    before = count_crossings(dia, order)
                    order[a], order[b] = order[b], order[a]
                    if count_crossings(dia, order) < before:
                        ids[k], ids[k + 1] = b, a
                        improved = True
                    else:
                        order[a], order[b] = order[b], order[a]

        score = count_crossings(dia, order)
        if score < best_score:
            best, best_score = dict(order), score
            if score == 0:
                break
    return best  # type: ignore[return-value]


# ---------------------------------------------------------------- placement


def place(dia: Diagram, order: dict[str, int]) -> None:
    """Positions every node, aligning narrow ones to the edge they exit from.

    An actor glyph is much narrower than a use case box. Centred in its
    column, the gap between its own border and the column boundary is still
    column interior, and a steep edge leaving it clips whatever box sits in
    that gap. Pushing the actor flush against the side its edges leave from
    puts the anchor exactly on the column boundary instead.
    """
    ls = layers_of(dia)
    tallest = max((len(v) for v in ls.values()), default=1)
    for l, ids in ls.items():
        ids.sort(key=lambda i: order[i])
        # Centre each column so short columns do not hug the top edge.
        pad = (tallest - len(ids)) * ROW_H / 2
        for k, i in enumerate(ids):
            n = dia.nodes[i]
            left = MARGIN_X + l * COL_W
            if n.w < NODE_W:
                rightward = sum(
                    1 for e in dia.edges
                    if (e.src == i and dia.nodes[e.tgt].layer > l)
                    or (e.tgt == i and dia.nodes[e.src].layer > l)
                )
                leftward = sum(
                    1 for e in dia.edges
                    if (e.src == i and dia.nodes[e.tgt].layer < l)
                    or (e.tgt == i and dia.nodes[e.src].layer < l)
                )
                if rightward and rightward >= leftward:
                    n.x = left + NODE_W - n.w      # flush right
                elif leftward:
                    n.x = left                      # flush left
                else:
                    n.x = left + (NODE_W - n.w) / 2
            else:
                n.x = left + (NODE_W - n.w) / 2
            n.y = MARGIN_Y + pad + k * ROW_H + (NODE_H - n.h) / 2


# ---------------------------------------------------------------- checking


def _seg(a: Node, b: Node) -> tuple[float, float, float, float]:
    return a.cx, a.cy, b.cx, b.cy


def _orient(ax, ay, bx, by, cx, cy) -> float:
    return (bx - ax) * (cy - ay) - (by - ay) * (cx - ax)


def segments_cross(s1, s2) -> bool:
    """Proper crossing only: shared endpoints and touching do not count."""
    ax, ay, bx, by = s1
    cx, cy, dx, dy = s2
    if {(ax, ay), (bx, by)} & {(cx, cy), (dx, dy)}:
        return False
    d1 = _orient(cx, cy, dx, dy, ax, ay)
    d2 = _orient(cx, cy, dx, dy, bx, by)
    d3 = _orient(ax, ay, bx, by, cx, cy)
    d4 = _orient(ax, ay, bx, by, dx, dy)
    return ((d1 > 0) != (d2 > 0)) and ((d3 > 0) != (d4 > 0))


def seg_hits_box(s, box, pad: float = 6.0) -> bool:
    """Does a segment pass through a node's rectangle (not just touch it)?"""
    x1, y1, x2, y2 = box
    x1, y1, x2, y2 = x1 - pad, y1 - pad, x2 + pad, y2 + pad
    ax, ay, bx, by = s

    def inside(px, py):
        return x1 < px < x2 and y1 < py < y2

    if inside(ax, ay) or inside(bx, by):
        return True
    edges = [
        (x1, y1, x2, y1), (x2, y1, x2, y2),
        (x2, y2, x1, y2), (x1, y2, x1, y1),
    ]
    return any(segments_cross(s, e) for e in edges)


def anchor(a: Node, b: Node) -> tuple[float, float]:
    """Where an edge leaves `a` when heading towards `b`.

    Edges attach to the left or right border, never the centre. Anchored at
    the centre, the first stretch of a steep edge is still inside its own
    column and clips whatever box sits directly above or below.
    """
    if b.layer > a.layer:
        return a.x + a.w, a.cy
    if b.layer < a.layer:
        return a.x, a.cy
    return a.cx, a.cy


def polyline(dia: Diagram, e: Edge, chains: dict[str, list[str]]):
    """The points an edge actually passes through, waypoints included."""
    src, tgt = dia.nodes[e.src], dia.nodes[e.tgt]
    way = [dia.nodes[d] for d in chains.get(e.id, [])]
    first_next = way[0] if way else tgt
    last_prev = way[-1] if way else src

    pts = [anchor(src, first_next)]
    for d in way:
        # A waypoint spans its whole column at its own row, so the edge runs
        # flat across the column it is only passing through. Bending at the
        # column centre instead would send a diagonal across that column and
        # straight over whatever sits a row above or below.
        left = MARGIN_X + d.layer * COL_W
        pts.append((left, d.cy))
        pts.append((left + NODE_W, d.cy))
    pts.append(anchor(tgt, last_prev))
    return pts


def verify(dia: Diagram, chains: dict[str, list[str]]) -> tuple[int, int]:
    """Counts real crossings and boxes an edge cuts through.

    Checks the drawn polyline segment by segment; a waypoint is a point, not
    a box, so only genuine nodes count as something to pass through.
    """
    drawn: list[tuple[Edge, tuple[float, float, float, float]]] = []
    for e in dia.edges:
        pts = polyline(dia, e, chains)
        for (x1, y1), (x2, y2) in zip(pts, pts[1:]):
            drawn.append((e, (x1, y1, x2, y2)))

    crossings = 0
    for (e1, s1), (e2, s2) in itertools.combinations(drawn, 2):
        if e1.id == e2.id:
            continue
        if segments_cross(s1, s2):
            crossings += 1

    real = {i: n for i, n in dia.nodes.items() if not i.startswith("__d_")}
    through = 0
    for e, s in drawn:
        for i, n in real.items():
            if i in (e.src, e.tgt):
                continue
            if seg_hits_box(s, n.box()):
                through += 1
    return crossings, through


# ---------------------------------------------------------------- emitting


def emit(dia: Diagram, chains: dict[str, list[str]]) -> str:
    real = [n for n in dia.nodes.values() if not n.id.startswith("__d_")]
    xs = [n.x + n.w for n in real]
    ys = [n.y + n.h for n in real]
    W, H = int(max(xs) + MARGIN_X), int(max(ys) + MARGIN_Y)

    parts = [
        '<mxfile host="app.diagrams.net">',
        f"  <diagram{dia.diagram_attrs}>",
        f'    <mxGraphModel dx="1400" dy="900" grid="0" gridSize="10" guides="1" '
        f'tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" '
        f'pageWidth="{W}" pageHeight="{H}" math="0" shadow="0">',
        "      <root>",
        '        <mxCell id="0" />',
        '        <mxCell id="1" parent="0" />',
    ]

    if dia.title:
        parts.append(
            f'        <mxCell id="title" value="{html.escape(dia.title)}" '
            f'style="text;html=1;align=center;verticalAlign=middle;fontSize=16;fontStyle=1" '
            f'vertex="1" parent="1">'
        )
        parts.append(
            f'          <mxGeometry x="{MARGIN_X}" y="20" width="{W - 2 * MARGIN_X}" '
            f'height="30" as="geometry" />'
        )
        parts.append("        </mxCell>")

    for n in dia.nodes.values():
        if n.id.startswith("__d_"):
            continue  # waypoints exist only for the layout maths
        parts.append(
            f'        <mxCell id="{n.id}" value="{n.raw_value}" style="{n.style}" '
            f'vertex="1" parent="1">'
        )
        parts.append(
            f'          <mxGeometry x="{n.x:g}" y="{n.y:g}" width="{n.w:g}" '
            f'height="{n.h:g}" as="geometry" />'
        )
        parts.append("        </mxCell>")

    for e in dia.edges:
        style = e.style or "endArrow=none;html=1;"
        # Straight edges: the verifier only proves straight lines are clean, so
        # anything curved or waypointed would invalidate the guarantee.
        style = re.sub(r"edgeStyle=[^;]*;?", "", style)
        style = re.sub(r"curved=1;?", "", style)
        if "rounded" not in style:
            style += ";rounded=0"
        # Pin the attachment points, otherwise draw.io re-routes the edge on
        # open and the verified geometry no longer describes what is drawn.
        style = re.sub(r"(exit|entry)[XY]=[^;]*;?", "", style)
        a, b = dia.nodes[e.src], dia.nodes[e.tgt]
        if b.layer > a.layer:
            style += ";exitX=1;exitY=0.5;exitDx=0;exitDy=0;entryX=0;entryY=0.5;entryDx=0;entryDy=0"
        elif b.layer < a.layer:
            style += ";exitX=0;exitY=0.5;exitDx=0;exitDy=0;entryX=1;entryY=0.5;entryDx=0;entryDy=0"
        parts.append(
            f'        <mxCell id="{e.id}" value="{e.raw_value}" style="{style}" '
            f'edge="1" parent="1" source="{e.src}" target="{e.tgt}">'
        )
        way = chains.get(e.id, [])
        if way:
            parts.append('          <mxGeometry relative="1" as="geometry">')
            parts.append('            <Array as="points">')
            for did in way:
                d = dia.nodes[did]
                # Two points per waypoint, matching polyline() exactly. Writing
                # only the centre would make the drawn edge differ from the one
                # the checker approved.
                left = MARGIN_X + d.layer * COL_W
                parts.append(f'              <mxPoint x="{left:g}" y="{d.cy:g}" />')
                parts.append(f'              <mxPoint x="{left + NODE_W:g}" y="{d.cy:g}" />')
            parts.append("            </Array>")
            parts.append("          </mxGeometry>")
        else:
            parts.append('          <mxGeometry relative="1" as="geometry" />')
        parts.append("        </mxCell>")

    parts += ["      </root>", "    </mxGraphModel>", "  </diagram>", "</mxfile>", ""]
    return "\n".join(parts)


# ---------------------------------------------------------------- driver


def process(path: str) -> bool:
    best = None
    best_state = None
    rng = random.Random(12345)

    def attempt(strategy, jitter: int) -> tuple[int, int]:
        """Lay the graph out once and report how clean the result is."""
        nonlocal best, best_state
        dia = parse(path)
        strategy(dia)

        # Nudging a few nodes one column further apart opens up orderings the
        # two fixed strategies cannot reach on their own.
        for _ in range(jitter):
            i = rng.choice(list(dia.nodes))
            dia.nodes[i].layer += 1
        for _ in range(len(dia.nodes)):
            clash = [e for e in dia.edges
                     if dia.nodes[e.src].layer == dia.nodes[e.tgt].layer]
            if not clash:
                break
            for e in clash:
                dia.nodes[e.tgt].layer += 1

        chains = add_dummies(dia)
        local = None
        for seed in range(12):
            order = order_layers(dia, seed)
            place(dia, order)
            score = verify(dia, chains)
            if local is None or score < local[0]:
                local = (score, dict(order))
            if score == (0, 0):
                break
        score, order = local  # type: ignore[misc]
        if best is None or score < best:
            best, best_state = score, (dia, chains, order)
        return score

    for strategy in (layer_bfs_from_actors, assign_layers):
        if attempt(strategy, 0) == (0, 0):
            break
    if best != (0, 0):
        for k in range(120):
            strategy = layer_bfs_from_actors if k % 2 == 0 else assign_layers
            if attempt(strategy, 1 + k % 4) == (0, 0):
                break

    dia, chains, order = best_state  # type: ignore[misc]
    place(dia, order)
    cross, through = verify(dia, chains)

    ok = cross == 0 and through == 0
    mark = "OK " if ok else "XX "
    print(f"  {mark}{path:42s} cắt nhau={cross}  xuyên ô={through}  "
          f"({len([n for n in dia.nodes if not n.startswith(chr(95)*2+chr(100)+chr(95))])} node, {len(dia.edges)} cạnh)")

    if ok:
        open(path, "w", encoding="utf-8").write(emit(dia, chains))
    return ok


if __name__ == "__main__":
    targets = sys.argv[1:]
    allok = all(process(p) for p in targets)
    sys.exit(0 if allok else 1)
