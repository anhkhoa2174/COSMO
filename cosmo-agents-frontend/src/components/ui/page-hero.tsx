import React from 'react';
import { cn } from '@/lib/utils';

export type HeroAccent = 'violet' | 'rose' | 'blue' | 'emerald' | 'amber';

const ACCENT: Record<
  HeroAccent,
  { surface: string; tile: string; art: string; orb: string }
> = {
  violet: {
    surface: 'from-violet-50 via-white to-indigo-50',
    tile: 'bg-gradient-to-br from-violet-500 to-indigo-600',
    art: 'text-violet-400',
    orb: 'bg-violet-400/25',
  },
  rose: {
    surface: 'from-rose-50 via-white to-pink-50',
    tile: 'bg-gradient-to-br from-rose-500 to-pink-600',
    art: 'text-rose-400',
    orb: 'bg-rose-400/25',
  },
  blue: {
    surface: 'from-sky-50 via-white to-indigo-50',
    tile: 'bg-gradient-to-br from-sky-500 to-blue-600',
    art: 'text-sky-400',
    orb: 'bg-sky-400/25',
  },
  emerald: {
    surface: 'from-emerald-50 via-white to-teal-50',
    tile: 'bg-gradient-to-br from-emerald-500 to-teal-600',
    art: 'text-emerald-400',
    orb: 'bg-emerald-400/25',
  },
  amber: {
    surface: 'from-amber-50 via-white to-orange-50',
    tile: 'bg-gradient-to-br from-amber-500 to-orange-600',
    art: 'text-amber-400',
    orb: 'bg-amber-400/25',
  },
};

/**
 * Hand-placed so the figure is byte-identical between server and client —
 * a random layout would flash a different constellation on hydration.
 */
const NODES: [number, number, number][] = [
  [40, 62, 1.6],
  [96, 28, 1.1],
  [148, 74, 1.9],
  [210, 34, 1.3],
  [246, 96, 1.5],
  [300, 52, 1.1],
  [354, 88, 1.7],
  [402, 30, 1.2],
  [456, 70, 1.5],
  [512, 40, 1.9],
  [560, 92, 1.2],
  [614, 46, 1.6],
  [668, 86, 1.3],
  [720, 34, 1.5],
  [774, 78, 1.8],
  [826, 44, 1.1],
  [880, 90, 1.6],
  [934, 38, 1.4],
  [986, 74, 1.2],
  [1040, 50, 1.7],
  [70, 118, 1.2],
  [180, 132, 1.5],
  [286, 122, 1.1],
  [392, 138, 1.6],
  [498, 118, 1.3],
  [604, 134, 1.5],
  [710, 120, 1.2],
  [816, 136, 1.7],
  [922, 116, 1.3],
  [1028, 130, 1.5],
];

const EDGES: [number, number][] = [
  [0, 1],
  [1, 2],
  [2, 3],
  [3, 4],
  [4, 5],
  [5, 6],
  [6, 7],
  [7, 8],
  [8, 9],
  [9, 10],
  [10, 11],
  [11, 12],
  [12, 13],
  [13, 14],
  [14, 15],
  [15, 16],
  [16, 17],
  [17, 18],
  [18, 19],
  [0, 20],
  [20, 21],
  [21, 22],
  [22, 23],
  [23, 24],
  [24, 25],
  [25, 26],
  [26, 27],
  [27, 28],
  [28, 29],
  [2, 21],
  [5, 22],
  [8, 24],
  [11, 25],
  [14, 27],
  [17, 28],
];

function Constellation({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 1080 170"
      preserveAspectRatio="xMidYMid slice"
      aria-hidden="true"
      className={cn('absolute inset-0 h-full w-full', className)}
    >
      <g stroke="currentColor" strokeWidth="0.6" opacity="0.35">
        {EDGES.map(([a, b]) => (
          <line
            key={`${a}-${b}`}
            x1={NODES[a][0]}
            y1={NODES[a][1]}
            x2={NODES[b][0]}
            y2={NODES[b][1]}
          />
        ))}
      </g>
      <g fill="currentColor" opacity="0.6">
        {NODES.map(([x, y, r], i) => (
          <circle key={i} cx={x} cy={y} r={r} />
        ))}
      </g>
    </svg>
  );
}

export function PageHero({
  icon: Icon,
  eyebrow,
  title,
  titleAdornment,
  description,
  accent = 'violet',
  actions,
  children,
  className,
}: {
  icon?: React.ComponentType<{ className?: string }>;
  /** Small uppercase label above the title. */
  eyebrow?: string;
  title: React.ReactNode;
  /** Rendered inline after the title, e.g. a "1 contacts" count chip. */
  titleAdornment?: React.ReactNode;
  description?: React.ReactNode;
  accent?: HeroAccent;
  /** Right-hand slot: a control, stat card or segmented filter. */
  actions?: React.ReactNode;
  /** Rendered below the description, e.g. the outreach stage filter chips. */
  children?: React.ReactNode;
  className?: string;
}) {
  const tone = ACCENT[accent];

  return (
    <section
      className={cn(
        'relative isolate overflow-hidden rounded-2xl border bg-gradient-to-r',
        tone.surface,
        className
      )}
    >
      <Constellation className={tone.art} />
      <div
        className={cn(
          'pointer-events-none absolute -right-10 top-1/2 size-40 -translate-y-1/2 rounded-full blur-2xl',
          tone.orb
        )}
      />
      <div
        className={cn(
          'pointer-events-none absolute right-16 top-1/2 size-24 -translate-y-1/2 rounded-full opacity-70 blur-md',
          tone.orb
        )}
      />

      <div className="relative flex flex-col gap-5 p-6 md:flex-row md:items-center md:justify-between md:gap-8 md:p-8">
        <div className="flex min-w-0 items-start gap-4">
          {Icon && (
            <span
              className={cn(
                'grid size-14 shrink-0 place-items-center rounded-2xl text-white shadow-sm',
                tone.tile
              )}
            >
              <Icon className="size-7" />
            </span>
          )}
          <div className="min-w-0 space-y-1.5">
            {eyebrow && (
              <p className="text-[0.7rem] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                {eyebrow}
              </p>
            )}
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="text-3xl font-bold tracking-tight text-foreground md:text-[2.1rem]">
                {title}
              </h1>
              {titleAdornment}
            </div>
            {description && (
              <p className="max-w-2xl text-[0.95rem] leading-relaxed text-muted-foreground">
                {description}
              </p>
            )}
            {children}
          </div>
        </div>

        {actions && <div className="shrink-0">{actions}</div>}
      </div>
    </section>
  );
}
