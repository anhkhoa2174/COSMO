import React from 'react';
import { Loader2, LucideIcon } from 'lucide-react';
import { Button, ButtonProps } from '@/components/ui/button';

interface MainButtonProps extends ButtonProps {
  text?: string;
  loading?: boolean;
  icon?: LucideIcon;
  rightIcon?: LucideIcon;
}

const MainButton = React.forwardRef<HTMLButtonElement, MainButtonProps>(
  ({ text = '', loading = false, icon: Icon, rightIcon: RightIcon, disabled, ...props }, ref) => {
    return (
      <Button ref={ref} disabled={disabled || loading} {...props}>
        {loading && Icon ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : Icon ? (
          <Icon className="h-4 w-4" />
        ) : null}
        {text}
        {loading && !Icon ?
          <Loader2 className="h-4 w-4 animate-spin" />
        : RightIcon ? (
          <RightIcon className="h-4 w-4" />
        ) : null}
      </Button>
    );
  }
);

MainButton.displayName = 'MainButton';

export { MainButton };
