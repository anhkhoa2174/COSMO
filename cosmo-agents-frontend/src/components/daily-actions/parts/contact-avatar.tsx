'use client';

/** Gradient-colored initials avatar for contacts (T035). */

const gradientPalette = [
  'linear-gradient(135deg, #3b82f6, #1d4ed8)',
  'linear-gradient(135deg, #f59e0b, #ea580c)',
  'linear-gradient(135deg, #ef4444, #dc2626)',
  'linear-gradient(135deg, #22c55e, #16a34a)',
  'linear-gradient(135deg, #8b5cf6, #6d28d9)',
  'linear-gradient(135deg, #ec4899, #db2777)',
  'linear-gradient(135deg, #14b8a6, #0d9488)',
  'linear-gradient(135deg, #f97316, #c2410c)',
];

function getGradient(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return gradientPalette[Math.abs(hash) % gradientPalette.length];
}

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

interface ContactAvatarProps {
  name: string;
  size?: number;
  onClick?: () => void;
}

export function ContactAvatar({
  name,
  size = 40,
  onClick,
}: ContactAvatarProps) {
  const initials = getInitials(name);
  const gradient = getGradient(name);
  const fontSize = size <= 32 ? '0.68rem' : size <= 40 ? '0.78rem' : '1rem';
  const borderRadius = size <= 32 ? 8 : size <= 40 ? 10 : 14;

  return (
    <div
      onClick={onClick}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      className="flex shrink-0 items-center justify-center font-bold text-white"
      style={{
        width: size,
        height: size,
        borderRadius,
        background: gradient,
        fontSize,
        cursor: onClick ? 'pointer' : undefined,
      }}
    >
      {initials}
    </div>
  );
}
