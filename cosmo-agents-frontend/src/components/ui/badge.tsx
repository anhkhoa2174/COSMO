import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const badgeVariants = cva(
  'inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
  {
    variants: {
      variant: {
        default:
          'border-transparent bg-primary text-primary-foreground shadow hover:bg-primary/80',
        secondary:
          'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        destructive:
          'border-transparent bg-destructive text-destructive-foreground shadow hover:bg-destructive/80',
        'destructive-secondary':
          'border-transparent bg-destructive-secondary text-destructive hover:bg-destructive-secondary/80',
        outline: 'text-foreground',
        success:
          'border-transparent bg-success text-success-foreground shadow hover:bg-success/80',
        'success-secondary':
          'border-transparent bg-success-secondary text-success hover:bg-success-secondary/80',
        warning:
          'border-transparent bg-warning text-warning-foreground shadow hover:bg-warning/80',
        'warning-secondary':
          'border-transparent bg-warning-secondary text-warning hover:bg-warning-secondary/80',
        info: 'border-transparent bg-info text-info-foreground shadow hover:bg-info/80',
        'info-secondary':
          'border-transparent bg-info-secondary text-info hover:bg-info-secondary/80',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
);

export interface BadgeProps
  extends
    React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
