import React from 'react';
import { ArrowLeft } from 'lucide-react';
import { Button, type ButtonProps } from '@/components/ui/button';
import Link from 'next/link';

export interface BackButtonProps extends Omit<ButtonProps, 'asChild'> {
  href: string;
  text?: string;
}

const BackButton = React.forwardRef<HTMLButtonElement, BackButtonProps>(
  ({ href, text = 'Back', ...props }, ref) => {
    return (
      <Link href={href}>
        <Button ref={ref} variant="secondary" {...props}>
          <ArrowLeft className="h-4 w-4" />
          {text}
        </Button>
      </Link>
    );
  }
);

BackButton.displayName = 'BackButton';

export { BackButton };
