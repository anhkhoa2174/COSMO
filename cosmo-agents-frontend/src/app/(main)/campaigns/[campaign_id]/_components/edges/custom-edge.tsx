import {
  BaseEdge,
  EdgeLabelRenderer,
  getSmoothStepPath,
  useReactFlow,
} from '@xyflow/react';
import { X } from 'lucide-react';
import { useState } from 'react';

/**
 * A gradient wire with a packet travelling along it, so the canvas reads as a
 * live pipeline rather than a static diagram. The delete control only appears
 * on hover — a permanent button on every edge buries the flow itself.
 */
export default function CustomEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  selected,
  pathOptions,
}: any) {
  const { setEdges } = useReactFlow();
  const [hovered, setHovered] = useState(false);

  // Smooth-step, not bezier: the campaign canvas lays nodes out on a grid and
  // the orthogonal routing is what the existing edges already use.
  const [edgePath, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
    borderRadius: pathOptions?.borderRadius ?? 12,
  });

  // Gradients and markers are per-edge: two edges sharing an id would make the
  // browser resolve url(#…) to whichever was painted first.
  const gradientId = `edge-gradient-${id}`;
  const active = hovered || selected;

  return (
    <>
      <defs>
        <linearGradient
          id={gradientId}
          gradientUnits="userSpaceOnUse"
          x1={sourceX}
          y1={sourceY}
          x2={targetX}
          y2={targetY}
        >
          <stop offset="0%" stopColor="#8b5cf6" />
          <stop offset="100%" stopColor="#3b82f6" />
        </linearGradient>
      </defs>

      {/* Widened transparent path — the visible wire is too thin to hover. */}
      <path
        d={edgePath}
        fill="none"
        stroke="transparent"
        strokeWidth={18}
        style={{ pointerEvents: 'stroke', cursor: 'pointer' }}
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      />

      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: `url(#${gradientId})`,
          strokeWidth: active ? 2.5 : 1.75,
          transition: 'stroke-width 150ms ease',
        }}
      />

      {/* Flow direction: a dash pattern crawling from source to target. */}
      <path
        className="motion-reduce:hidden"
        d={edgePath}
        fill="none"
        stroke="#fff"
        strokeWidth={2}
        strokeOpacity={0.55}
        strokeLinecap="round"
        strokeDasharray="5 14"
        style={{ pointerEvents: 'none' }}
      >
        <animate
          attributeName="stroke-dashoffset"
          from="19"
          to="0"
          dur="0.9s"
          repeatCount="indefinite"
        />
      </path>

      {/* The packet itself. */}
      <circle
        className="motion-reduce:hidden"
        r={active ? 4 : 3}
        fill={`url(#${gradientId})`}
        stroke="#fff"
        strokeWidth={1.5}
        style={{ pointerEvents: 'none' }}
      >
        <animateMotion dur="2.2s" repeatCount="indefinite" path={edgePath} />
        <animate
          attributeName="opacity"
          values="0;1;1;0"
          keyTimes="0;0.15;0.85;1"
          dur="2.2s"
          repeatCount="indefinite"
        />
      </circle>

      <EdgeLabelRenderer>
        <div
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            pointerEvents: 'all',
          }}
          className="nodrag nopan"
          onMouseEnter={() => setHovered(true)}
          onMouseLeave={() => setHovered(false)}
        >
          <button
            type="button"
            aria-label="Remove connection"
            onClick={() => setEdges((es) => es.filter((e) => e.id !== id))}
            className={`grid size-6 place-items-center rounded-full border border-violet-200 bg-white text-slate-500 shadow-sm transition-all duration-150 hover:border-red-300 hover:text-red-600 ${
              active ? 'scale-100 opacity-100' : 'scale-75 opacity-0'
            }`}
          >
            <X className="size-3.5" />
          </button>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}
