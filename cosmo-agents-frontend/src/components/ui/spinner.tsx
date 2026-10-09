import React from 'react';
import { cva, VariantProps } from 'class-variance-authority';
import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

const overlayVariants = cva('', {
  variants: {
    withOverlay: {
      true: 'absolute inset-0 z-50 bg-white/70',
      false: '',
    },
  },
  defaultVariants: {
    withOverlay: false,
  },
});

const spinnerVariants = cva('flex-col items-center justify-center', {
  variants: {
    show: {
      true: 'flex',
      false: 'hidden',
    },
  },
  defaultVariants: {
    show: true,
  },
});

const loaderVariants = cva('animate-spin text-primary', {
  variants: {
    size: {
      small: 'size-6',
      medium: 'size-10',
      large: 'size-14',
    },
  },
  defaultVariants: {
    size: 'medium',
  },
});

interface SpinnerContentProps
  extends
    VariantProps<typeof overlayVariants>,
    VariantProps<typeof spinnerVariants>,
    VariantProps<typeof loaderVariants> {
  className?: string;
  label?: string;
}

export function Spinner({
  size,
  show,
  withOverlay,
  className,
  label,
}: SpinnerContentProps) {
  return (
    <span
      className={cn(
        overlayVariants({ withOverlay }),
        spinnerVariants({ show })
      )}
    >
      <Loader2
        className={cn(
          loaderVariants({ size }),
          'text-accent-foreground',
          className
        )}
      />
      {label && <span className="text-sm text-accent-foreground">{label}</span>}
    </span>
  );
}
