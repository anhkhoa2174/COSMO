import { cn } from '@/lib/utils';

/**
 * The official COSMO brand mark: a violet sphere crossed by two orbital arcs
 * with a four-point spark at the centre. Vector copy of public/favicon.svg so
 * the same asset renders crisply at every size across the app shell, the
 * landing header, the legal pages and the login screen.
 *
 * The gradient/clip ids are suffixed per instance to stay unique when several
 * marks are rendered on one page.
 */
export function CosmoMark({
  className,
  size = 32,
  id = 'cosmo',
}: {
  className?: string;
  size?: number;
  id?: string;
}) {
  const gradientId = `${id}-orb-gradient`;
  const clipId = `${id}-orb-clip`;

  return (
    <svg
      viewBox="0 0 56 44"
      width={size}
      height={(size * 44) / 56}
      fill="none"
      role="img"
      aria-label="COSMO"
      className={cn('shrink-0', className)}
      xmlns="http://www.w3.org/2000/svg"
    >
      <g clipPath={`url(#${clipId})`}>
        {/* sphere */}
        <path
          d="M27.6833 44C39.8336 44 49.6833 34.1503 49.6833 22C49.6833 9.84974 39.8336 0 27.6833 0C15.5331 0 5.68333 9.84974 5.68333 22C5.68333 34.1503 15.5331 44 27.6833 44Z"
          fill={`url(#${gradientId})`}
        />
        {/* inner orbit */}
        <path
          opacity="0.7"
          d="M27.6833 31.1673C41.8586 31.1673 53.35 27.0633 53.35 22.0007C53.35 16.938 41.8586 12.834 27.6833 12.834C13.508 12.834 2.01667 16.938 2.01667 22.0007C2.01667 27.0633 13.508 31.1673 27.6833 31.1673Z"
          stroke="white"
        />
        {/* outer orbit */}
        <path
          opacity="0.5"
          d="M27.6833 36.6673C42.8712 36.6673 55.1833 30.1008 55.1833 22.0007C55.1833 13.9005 42.8712 7.33398 27.6833 7.33398C12.4955 7.33398 0.183334 13.9005 0.183334 22.0007C0.183334 30.1008 12.4955 36.6673 27.6833 36.6673Z"
          stroke="white"
        />
        {/* spark */}
        <path
          d="M27.6833 18.334L28.7833 20.9007L31.35 22.0007L28.7833 23.1007L27.6833 25.6673L26.5833 23.1007L24.0167 22.0007L26.5833 20.9007L27.6833 18.334Z"
          fill="white"
        />
      </g>
      <defs>
        <linearGradient
          id={gradientId}
          x1="5.68333"
          y1="-0.183333"
          x2="49.6833"
          y2="43.8167"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#1A237E" />
          <stop offset="1" stopColor="#4A148C" />
        </linearGradient>
        <clipPath id={clipId}>
          <rect width="55.3667" height="44" fill="white" />
        </clipPath>
      </defs>
    </svg>
  );
}
